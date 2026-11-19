import { useQuery } from '@tanstack/react-query';
import { z } from 'zod';

import { HttpUtil } from '@/utils';
import { parseMsg } from '@/utils/zodValidate';
import { TorExitViewSchema, type TorExitView } from '@/generated/zod';

export type { TorExitView };

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
