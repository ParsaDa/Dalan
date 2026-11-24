import { useEffect, useRef } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { keys } from '@/api/queryKeys';
import { SpeedtestResultSchema, SpeedtestStatusSchema } from '@/schemas/speedtest';
import { HttpUtil } from '@/utils';
import { parseMsg } from '@/utils/zodValidate';

const POLL_MS = 500;

// Target 0 is this server's own test; any other id tests that entry tunnel.
function speedtestUrl(action: 'start' | 'status' | 'history', tunnelId: number): string {
  return tunnelId === 0
    ? `/panel/api/server/speedtest/${action}`
    : `/panel/api/tunnels/speedtest/${action}/${tunnelId}`;
}

export function useSpeedtest(tunnelId = 0) {
  const qc = useQueryClient();

  const status = useQuery({
    queryKey: keys.speedtest.status(tunnelId),
    queryFn: async () => {
      const msg = await HttpUtil.get(speedtestUrl('status', tunnelId), undefined, {
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
    queryKey: keys.speedtest.history(tunnelId),
    queryFn: async () => {
      const msg = await HttpUtil.get(speedtestUrl('history', tunnelId), undefined, {
        silent: true,
      });
      if (!msg?.success) throw new Error(msg?.msg || 'Failed to read the speed test history');
      return parseMsg(msg, z.array(SpeedtestResultSchema), 'server/speedtest/history').obj ?? [];
    },
    staleTime: 30_000,
  });

  const start = useMutation({
    mutationFn: async () => {
      const msg = await HttpUtil.post(speedtestUrl('start', tunnelId));
      if (!msg?.success) throw new Error(msg?.msg || 'Failed to start the speed test');
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: keys.speedtest.status(tunnelId) }),
  });

  const running = status.data?.running ?? false;
  const wasRunning = useRef(false);
  useEffect(() => {
    if (wasRunning.current && !running) {
      void qc.invalidateQueries({ queryKey: keys.speedtest.history(tunnelId) });
    }
    wasRunning.current = running;
  }, [running, qc, tunnelId]);

  return { status: status.data, history: history.data ?? [], start };
}
