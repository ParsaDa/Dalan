import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router';

import { useStatusQuery } from '@/api/queries/useStatusQuery';
import type { NodeRecord } from '@/api/queries/useNodesQuery';
import { HIGH_USAGE } from '../primitives';
import type { Status } from '../primitives';
import { useActiveNode } from './data';

export function nodeStatus(node: NodeRecord): Status {
  if (node.status === 'offline') return 'bad';
  if (node.status !== 'online') return 'off';
  const busy = (node.cpuPct ?? 0) / 100 >= HIGH_USAGE || (node.memPct ?? 0) / 100 >= HIGH_USAGE;
  return busy ? 'warn' : 'ok';
}

export function useNow(intervalMs = 1000): number {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const id = window.setInterval(() => setNow(Date.now()), intervalMs);
    return () => window.clearInterval(id);
  }, [intervalMs]);
  return now;
}

export function useFrameStatus() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { status, updatedAt } = useStatusQuery();
  const { active, direct } = useActiveNode();
  const now = useNow();
  let tone: Status;
  if (active) {
    tone = nodeStatus(active);
  } else {
    const xrayDown = status.xray.state !== 'running';
    const nodeDown = direct.some((n) => nodeStatus(n) === 'bad');
    tone = xrayDown || nodeDown ? 'warn' : 'ok';
  }
  const word = active
    ? t(
        {
          ok: 'shell.status.healthy',
          warn: 'shell.status.highLoad',
          bad: 'shell.status.notAnswering',
          off: 'shell.status.paused',
        }[tone],
      )
    : t(tone === 'ok' ? 'shell.status.healthy' : 'shell.status.attention');
  const ago = updatedAt ? Math.max(0, Math.round((now - updatedAt) / 1000)) : 0;
  return {
    name: active?.name ?? t('pages.index.thisServer'),
    tone,
    word,
    ago,
    openAlerts: () => navigate({ pathname: '/', hash: '#alerts' }),
  };
}
