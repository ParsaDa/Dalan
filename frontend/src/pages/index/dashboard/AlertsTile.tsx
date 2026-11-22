import { useTranslation } from 'react-i18next';
import { Link } from 'react-router';

import Icon from '@/components/dl/Icon';
import type { IconName } from '@/components/dl/Icon';
import { Btn, Pill, TileHead } from '@/components/dl/primitives';
import type { Alert } from '@/generated/zod';
import { relTime } from './ActivityList';
import type { NodeRef } from './tunnelRow';
import { fmtInt } from '@/components/dl/numbers';

type Tone = 'bad' | 'warn' | 'neutral';
const KIND: Record<string, { tone: Tone; icon: IconName }> = {
  'node.offline': { tone: 'bad', icon: 'offc' },
  'xray.down': { tone: 'bad', icon: 'offc' },
  'node.load': { tone: 'warn', icon: 'alert' },
  'cpu.high': { tone: 'warn', icon: 'alert' },
  'memory.high': { tone: 'warn', icon: 'alert' },
  'clients.expiring': { tone: 'neutral', icon: 'clock' },
};

export function AlertsTile({
  alerts,
  nodes,
  onOpenNode,
  onRestartXray,
  className = 's5',
}: {
  alerts: readonly Alert[];
  nodes: readonly NodeRef[];
  onOpenNode?: (id: number) => void;
  onRestartXray?: () => void;
  className?: string;
}) {
  const { t } = useTranslation();
  const urgent = alerts.some((a) => KIND[a.kind]?.tone === 'bad');
  return (
    <section
      className={`tile edge${alerts.length ? (urgent ? ' st-bad' : ' st-warn') : ''} ${className}`}
      aria-label={t('dashboard.alerts.title')}
    >
      <TileHead
        icon="bell"
        title={t('dashboard.alerts.title')}
        right={
          alerts.length ? (
            <Pill status="warn">{t('dashboard.alerts.count', { n: fmtInt(alerts.length) })}</Pill>
          ) : (
            <Pill status="ok">{t('dashboard.alerts.none')}</Pill>
          )
        }
      />
      {alerts.length === 0 ? (
        <p className="mut" style={{ margin: 0, fontSize: 13 }}>
          {t('dashboard.alerts.noneHint')}
        </p>
      ) : (
        <div className="list">
          {alerts.map((a) => {
            const kind = KIND[a.kind] ?? { tone: 'warn' as Tone, icon: 'alert' as IconName };
            const node = nodes.find((n) => n.id === a.nodeId);
            const name = node?.name ?? t('pages.index.thisServer');
            const pct = fmtInt(Math.round(a.value));
            const since = a.startedAt ? relTime(t, a.startedAt) : '';
            let title = '';
            let sub = '';
            let action = null;
            switch (a.kind) {
              case 'node.offline':
                title = t('dashboard.alerts.offline', { node: name });
                sub = t('dashboard.alerts.offlineSub', { since });
                break;
              case 'node.load':
                title = t('dashboard.alerts.nodeLoad', { node: name, pct });
                sub = t('dashboard.alerts.sustained', { since });
                break;
              case 'cpu.high':
                title = t('dashboard.alerts.cpu', { pct });
                sub = t('dashboard.alerts.sustained', { since });
                break;
              case 'memory.high':
                title = t('dashboard.alerts.memory', { pct });
                sub = t('dashboard.alerts.sustained', { since });
                break;
              case 'xray.down':
                title = t('dashboard.alerts.xray');
                sub = t('dashboard.alerts.xraySub');
                if (onRestartXray)
                  action = (
                    <Btn size="sm" onClick={onRestartXray}>
                      {t('pages.index.restartXray')}
                    </Btn>
                  );
                break;
              case 'clients.expiring':
                title = t('dashboard.alerts.expiring', { n: pct });
                sub = t('dashboard.alerts.expiringSub');
                action = (
                  <Link className="btn sm" to="/clients">
                    {t('dashboard.alerts.review')}
                  </Link>
                );
                break;
              default:
                title = a.kind;
            }
            if (node && onOpenNode && (a.kind === 'node.offline' || a.kind === 'node.load'))
              action = (
                <Btn size="sm" onClick={() => onOpenNode(node.id)}>
                  {t('dashboard.open')}
                </Btn>
              );
            return (
              <div className="li" key={a.id}>
                <span className={`ico ${kind.tone === 'neutral' ? '' : kind.tone}`}>
                  <Icon name={kind.icon} />
                </span>
                <div className="f1">
                  <div style={{ fontWeight: 750 }}>{title}</div>
                  <div className="mut" style={{ fontSize: 12.5 }}>
                    {sub}
                  </div>
                </div>
                {action}
              </div>
            );
          })}
        </div>
      )}
    </section>
  );
}
