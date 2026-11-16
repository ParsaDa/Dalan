import { useMutation } from '@tanstack/react-query';
import { z } from 'zod';

import { HttpUtil } from '@/utils';
import { parseMsg } from '@/utils/zodValidate';

const JSON_HEADERS = { headers: { 'Content-Type': 'application/json' } };

// Ports worth trying first: the ones filtered networks tend to leave open.
export const ROUTE_TEST_PORTS = [443, 8443, 2053, 2083, 2087, 2096, 8080, 80];

const PortResultSchema = z.object({
  port: z.number(),
  reachable: z.boolean(),
  latencyMs: z.number(),
  error: z.string(),
});
export type PortResult = z.infer<typeof PortResultSchema>;

export function useRouteTest() {
  const mutation = useMutation({
    mutationFn: async (request: { nodeId: number; address: string }): Promise<PortResult[]> => {
      const msg = await HttpUtil.post(
        '/panel/api/tunnels/routetest',
        { ...request, ports: ROUTE_TEST_PORTS },
        JSON_HEADERS,
      );
      if (!msg?.success) throw new Error(msg?.msg || 'The route test could not run');
      return parseMsg(msg, z.array(PortResultSchema), 'tunnels/routetest').obj ?? [];
    },
  });
  return { run: mutation.mutateAsync, testing: mutation.isPending };
}
