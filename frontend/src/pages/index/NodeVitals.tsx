import { useTranslation } from 'react-i18next';
import { Card } from 'antd';

import { CountUp } from '@/components/ui';
import { formatUptime } from '@/lib/format-uptime';
import type { NodeRecord } from '@/api/queries/useNodesQuery';

interface NodeVitalsProps {
  node: NodeRecord;
}

interface Tile {
  key: string;
  label: string;
  value: number;
  unit: string;
  format: (value: number) => string;
  available: boolean;
}

export default function NodeVitals({ node }: NodeVitalsProps) {
  const { t } = useTranslation();
  const missing = node.unavailableMetrics ?? [];
  const tiles: Tile[] = [
    {
      key: 'cpu',
      label: t('pages.index.cpu'),
      value: node.cpuPct ?? 0,
      unit: '%',
      format: (v) => v.toFixed(1),
      available: !missing.includes('cpu'),
    },
    {
      key: 'mem',
      label: t('pages.index.memory'),
      value: node.memPct ?? 0,
      unit: '%',
      format: (v) => v.toFixed(1),
      available: !missing.includes('mem'),
    },
    {
      key: 'latency',
      label: t('pages.nodes.latency'),
      value: node.latencyMs ?? 0,
      unit: 'ms',
      format: (v) => v.toFixed(0),
      available: node.status === 'online',
    },
  ];

  return (
    <div className="ov-vitals node-vitals">
      {tiles.map((tile) => (
        <Card key={tile.key} className="ov-tile" styles={{ body: { padding: 0 } }}>
          <div className="ov-tile-head">
            <span className="ov-kicker">{tile.label}</span>
          </div>
          <div className="ov-tile-value">
            <span className="ov-tile-number">
              {tile.available ? <CountUp value={tile.value} format={tile.format} /> : '—'}
            </span>
            {tile.available && <span className="ov-tile-unit">{tile.unit}</span>}
          </div>
        </Card>
      ))}
      <Card className="ov-tile" styles={{ body: { padding: 0 } }}>
        <div className="ov-tile-head">
          <span className="ov-kicker">{t('pages.nodes.uptime')}</span>
        </div>
        <div className="ov-tile-value">
          <span className="ov-tile-number">
            {formatUptime(node.uptimeSecs, !missing.includes('uptime'))}
          </span>
        </div>
        <div className="ov-tile-foot">
          <span>
            {t('pages.nodes.xrayVersion')} {node.xrayVersion || '—'}
          </span>
          <span>
            {t('pages.nodes.panelVersion')} {node.panelVersion || '—'}
          </span>
        </div>
      </Card>
      {node.lastError && node.status !== 'online' && (
        <div className="ov-health node-error">{node.lastError}</div>
      )}
    </div>
  );
}
