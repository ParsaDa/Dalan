import { useTranslation } from 'react-i18next';

import { useTorExits } from '@/api/queries/useTor';
import Icon from '@/components/dl/Icon';
import { Btn, TileHead } from '@/components/dl/primitives';
import { useNum } from '@/components/dl/numbers';
import TorPanel from '@/pages/tor/TorPanel';

// Tor exits and PIA live with the other outbounds; PIA's setup opens from the Outbounds page.
export default function TorPiaTiles({
  onOpenPia,
  onChanged,
}: {
  onOpenPia: () => void;
  onChanged?: () => void;
}) {
  const { t } = useTranslation();
  const num = useNum();
  const { exits } = useTorExits();
  const working = exits.filter((e) => e.status?.state === 'connected').length;
  return (
    <>
      <section className="tile edge s7" aria-label="Tor">
        <TileHead icon="tor" title="Tor" />
        <div className="row wrap" style={{ gap: 22, marginBottom: 12 }}>
          <div>
            <div className="kpi num" style={{ fontSize: 30 }}>
              {num.int(exits.length)}
            </div>
            <div className="lbl">{t('torPage.exitsOn')}</div>
          </div>
          <div>
            <div className="kpi num" style={{ fontSize: 30 }}>
              {num.int(working)}
            </div>
            <div className="lbl">{t('torPage.exitsWorking')}</div>
          </div>
        </div>
        <TorPanel onChanged={onChanged} />
      </section>
      <section className="tile edge s5" aria-label="PIA" style={{ alignSelf: 'start' }}>
        <TileHead icon="shield" title="PIA" />
        <p className="mut" style={{ margin: '0 0 12px', fontSize: 13 }}>
          {t('torPage.piaHint')}
        </p>
        <div className="row wrap" style={{ gap: 8 }}>
          <Btn kind="pri" icon="shield" onClick={onOpenPia}>
            {t('pages.xray.pia.menu')}
          </Btn>
        </div>
        <div className="sep" style={{ margin: '12px 0' }} />
        <p className="mut row" style={{ margin: 0, fontSize: 13, gap: 8 }}>
          <Icon name="info" size={16} />
          {t('torPage.flagsHint')}
        </p>
      </section>
    </>
  );
}
