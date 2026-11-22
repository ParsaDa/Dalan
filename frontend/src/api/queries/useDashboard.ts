import { useQuery } from '@tanstack/react-query';
import { z } from 'zod';

import { ActivityPageSchema, AlertSchema, TrafficUsageSchema } from '@/generated/zod';
import type { ActivityPage, Alert, TrafficUsage } from '@/generated/zod';
import { HttpUtil } from '@/utils';
import { parseMsg } from '@/utils/zodValidate';
import { OnlinesSchema } from '@/schemas/client';

export const dashboardKeys = {
  alerts: () => ['dashboard', 'alerts'] as const,
  activity: (params: ActivityParams) => ['dashboard', 'activity', params] as const,
  traffic: () => ['dashboard', 'monthlyTraffic'] as const,
  onlines: () => ['dashboard', 'onlines'] as const,
};

export interface ActivityParams {
  page: number;
  pageSize: number;
  category?: string;
  actor?: string;
  nodeId?: number;
  days?: number;
}

async function getObj<T extends z.ZodType>(
  url: string,
  schema: T,
  params?: unknown,
): Promise<z.infer<T>> {
  const msg = await HttpUtil.get(url, params, { silent: true });
  if (!msg?.success) throw new Error(msg?.msg || `Failed to fetch ${url}`);
  return parseMsg(msg, schema, url).obj as z.infer<T>;
}

export function useAlerts() {
  const q = useQuery({
    queryKey: dashboardKeys.alerts(),
    queryFn: () => getObj('/panel/api/alerts/list', z.array(AlertSchema).nullable()),
    refetchInterval: 5000,
  });
  return { alerts: (q.data ?? []) as Alert[], loaded: q.data !== undefined };
}

export function useActivity(params: ActivityParams) {
  const q = useQuery({
    queryKey: dashboardKeys.activity(params),
    queryFn: () => getObj('/panel/api/activity/list', ActivityPageSchema, params),
    refetchInterval: 15000,
  });
  const page = q.data as ActivityPage | undefined;
  return { items: page?.items ?? [], total: page?.total ?? 0, loaded: page !== undefined };
}

export function useMonthlyTraffic() {
  const q = useQuery({
    queryKey: dashboardKeys.traffic(),
    queryFn: async () => {
      const usage = (await getObj(
        '/panel/api/server/monthlyTraffic',
        TrafficUsageSchema,
      )) as TrafficUsage;
      return {
        ...usage,
        daysLeft: Math.max(0, Math.floor((usage.nextReset - Date.now() / 1000) / 86400)),
      };
    },
    refetchInterval: 60000,
  });
  return q.data;
}

export function useOnlineCount(): number | null {
  const q = useQuery({
    queryKey: dashboardKeys.onlines(),
    queryFn: async () => {
      const msg = await HttpUtil.post('/panel/api/clients/onlines', undefined, { silent: true });
      if (!msg?.success) throw new Error(msg?.msg || 'Failed to fetch onlines');
      const obj = parseMsg(msg, OnlinesSchema, 'clients/onlines').obj;
      return Array.isArray(obj) ? obj.length : 0;
    },
    refetchInterval: 5000,
  });
  return q.data ?? null;
}
