import { useTranslation } from 'react-i18next';

import { useSpeedtest } from '@/api/queries/useSpeedtest';
import { TileHead } from '@/components/dl/primitives';
import { SpeedtestPanel, SpeedtestRunButton } from './SpeedtestPanel';

export function SpeedtestTile() {
  const { t } = useTranslation();
  const run = useSpeedtest();
  return (
    <section className="tile edge s12" aria-label={t('dashboard.speedtest.title')}>
      <TileHead
        icon="bolt"
        title={t('dashboard.speedtest.title')}
        right={<SpeedtestRunButton run={run} />}
      />
      <SpeedtestPanel run={run} />
    </section>
  );
}
