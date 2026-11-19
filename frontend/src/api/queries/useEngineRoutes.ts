import { useQuery } from '@tanstack/react-query';
import { z } from 'zod';

import { HttpUtil } from '@/utils';
import { parseMsg } from '@/utils/zodValidate';
import { EngineRouteChoiceSchema, type EngineRouteChoice } from '@/generated/zod';

export type { EngineRouteChoice };

// The outbounds change when tunnels or Tor exits are added, so the list is read again each time a dialog opens.
export function useEngineRoutes() {
  return useQuery({
    queryKey: ['engine-routes'],
    staleTime: 0,
    retry: false,
    queryFn: async () => {
      const msg = await HttpUtil.get('/panel/api/tunnels/engine/routes', undefined, {
        silent: true,
      });
      if (!msg?.success) throw new Error(msg?.msg || 'Failed to read the download routes');
      return parseMsg(msg, z.array(EngineRouteChoiceSchema), 'tunnels/engine/routes').obj ?? [];
    },
  });
}
