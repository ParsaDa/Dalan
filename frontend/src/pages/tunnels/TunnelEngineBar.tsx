import { useTranslation } from 'react-i18next';
import { Alert, Button, Space, Typography } from 'antd';

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
  if (!info) return null;

  if (!info.installed) {
    return (
      <Alert
        type="warning"
        showIcon
        title={t('pages.tunnels.engineMissing', { engine: label })}
        description={t('pages.tunnels.engineMissingHint', { engine: label })}
        action={
          <Button
            size="small"
            loading={installingEngine}
            onClick={async () =>
              report(await (nodeId ? updateEngine({ nodeId, engine }) : installEngine(engine)))
            }
          >
            {t('pages.tunnels.installEngine', { engine: label })}
          </Button>
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
      <Button
        size="small"
        loading={installingEngine}
        onClick={async () => report(await updateEngine({ nodeId: nodeId ?? 0, engine }))}
      >
        {t('pages.tunnels.updateEngine')}
      </Button>
    </Space>
  );
}
