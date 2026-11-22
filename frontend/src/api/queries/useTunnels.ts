import { useMutation, useQueries, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { HttpUtil } from '@/utils';
import { parseMsg } from '@/utils/zodValidate';
import { OptionSchema, ServerTunnelSchema, type ServerTunnel } from '@/generated/zod';
import type { EngineRoute } from '@/schemas/engineRoute';

export type { ServerTunnel };
export type TunnelMode = 'reverse' | 'direct';
export type TunnelEngine = 'backpack' | 'rakhsh';
export type TunnelOptionValues = Record<string, unknown>;

const JSON_HEADERS = { headers: { 'Content-Type': 'application/json' } };
export const TUNNELS_KEY = ['tunnels'] as const;
// A tunnel's entry outbound is named tunnel-<name> in the Xray config.
const TUNNEL_OUTBOUND_PREFIX = 'tunnel-';
// The API sends null for an empty tag list; the table and form want an array.
const stringList = z.array(z.string()).nullish();
const TunnelListSchema = z.array(
  ServerTunnelSchema.extend({
    routeInbounds: stringList,
    forwardPorts: stringList,
    fallbackTransports: stringList,
    options: z.record(z.string(), z.unknown()).nullish(),
  }),
);
// Only choice options carry a list of values; the API sends null for the rest.
const TunnelOptionListSchema = z.array(OptionSchema.extend({ choices: stringList }));
export type TunnelOption = z.infer<typeof TunnelOptionListSchema>[number];

export type NewPortalValues =
  | {
      name: string;
      address: string;
      port: number;
      engine: 'backpack';
      transport: string;
      fallbackTransports: string[];
      forwardPorts: string[];
      acceptUdp: boolean;
      routeInbounds: string[];
      mode: TunnelMode;
      options: TunnelOptionValues;
      nodeId?: number;
    }
  | {
      name: string;
      address: string;
      port: number;
      engine: 'rakhsh';
      transport: string;
      forwardPorts: string[];
      routeInbounds: string[];
      mode: TunnelMode;
      options: TunnelOptionValues;
      nodeId?: number;
    };

export interface TunnelEditValues {
  address?: string;
  port?: number;
  transport?: string;
  fallbackTransports?: string[];
  forwardPorts?: string[];
  acceptUdp?: boolean;
  options?: TunnelOptionValues;
  routeInbounds?: string[];
  exitOutbound?: string;
}

export function withLiveTraffic(
  tunnels: ServerTunnel[],
  rows: { tag?: string; up?: number; down?: number }[],
): ServerTunnel[] {
  const byTag = new Map(rows.map((r) => [r.tag, r]));
  return tunnels.map((t) => {
    const row = byTag.get(TUNNEL_OUTBOUND_PREFIX + t.name);
    return row ? { ...t, up: row.up ?? 0, down: row.down ?? 0 } : t;
  });
}

async function fetchTunnels(): Promise<ServerTunnel[]> {
  const msg = await HttpUtil.get('/panel/api/tunnels/list', undefined, { silent: true });
  if (!msg?.success) throw new Error(msg?.msg || 'Failed to fetch tunnels');
  const validated = parseMsg(msg, TunnelListSchema, 'tunnels/list');
  return (validated.obj ?? []).map((t) => ({
    ...t,
    routeInbounds: t.routeInbounds ?? [],
    forwardPorts: t.forwardPorts ?? [],
    fallbackTransports: t.fallbackTransports ?? [],
    options: t.options ?? {},
  }));
}

export function useTunnelsQuery() {
  const query = useQuery({ queryKey: TUNNELS_KEY, queryFn: fetchTunnels });
  return {
    tunnels: query.data ?? [],
    loading: query.isFetching,
    fetched: query.data !== undefined || query.isError,
    fetchError: query.error ? (query.error as Error).message : '',
    refetch: query.refetch,
  };
}

export function useTunnelMutations() {
  const queryClient = useQueryClient();
  const refresh = (msg: { success?: boolean } | null | undefined) => {
    if (msg?.success) queryClient.invalidateQueries({ queryKey: TUNNELS_KEY });
  };

  const create = useMutation({
    mutationFn: (values: NewPortalValues) =>
      HttpUtil.post('/panel/api/tunnels/add', values, JSON_HEADERS),
    onSuccess: refresh,
  });
  const importLink = useMutation({
    mutationFn: (values: { link: string; exitOutbound: string }) =>
      HttpUtil.post('/panel/api/tunnels/import', values, JSON_HEADERS),
    onSuccess: refresh,
  });
  const update = useMutation({
    mutationFn: ({ id, values }: { id: number; values: TunnelEditValues }) =>
      HttpUtil.post(`/panel/api/tunnels/update/${id}`, values, JSON_HEADERS),
    onSuccess: refresh,
  });
  const setEnable = useMutation({
    mutationFn: ({ id, enable }: { id: number; enable: boolean }) =>
      HttpUtil.post(`/panel/api/tunnels/setEnable/${id}`, { enable }),
    onSuccess: refresh,
  });
  const remove = useMutation({
    mutationFn: ({ id, here }: { id: number; here?: boolean }) =>
      HttpUtil.post(`/panel/api/tunnels/del/${id}${here ? '?here=true' : ''}`),
    onSuccess: refresh,
  });

  const installEngine = useMutation({
    mutationFn: ({ engine, route }: { engine: TunnelEngine; route: EngineRoute }) =>
      HttpUtil.post('/panel/api/tunnels/engine/install', { engine, route }, JSON_HEADERS),
    onSuccess: (msg) => {
      if (msg?.success) queryClient.invalidateQueries({ queryKey: TUNNELS_KEY });
    },
  });

  const updateEngine = useMutation({
    mutationFn: ({
      nodeId,
      engine,
      route,
    }: {
      nodeId: number;
      engine: TunnelEngine;
      route: EngineRoute;
    }) =>
      HttpUtil.post(
        '/panel/api/tunnels/engine/update',
        { nodeId, engine, version: '', route },
        JSON_HEADERS,
      ),
    onSuccess: (msg) => {
      if (msg?.success) queryClient.invalidateQueries({ queryKey: TUNNELS_KEY });
    },
  });

  return {
    installEngine: installEngine.mutateAsync,
    updateEngine: updateEngine.mutateAsync,
    installingEngine: installEngine.isPending || updateEngine.isPending,
    create: create.mutateAsync,
    importLink: importLink.mutateAsync,
    update: update.mutateAsync,
    setEnable: setEnable.mutateAsync,
    remove: remove.mutateAsync,
  };
}

export async function fetchSetupLink(id: number): Promise<string> {
  const msg = await HttpUtil.get(`/panel/api/tunnels/link/${id}`);
  if (!msg?.success || typeof msg.obj !== 'string') throw new Error(msg?.msg || '');
  return msg.obj;
}

const TunnelStatusSchema = z.object({
  connected: z.boolean(),
  latencyMs: z.number(),
  exitIp: z.string(),
  exitLoc: z.string(),
  error: z.string(),
  // Only a Rakhsh tunnel reports lanes; other engines send zeros.
  lanesUp: z.number().optional(),
  targetLanes: z.number().optional(),
  lossPct: z.number().optional(),
  history: z.array(z.number()).optional(),
});
export type TunnelStatus = z.infer<typeof TunnelStatusSchema>;

async function fetchTunnelStatus(id: number): Promise<TunnelStatus> {
  const msg = await HttpUtil.get(`/panel/api/tunnels/status/${id}`, undefined, { silent: true });
  if (!msg?.success) throw new Error(msg?.msg || 'Failed to test tunnel');
  const validated = parseMsg(msg, TunnelStatusSchema, 'tunnels/status');
  if (!validated.obj) throw new Error('Failed to test tunnel');
  return validated.obj;
}

// Each test sends a real request through the tunnel, so it runs once a minute, not on every render.
export function useTunnelStatus(id: number, enabled: boolean) {
  return useQuery({
    queryKey: [...TUNNELS_KEY, 'status', id],
    queryFn: () => fetchTunnelStatus(id),
    enabled,
    retry: false,
    refetchInterval: 60_000,
  });
}

// Same cache entries as useTunnelStatus, so a row and the page counts share one probe per tunnel.
export function useTunnelStatuses(
  tunnels: readonly { id: number; enable: boolean; managePort?: number }[],
) {
  return useQueries({
    queries: tunnels.map((tu) => ({
      queryKey: [...TUNNELS_KEY, 'status', tu.id],
      queryFn: () => fetchTunnelStatus(tu.id),
      enabled: tu.enable && !tu.managePort,
      retry: false,
      refetchInterval: 60_000,
    })),
  });
}

const EngineInfoSchema = z.object({
  engine: z.string().optional(),
  installed: z.boolean(),
  version: z.string(),
  pinned: z.string(),
});

export function useTunnelEngine(engine: TunnelEngine) {
  return useQuery({
    queryKey: [...TUNNELS_KEY, 'engine', engine],
    queryFn: async () => {
      const msg = await HttpUtil.get('/panel/api/tunnels/engine', { engine }, { silent: true });
      if (!msg?.success) throw new Error(msg?.msg || 'Failed to read the tunnel engine');
      return parseMsg(msg, EngineInfoSchema, 'tunnels/engine').obj ?? null;
    },
  });
}

// The catalog changes only with a Dalan release, so it is fetched once per engine and page visit.
export function useTunnelOptions(engine: TunnelEngine, enabled = true) {
  return useQuery({
    queryKey: ['tunnel-options', engine],
    staleTime: Infinity,
    enabled,
    queryFn: async () => {
      const msg = await HttpUtil.get('/panel/api/tunnels/options', { engine }, { silent: true });
      if (!msg?.success) throw new Error(msg?.msg || 'Failed to read the tunnel options');
      const options = parseMsg(msg, TunnelOptionListSchema, 'tunnels/options').obj ?? [];
      return options.map((o) => ({ ...o, choices: o.choices ?? [] }));
    },
  });
}
