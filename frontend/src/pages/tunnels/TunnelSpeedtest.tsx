import { useTranslation } from 'react-i18next';

import { useSpeedtest } from '@/api/queries/useSpeedtest';
import { Btn } from '@/components/dl/primitives';
import { Modal } from '@/components/dl/overlays';
import { SpeedtestPanel, SpeedtestRunButton } from '@/pages/index/dashboard/SpeedtestPanel';
import type { ServerTunnel } from '@/api/queries/useTunnels';

function Body({ tunnel, onClose }: { tunnel: ServerTunnel; onClose: () => void }) {
  const { t } = useTranslation();
  const run = useSpeedtest(tunnel.id);
  return (
    <div className="col" style={{ gap: 16 }}>
      <div className="row sp">
        <h2 className="h2">{t('tunnels.speedtest.title', { name: tunnel.name })}</h2>
        <SpeedtestRunButton run={run} />
      </div>
      <p className="mut" style={{ margin: 0, fontSize: 13 }}>
        {t('tunnels.speedtest.hint')}
      </p>
      <SpeedtestPanel run={run} />
      <div className="row" style={{ justifyContent: 'flex-end' }}>
        <Btn onClick={onClose}>{t('close')}</Btn>
      </div>
    </div>
  );
}

export function TunnelSpeedtest({
  tunnel,
  onClose,
}: {
  tunnel: ServerTunnel | null;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  return (
    <Modal open={tunnel !== null} onClose={onClose} label={t('tunnels.speedtest.open')} width={680}>
      {tunnel && <Body tunnel={tunnel} onClose={onClose} />}
    </Modal>
  );
}
