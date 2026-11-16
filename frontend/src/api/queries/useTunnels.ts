import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { HttpUtil } from '@/utils';
import { parseMsg } from '@/utils/zodValidate';
import { ServerTunnelSchema, type ServerTunnel } from '@/generated/zod';

export type { ServerTunnel };

const JSON_HEADERS = { headers: { 'Content-Type': 'application/json' } };
const TUNNELS_KEY = ['tunnels'] as const;
// The API sends null for an empty tag list; the table and form want an array.
const stringList = z.array(z.string()).nullish();
const TunnelListSchema = z.array(
  ServerTunnelSchema.extend({
    routeInbounds: stringList,
    forwardPorts: stringList,
    fallbackTransports: stringList,
  }),
);

export type NewPortalValues =
  | {
      name: string;
      address: string;
      port: number;
      engine: 'xray';
      security: 'reality' | 'none';
      serverName: string;
      routeInbounds: string[];
    }
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
    };

async function fetchTunnels(): Promise<ServerTunnel[]> {
  const msg = await HttpUtil.get('/panel/api/tunnels/list', undefined, { silent: true });
  if (!msg?.success) throw new Error(msg?.msg || 'Failed to fetch tunnels');
  const validated = parseMsg(msg, TunnelListSchema, 'tunnels/list');
  return (validated.obj ?? []).map((t) => ({
    ...t,
    routeInbounds: t.routeInbounds ?? [],
    forwardPorts: t.forwardPorts ?? [],
    fallbackTransports: t.fallbackTransports ?? [],
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
  const setEnable = useMutation({
    mutationFn: ({ id, enable }: { id: number; enable: boolean }) =>
      HttpUtil.post(`/panel/api/tunnels/setEnable/${id}`, { enable }),
    onSuccess: refresh,
  });
  const remove = useMutation({
    mutationFn: (id: number) => HttpUtil.post(`/panel/api/tunnels/del/${id}`),
    onSuccess: refresh,
  });

  const installEngine = useMutation({
    mutationFn: () => HttpUtil.post('/panel/api/tunnels/engine/install'),
    onSuccess: (msg) => {
      if (msg?.success) queryClient.invalidateQueries({ queryKey: TUNNELS_KEY });
    },
  });

  return {
    installEngine: installEngine.mutateAsync,
    installingEngine: installEngine.isPending,
    create: create.mutateAsync,
    importLink: importLink.mutateAsync,
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

const EngineInfoSchema = z.object({ installed: z.boolean(), version: z.string() });

export function useTunnelEngine() {
  return useQuery({
    queryKey: [...TUNNELS_KEY, 'engine'],
    queryFn: async () => {
      const msg = await HttpUtil.get('/panel/api/tunnels/engine', undefined, { silent: true });
      if (!msg?.success) throw new Error(msg?.msg || 'Failed to read the tunnel engine');
      return parseMsg(msg, EngineInfoSchema, 'tunnels/engine').obj ?? null;
    },
  });
}
