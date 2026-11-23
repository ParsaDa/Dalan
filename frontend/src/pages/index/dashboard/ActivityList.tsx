import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import type { TFunction } from 'i18next';

import Icon from '@/components/dl/Icon';
import type { IconName } from '@/components/dl/Icon';
import { Badge } from '@/components/dl/primitives';
import type { Activity } from '@/generated/zod';
import type { NodeRef } from './tunnelRow';
import { fmtInt } from '@/components/dl/numbers';

// Bold runs come from <b>…</b> in the translation; elements are built, never parsed as HTML.
export function boldRuns(text: string): ReactNode[] {
  return text.split(/(<b>.*?<\/b>)/g).map((part, i) =>
    part.startsWith('<b>') ? (
      <b key={i}>
        <bdi>{part.slice(3, -4)}</bdi>
      </b>
    ) : (
      part
    ),
  );
}

export function relTime(t: TFunction, atSec: number, nowMs = Date.now()): string {
  const s = Math.max(0, Math.round(nowMs / 1000 - atSec));
  if (s < 60) return t('dashboard.ago.now');
  if (s < 3600) return t('dashboard.ago.min', { n: fmtInt(Math.floor(s / 60)) });
  if (s < 86400) return t('dashboard.ago.hour', { n: fmtInt(Math.floor(s / 3600)) });
  return t('dashboard.ago.day', { n: fmtInt(Math.floor(s / 86400)) });
}

const AREA_ICON: Record<string, IconName> = {
  client: 'users',
  group: 'groups',
  inbound: 'inbound',
  tunnel: 'tunnel',
  node: 'nodes',
  host: 'hosts',
  tor: 'tor',
  xray: 'xray',
  panel: 'refresh',
  settings: 'settings',
  account: 'lock',
  session: 'lock',
};

export function activityIcon(action: string): IconName {
  return AREA_ICON[action.split('.')[0]] ?? 'activity';
}

export function activitySentence(t: TFunction, i18nExists: (k: string) => boolean, a: Activity) {
  const who = t(`dashboard.activity.who.${a.actor}`, { defaultValue: a.actor });
  const key = `dashboard.activity.act.${a.action}`;
  const text = i18nExists(key)
    ? t(key, { who, target: a.target })
    : t('dashboard.activity.generic', { who, action: a.action, target: a.target });
  return boldRuns(text);
}

export function ActivityList({
  items,
  nodes,
}: {
  items: readonly Activity[];
  nodes: readonly NodeRef[];
}) {
  const { t, i18n } = useTranslation();
  return (
    <div className="list">
      {items.map((a) => {
        const node = a.nodeId ? nodes.find((n) => n.id === a.nodeId) : undefined;
        return (
          <div className="li" key={a.id}>
            <span className="ico acc">
              <Icon name={activityIcon(a.action)} />
            </span>
            <div className="f1">{activitySentence(t, (k) => i18n.exists(k), a)}</div>
            {node && <Badge>{node.name}</Badge>}
            <span className="mut num" style={{ fontSize: 12 }}>
              {relTime(t, a.at)}
            </span>
          </div>
        );
      })}
    </div>
  );
}
