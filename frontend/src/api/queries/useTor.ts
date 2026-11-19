import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { HttpUtil } from '@/utils';
import { parseMsg } from '@/utils/zodValidate';
import {
  TorExitViewSchema,
  TrialResultSchema,
  type TorExitView,
  type TrialResult,
} from '@/generated/zod';
import type { EngineRoute } from '@/schemas/engineRoute';

export type { TorExitView, TrialResult };

export const TOR_KEY = ['tor'] as const;
// Each exit is probed through Tor on the server, which reuses a result for 30 seconds.
const TOR_POLL_MS = 15_000;

async function fetchTorExits(): Promise<TorExitView[]> {
  const msg = await HttpUtil.get('/panel/api/tor/list', undefined, { silent: true });
  if (!msg?.success) throw new Error(msg?.msg || 'Failed to fetch Tor exits');
  return parseMsg(msg, z.array(TorExitViewSchema), 'tor/list').obj ?? [];
}

export function useTorExits() {
  const query = useQuery({
    queryKey: [...TOR_KEY, 'list'],
    queryFn: fetchTorExits,
    refetchInterval: TOR_POLL_MS,
    refetchIntervalInBackground: false,
  });
  return {
    exits: query.data ?? [],
    loading: query.isFetching,
    fetched: query.data !== undefined || query.isError,
    fetchError: query.error ? (query.error as Error).message : '',
    refetch: query.refetch,
  };
}

const EngineInfoSchema = z.object({
  installed: z.boolean(),
  version: z.string(),
  pinned: z.string(),
});

export function useTorEngine(enabled: boolean) {
  return useQuery({
    queryKey: [...TOR_KEY, 'engine'],
    enabled,
    queryFn: async () => {
      const msg = await HttpUtil.get('/panel/api/tor/engine', undefined, { silent: true });
      if (!msg?.success) throw new Error(msg?.msg || 'Failed to read the Tor engine');
      return parseMsg(msg, EngineInfoSchema, 'tor/engine').obj ?? null;
    },
  });
}

export function useTorBridges(enabled: boolean) {
  return useQuery({
    queryKey: [...TOR_KEY, 'bridges'],
    enabled,
    queryFn: async () => {
      const msg = await HttpUtil.get('/panel/api/tor/bridges', undefined, { silent: true });
      if (!msg?.success) throw new Error(msg?.msg || 'Failed to read the Tor bridges');
      return parseMsg(msg, z.object({ bridges: z.string() }), 'tor/bridges').obj?.bridges ?? '';
    },
  });
}

// A test can run for 90 seconds and an install for minutes, so neither uses the default timeout.
const TEST_TIMEOUT_MS = 120_000;
const INSTALL_TIMEOUT_MS = 600_000;

export function useTorMutations() {
  const queryClient = useQueryClient();
  const refresh = (msg: { success?: boolean } | null | undefined) => {
    if (msg?.success) queryClient.invalidateQueries({ queryKey: TOR_KEY });
  };
  const quiet = { silent: true, silentSuccess: true } as const;

  const add = useMutation({
    mutationFn: (country: string) => HttpUtil.post('/panel/api/tor/add', { country }, quiet),
    onSuccess: refresh,
  });
  const remove = useMutation({
    mutationFn: (id: number) => HttpUtil.post(`/panel/api/tor/del/${id}`, undefined, quiet),
    onSuccess: refresh,
  });
  const test = useMutation({
    mutationFn: async (country: string): Promise<{ result: TrialResult } | { error: string }> => {
      const msg = await HttpUtil.post(
        '/panel/api/tor/test',
        { country },
        { ...quiet, timeout: TEST_TIMEOUT_MS },
      );
      if (!msg?.success) return { error: msg?.msg || 'Failed to test the country' };
      const result = parseMsg(msg, TrialResultSchema, 'tor/test').obj;
      return result ? { result } : { error: 'Failed to test the country' };
    },
  });
  const installEngine = useMutation({
    mutationFn: (route: EngineRoute) =>
      HttpUtil.post(
        '/panel/api/tor/engine/install',
        { route },
        { ...quiet, timeout: INSTALL_TIMEOUT_MS },
      ),
    onSuccess: refresh,
  });
  const saveBridges = useMutation({
    mutationFn: (bridges: string) => HttpUtil.post('/panel/api/tor/bridges', { bridges }, quiet),
    onSuccess: refresh,
  });

  return {
    add: add.mutateAsync,
    remove: remove.mutateAsync,
    test: test.mutateAsync,
    installEngine: installEngine.mutateAsync,
    installing: installEngine.isPending,
    saveBridges: saveBridges.mutateAsync,
    savingBridges: saveBridges.isPending,
  };
}
