import { useEffect, useRef } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { keys } from '@/api/queryKeys';
import { SpeedtestResultSchema, SpeedtestStatusSchema } from '@/schemas/speedtest';
import { HttpUtil } from '@/utils';
import { parseMsg } from '@/utils/zodValidate';

const POLL_MS = 500;

export function useSpeedtest() {
  const qc = useQueryClient();

  const status = useQuery({
    queryKey: keys.speedtest.status(),
    queryFn: async () => {
      const msg = await HttpUtil.get('/panel/api/server/speedtest/status', undefined, {
        silent: true,
      });
      if (!msg?.success) throw new Error(msg?.msg || 'Failed to read the speed test');
      const parsed = parseMsg(msg, SpeedtestStatusSchema, 'server/speedtest/status').obj;
      if (!parsed) throw new Error('Empty speed test status');
      return parsed;
    },
    staleTime: 0,
    refetchInterval: (q) => (q.state.data?.running ? POLL_MS : false),
  });

  const history = useQuery({
    queryKey: keys.speedtest.history(),
    queryFn: async () => {
      const msg = await HttpUtil.get('/panel/api/server/speedtest/history', undefined, {
        silent: true,
      });
      if (!msg?.success) throw new Error(msg?.msg || 'Failed to read the speed test history');
      return parseMsg(msg, z.array(SpeedtestResultSchema), 'server/speedtest/history').obj ?? [];
    },
    staleTime: 30_000,
  });

  const start = useMutation({
    mutationFn: async () => {
      const msg = await HttpUtil.post('/panel/api/server/speedtest/start');
      if (!msg?.success) throw new Error(msg?.msg || 'Failed to start the speed test');
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: keys.speedtest.status() }),
  });

  const running = status.data?.running ?? false;
  const wasRunning = useRef(false);
  useEffect(() => {
    if (wasRunning.current && !running) {
      void qc.invalidateQueries({ queryKey: keys.speedtest.history() });
    }
    wasRunning.current = running;
  }, [running, qc]);

  return { status: status.data, history: history.data ?? [], start };
}
