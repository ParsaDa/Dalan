import { lazy, useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { useStatusQuery } from '@/api/queries/useStatusQuery';
import Icon from '@/components/dl/Icon';
import { Badge, Btn, TileHead } from '@/components/dl/primitives';
import { LazyMount } from '@/components/utility';
import { formatPanelVersion } from '@/lib/panel-version';
import type { PanelUpdateInfo } from '@/pages/index/PanelUpdateModal';
import { HttpUtil } from '@/utils';
import { BusyDialog, type BusyEvent } from './BusyDialog';

const PanelUpdateModal = lazy(() => import('@/pages/index/PanelUpdateModal'));
const VersionModal = lazy(() => import('@/pages/index/VersionModal'));

async function fetchUpdateInfo(): Promise<PanelUpdateInfo | null> {
  const msg = await HttpUtil.get<PanelUpdateInfo>('/panel/api/server/getPanelUpdateInfo');
  return msg?.success && msg.obj ? msg.obj : null;
}

// Panel and Xray versions with their update dialogs (Settings › Updates).
export default function UpdatesSection() {
  const { t } = useTranslation();
  const { status } = useStatusQuery();
  const [info, setInfo] = useState<PanelUpdateInfo>({
    currentVersion: '',
    latestVersion: '',
    updateAvailable: false,
  });
  const [dev, setDev] = useState(false);
  const [open, setOpen] = useState<'panel' | 'xray' | null>(null);
  const [busy, setBusyState] = useState({ busy: false, tip: '' });
  const onBusy = useCallback(
    (e: BusyEvent) => setBusyState((b) => ({ busy: e.busy, tip: e.tip ?? b.tip })),
    [],
  );

  useEffect(() => {
    HttpUtil.post<{ devChannelEnable?: boolean }>('/panel/api/setting/defaultSettings').then(
      (msg) => {
        if (msg?.success && msg.obj) setDev(!!msg.obj.devChannelEnable);
      },
    );
    fetchUpdateInfo().then((next) => next && setInfo(next));
  }, []);

  async function changeChannel(next: boolean) {
    const res = await HttpUtil.post('/panel/api/server/setUpdateChannel', { dev: next });
    if (!res?.success) return;
    setDev(next);
    const fresh = await fetchUpdateInfo();
    if (fresh) setInfo(fresh);
  }

  const current = window.X_UI_CUR_VER || info.currentVersion || '?';
  return (
    <section className="tile">
      <TileHead icon="refresh" title={t('settingsPage.nav.updates')} />
      <div className="list">
        <div className="li">
          <span className="ico acc">
            <Icon name="refresh" />
          </span>
          <div className="f1">
            <b>{t('settingsPage.updates.panel')}</b>
            <div className="mut num" style={{ fontSize: 12.5 }}>
              <bdi>{formatPanelVersion(current)}</bdi>
              {info.updateAvailable ? (
                <>
                  {' → '}
                  <bdi>{formatPanelVersion(info.latestVersion)}</bdi>
                </>
              ) : (
                ` · ${t('settingsPage.updates.upToDate')}`
              )}
            </div>
          </div>
          {info.updateAvailable && <Badge acc>{t('settingsPage.updates.available')}</Badge>}
          <Btn
            kind={info.updateAvailable ? 'pri' : undefined}
            size="sm"
            icon="refresh"
            onClick={() => setOpen('panel')}
          >
            {t('pages.index.updatePanel')}
          </Btn>
        </div>
        <div className="li">
          <span className="ico">
            <Icon name="xray" />
          </span>
          <div className="f1">
            <b>{t('settingsPage.updates.xray')}</b>
            <div className="mut num" style={{ fontSize: 12.5 }}>
              <bdi>{status.xray.version || '–'}</bdi>
            </div>
          </div>
          <Btn size="sm" icon="xray" onClick={() => setOpen('xray')}>
            {t('pages.index.xraySwitch')}
          </Btn>
        </div>
      </div>
      <LazyMount when={open === 'panel'}>
        <PanelUpdateModal
          open={open === 'panel'}
          info={info}
          devChannelEnable={dev}
          onChannelChange={changeChannel}
          onClose={() => setOpen(null)}
          onBusy={onBusy}
        />
      </LazyMount>
      <LazyMount when={open === 'xray'}>
        <VersionModal
          open={open === 'xray'}
          status={status}
          onClose={() => setOpen(null)}
          onBusy={onBusy}
        />
      </LazyMount>
      <BusyDialog busy={busy.busy} tip={busy.tip || t('loading')} />
    </section>
  );
}
