import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import Icon from '@/components/dl/Icon';
import { Btn, TileHead, Toggle } from '@/components/dl/primitives';
import { HttpUtil, PromiseUtil } from '@/utils';
import { BusyDialog } from './BusyDialog';

function download(path: string) {
  window.location.href = (window.X_UI_BASE_PATH || '') + path;
}

// Back up and restore this panel's database (Settings › Backups).
export default function BackupsSection() {
  const { t } = useTranslation();
  const isPostgres = window.X_UI_DB_TYPE === 'postgres';
  const [keepHost, setKeepHost] = useState(true);
  const [busy, setBusy] = useState('');

  function restore() {
    const input = document.createElement('input');
    input.type = 'file';
    input.accept = '.dump,.db';
    input.addEventListener('change', async () => {
      const file = input.files?.[0];
      if (!file) return;
      const form = new FormData();
      form.append('db', file);
      form.append('keepHostSettings', String(keepHost));
      setBusy(`${t('pages.index.importDatabase')}…`);
      const upload = await HttpUtil.post('/panel/api/server/importDB', form, {
        headers: { 'Content-Type': 'multipart/form-data' },
      });
      if (!upload?.success) return setBusy('');
      // importDB schedules the panel restart server-side; wait it out, then reload.
      setBusy(`${t('pages.settings.restartPanel')}…`);
      await PromiseUtil.sleep(5000);
      window.location.reload();
    });
    input.click();
  }

  const rows = [
    {
      icon: 'down' as const,
      title: t('pages.index.exportDatabase'),
      desc: isPostgres
        ? t('pages.index.exportDatabasePgDesc')
        : t('pages.index.exportDatabaseDesc'),
      action: (
        <Btn kind="pri" size="sm" icon="down" onClick={() => download('panel/api/server/getDb')}>
          {t('pages.index.exportDatabase')}
        </Btn>
      ),
    },
    ...(isPostgres
      ? [
          {
            icon: 'down' as const,
            title: t('pages.index.migrationDownload'),
            desc: t('pages.index.migrationDownloadPgDesc'),
            action: (
              <Btn size="sm" icon="down" onClick={() => download('panel/api/server/getMigration')}>
                {t('pages.index.migrationDownload')}
              </Btn>
            ),
          },
        ]
      : []),
    {
      icon: 'refresh' as const,
      title: t('pages.index.importDatabase'),
      desc: isPostgres
        ? t('pages.index.importDatabasePgDesc')
        : t('pages.index.importDatabaseDesc'),
      action: (
        <Btn size="sm" icon="refresh" onClick={restore}>
          {t('pages.index.importDatabase')}
        </Btn>
      ),
    },
  ];

  return (
    <section className="tile">
      <TileHead icon="server" title={t('pages.index.backupTitle')} />
      {isPostgres && (
        <p className="mut" style={{ margin: '0 0 12px', fontSize: 13 }}>
          {t('pages.index.backupPostgresNote')}
        </p>
      )}
      <div className="list">
        {rows.map((r) => (
          <div className="li wrap" key={r.title}>
            <span className="ico acc">
              <Icon name={r.icon} />
            </span>
            <div className="f1" style={{ minWidth: 200 }}>
              <b>{r.title}</b>
              <div className="mut" style={{ fontSize: 12.5 }}>
                {r.desc}
              </div>
            </div>
            {r.action}
          </div>
        ))}
        <div className="li">
          <div className="f1">
            <b>{t('pages.index.importKeepHostSettings')}</b>
            <div className="mut" style={{ fontSize: 12.5 }}>
              {t('pages.index.importKeepHostSettingsDesc')}
            </div>
          </div>
          <Toggle
            on={keepHost}
            label={t('pages.index.importKeepHostSettings')}
            onChange={setKeepHost}
          />
        </div>
      </div>
      <BusyDialog busy={!!busy} tip={busy} />
    </section>
  );
}
