import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { RunStatusSchema, type Candidate } from '@/generated/zod';
import { HttpUtil } from '@/utils';
import { parseMsg } from '@/utils/zodValidate';

const POLL_MS = 1500;
const KEY = ['tunnel-benchmark'] as const;
const JSON_HEADERS = { headers: { 'Content-Type': 'application/json' } };

export interface BenchmarkRequest {
  nodeId: number;
  address: string;
  objective: 'speed' | 'balanced';
  candidates: Candidate[];
}

export function useBenchmark() {
  const qc = useQueryClient();
  const refresh = () => qc.invalidateQueries({ queryKey: KEY });

  const status = useQuery({
    queryKey: KEY,
    staleTime: 0,
    refetchInterval: (q) => (q.state.data?.running ? POLL_MS : false),
    queryFn: async () => {
      const msg = await HttpUtil.get('/panel/api/tunnels/benchmark/status', undefined, {
        silent: true,
      });
      if (!msg?.success) throw new Error(msg?.msg || 'Failed to read the benchmark');
      const parsed = parseMsg(msg, RunStatusSchema, 'tunnels/benchmark/status').obj;
      if (!parsed) throw new Error('Empty benchmark status');
      return parsed;
    },
  });

  const start = useMutation({
    mutationFn: async (request: BenchmarkRequest) => {
      const msg = await HttpUtil.post('/panel/api/tunnels/benchmark/start', request, JSON_HEADERS);
      if (!msg?.success) throw new Error(msg?.msg || 'Failed to start the benchmark');
    },
    onSuccess: refresh,
  });

  const cancel = useMutation({
    mutationFn: async () => {
      await HttpUtil.post('/panel/api/tunnels/benchmark/cancel');
    },
    onSuccess: refresh,
  });

  return { status: status.data, start, cancel };
}
