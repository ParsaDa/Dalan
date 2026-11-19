import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Alert, Button, Space, Typography } from 'antd';

import EngineRoutePicker from '@/components/engine/EngineRoutePicker';
import { engineRouteReady, PANEL_ROUTE, type EngineRoute } from '@/schemas/engineRoute';
import { useTunnelEngine, useTunnelMutations, type TunnelEngine } from '@/api/queries/useTunnels';

interface Props {
  engine: TunnelEngine;
  label: string;
  // A node that downloads the engine for a server that cannot reach GitHub; 0 or undefined means this server.
  nodeId?: number;
  report: (msg: { success?: boolean; msg?: string } | null | undefined) => boolean;
}

// One engine's state on this server: missing (offer the install) or installed (its release and an update).
export default function TunnelEngineBar({ engine, label, nodeId, report }: Props) {
  const { t } = useTranslation();
  const { data: info } = useTunnelEngine(engine);
  const { installEngine, updateEngine, installingEngine } = useTunnelMutations();
  const [route, setRoute] = useState<EngineRoute>(PANEL_ROUTE);
  if (!info) return null;

  // A node downloads as its own panel is set, so the route is only offered for this server.
  const picker = nodeId ? null : (
    <EngineRoutePicker id={`engine-route-${engine}`} value={route} onChange={setRoute} />
  );

  if (!info.installed) {
    return (
      <Alert
        type="warning"
        showIcon
        title={t('pages.tunnels.engineMissing', { engine: label })}
        description={t('pages.tunnels.engineMissingHint', { engine: label })}
        action={
          <Space wrap>
            {picker}
            <Button
              size="small"
              loading={installingEngine}
              disabled={!engineRouteReady(route)}
              onClick={async () =>
                report(
                  await (nodeId
                    ? updateEngine({ nodeId, engine, route })
                    : installEngine({ engine, route })),
                )
              }
            >
              {t('pages.tunnels.installEngine', { engine: label })}
            </Button>
          </Space>
        }
        style={{ marginBottom: 12 }}
      />
    );
  }
  return (
    <Space style={{ marginBottom: 12 }} wrap>
      <Typography.Text type="secondary">
        {t('pages.tunnels.engineVersion', {
          engine: label,
          version: info.version || t('pages.tunnels.engineVersionUnknown'),
        })}
      </Typography.Text>
      {picker}
      <Button
        size="small"
        loading={installingEngine}
        disabled={!engineRouteReady(route)}
        onClick={async () => report(await updateEngine({ nodeId: nodeId ?? 0, engine, route }))}
      >
        {t('pages.tunnels.updateEngine')}
      </Button>
    </Space>
  );
}
