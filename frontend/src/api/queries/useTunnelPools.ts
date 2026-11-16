import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { HttpUtil } from '@/utils';
import { parseMsg } from '@/utils/zodValidate';
import { TunnelPoolSchema, type TunnelPool } from '@/generated/zod';

export type { TunnelPool };

const JSON_HEADERS = { headers: { 'Content-Type': 'application/json' } };
const POOLS_KEY = ['tunnel-pools'] as const;
// The API sends null for an empty list; the table and form want an array.
const PoolListSchema = z.array(
  TunnelPoolSchema.extend({
    members: z.array(z.string()).nullish(),
    routeInbounds: z.array(z.string()).nullish(),
  }),
);

export interface NewPoolValues {
  name: string;
  members: string[];
  routeInbounds: string[];
}

async function fetchPools(): Promise<TunnelPool[]> {
  const msg = await HttpUtil.get('/panel/api/tunnels/pools/list', undefined, { silent: true });
  if (!msg?.success) throw new Error(msg?.msg || 'Failed to fetch tunnel pools');
  const validated = parseMsg(msg, PoolListSchema, 'tunnels/pools/list');
  return (validated.obj ?? []).map((p) => ({
    ...p,
    members: p.members ?? [],
    routeInbounds: p.routeInbounds ?? [],
  }));
}

export function useTunnelPoolsQuery() {
  const query = useQuery({
    queryKey: POOLS_KEY,
    queryFn: fetchPools,
    // The panel moves traffic on its own, so the page keeps up with it.
    refetchInterval: 15_000,
  });
  return { pools: query.data ?? [], loading: query.isFetching };
}

export function useTunnelPoolMutations() {
  const queryClient = useQueryClient();
  const refresh = (msg: { success?: boolean } | null | undefined) => {
    if (msg?.success) queryClient.invalidateQueries({ queryKey: POOLS_KEY });
  };
  const create = useMutation({
    mutationFn: (values: NewPoolValues) =>
      HttpUtil.post('/panel/api/tunnels/pools/add', values, JSON_HEADERS),
    onSuccess: refresh,
  });
  const remove = useMutation({
    mutationFn: (id: number) => HttpUtil.post(`/panel/api/tunnels/pools/del/${id}`),
    onSuccess: refresh,
  });
  return { create: create.mutateAsync, remove: remove.mutateAsync };
}
