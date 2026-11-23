import { useQuery } from '@tanstack/react-query';
import { useCallback, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router';

import { HttpUtil } from '@/utils';
import { keys } from '@/api/queryKeys';
import { useNodesQuery } from '@/api/queries/useNodesQuery';
import type { NodeRecord } from '@/api/queries/useNodesQuery';

export function useCurrentUser(): string {
  const { data } = useQuery({
    queryKey: keys.settings.currentUser(),
    queryFn: async () => {
      const msg = await HttpUtil.get('/panel/api/setting/user', undefined, { silent: true });
      const obj = msg?.success ? (msg.obj as { username?: unknown }) : null;
      return typeof obj?.username === 'string' ? obj.username : '';
    },
    staleTime: Infinity,
  });
  return data ?? '';
}

export function useClientTotal(): number | null {
  const { data } = useQuery({
    queryKey: keys.clients.list({ countOnly: true }),
    queryFn: async () => {
      const msg = await HttpUtil.get('/panel/api/clients/list/paged?page=1&pageSize=1', undefined, {
        silent: true,
      });
      const total = msg?.success ? (msg.obj as { total?: unknown })?.total : null;
      return typeof total === 'number' ? total : null;
    },
    refetchInterval: 30_000,
  });
  return data ?? null;
}

export function useGreetingPart(): GreetingPart {
  const [hour] = useState(() => new Date().getHours());
  return greetingPart(hour);
}

export type GreetingPart = 'morning' | 'afternoon' | 'evening';
export function greetingPart(hour: number): GreetingPart {
  if (hour >= 5 && hour < 12) return 'morning';
  if (hour >= 12 && hour < 18) return 'afternoon';
  return 'evening';
}

// The open node lives in the URL (?node=<id>), so tabs, the dashboard and reloads agree on it.
export function useActiveNode(): {
  nodes: NodeRecord[];
  direct: NodeRecord[];
  active: NodeRecord | null;
  select: (id: number | null) => void;
} {
  const { nodes } = useNodesQuery();
  const [params, setParams] = useSearchParams();
  const raw = Number(params.get('node'));
  const direct = useMemo(() => nodes.filter((n) => n.enable !== false && !n.transitive), [nodes]);
  const active = direct.find((n) => n.id === raw) ?? null;
  const select = useCallback(
    (id: number | null) => {
      setParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          if (id === null) next.delete('node');
          else next.set('node', String(id));
          return next;
        },
        { replace: true },
      );
    },
    [setParams],
  );
  return { nodes, direct, active, select };
}
