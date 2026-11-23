import { useCallback, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { QRCode } from 'antd';

import { ClipboardManager, IntlUtil, LanguageManager, SizeFormatter } from '@/utils';
import { useTheme } from '@/hooks/useTheme';
import Icon from '@/components/dl/Icon';
import Mark from '@/components/dl/Mark';
import { DropChip } from '@/components/dl/DropChip';
import { Bar, IconBtn, Pill, Ring, TileHead } from '@/components/dl/primitives';
import { useNum } from '@/components/dl/numbers';
import { APP_ICONS } from './appIcons';
import { isRtlLanguage } from '@/i18n/direction';
import {
  buildSubApps,
  daysUntil,
  detectPlatform,
  resolveSubStatus,
  usagePercent,
  type AppPlatform,
} from './subPageModel';

const subData = window.__SUB_PAGE_DATA__ || {};

const sId = subData.sId || '';
const subUrl = subData.subUrl || '';
const subJsonUrl = subData.subJsonUrl || '';
const subClashUrl = subData.subClashUrl || '';
const subTitle = subData.subTitle || '';
const subSupportUrl = subData.subSupportUrl || '';
const updateHours = Number(subData.subUpdates || 0);
const announce = subData.announce || '';
const links: string[] = Array.isArray(subData.links) ? subData.links : [];
const linkEmails: string[] = Array.isArray(subData.emails) ? subData.emails : [];
const totalByte = Number(subData.totalByte || 0);
const usedByte =
  Number(subData.usedByte || 0) ||
  Number(subData.downloadByte || 0) + Number(subData.uploadByte || 0);
const expireMs = Number(subData.expire || 0) * 1000;
const clientEmail = [...new Set(linkEmails.filter(Boolean))].join(', ');
const loadedAt = Date.now();

const heroData = {
  status: resolveSubStatus({ enabled: !!subData.enabled, usedByte, totalByte, expireMs }, loadedAt),
  daysLeft: daysUntil(expireMs, loadedAt),
  usedByte,
  totalByte,
  expireMs,
  lastOnlineMs: Number(subData.lastOnline || 0),
  download: subData.download || '0',
  upload: subData.upload || '0',
  used: subData.used || '0',
  total: subData.total || '∞',
  remained: subData.remained || '',
  datepicker: subData.datepicker || 'gregorian',
};

const apps = buildSubApps({ subUrl, sId, subTitle });
const initialPlatform = detectPlatform(navigator.userAgent);

export default function SubPage() {
  const { t } = useTranslation();
  const num = useNum();
  const { isDark, toggleTheme } = useTheme();
  const [lang, setLang] = useState<string>(() => LanguageManager.getLanguage('subscription'));
  const onLangChange = useCallback((next: string) => {
    setLang(next);
    LanguageManager.setLanguage(next, 'subscription');
  }, []);
  const [copied, setCopied] = useState('');
  const [showQr, setShowQr] = useState(false);
  const [platform, setPlatform] = useState<AppPlatform>(initialPlatform);
  const copy = useCallback(async (value: string) => {
    if (value && (await ClipboardManager.copyText(value))) {
      setCopied(value);
      window.setTimeout(() => setCopied((c) => (c === value ? '' : c)), 3000);
    }
  }, []);
  const dir = isRtlLanguage(lang) ? 'rtl' : 'ltr';
  const status = heroData.status;
  const days = heroData.daysLeft;
  const used = usagePercent(usedByte, totalByte) / 100;
  const firstApp = subUrl ? apps[platform][0] : undefined;
  const statusTone =
    status === 'active' || status === 'unlimited' ? 'ok' : status === 'disabled' ? 'off' : 'bad';
  const languages: readonly { value: string; name: string }[] = LanguageManager.supportedLanguages;

  return (
    <div
      dir={dir}
      style={{
        minHeight: '100vh',
        display: 'grid',
        placeItems: 'start center',
        padding: '28px 16px 48px',
      }}
    >
      <div className="bgfx" aria-hidden="true">
        <div className="glow g1" />
        <div className="glow g2" />
      </div>
      <div
        className="stag"
        style={{
          width: 'min(980px, 100%)',
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fit, minmax(min(100%, 380px), 1fr))',
          gap: 22,
          position: 'relative',
        }}
      >
        <div className="col" style={{ gap: 16 }}>
          <div className="row sp">
            <div className="row" style={{ gap: 10, minWidth: 0 }}>
              <span
                style={{
                  width: 38,
                  height: 38,
                  borderRadius: 12,
                  display: 'grid',
                  placeItems: 'center',
                  background: 'var(--glass)',
                  border: '1px solid var(--glass-b)',
                  flex: 'none',
                }}
              >
                <Mark size={26} />
              </span>
              <div style={{ minWidth: 0 }}>
                <div style={{ fontWeight: 800, overflowWrap: 'anywhere' }}>
                  {subTitle || 'Dalan'}
                </div>
                <div className="mut mono" style={{ fontSize: 12 }}>
                  {sId}
                </div>
              </div>
            </div>
            <div className="row" style={{ gap: 6 }}>
              <DropChip
                label={t('pages.settings.language')}
                value={lang}
                onChange={onLangChange}
                options={languages.map((l) => ({ value: l.value, label: l.name }))}
              />
              <button
                type="button"
                className="ib sm"
                aria-label={t('menu.theme')}
                onClick={toggleTheme}
              >
                <Icon name={isDark ? 'moon' : 'sun'} />
              </button>
            </div>
          </div>
          <div>
            <h1 className="h1" style={{ fontSize: 28, overflowWrap: 'anywhere' }}>
              {clientEmail ? t('subPage.hi', { name: clientEmail }) : t('subscription.title')}
            </h1>
            <div className="row" style={{ gap: 8, marginTop: 6 }}>
              <Pill status={statusTone}>{t(`subPage.status.${status}`)}</Pill>
              <span className="mut" style={{ fontWeight: 650 }}>
                {statusTone === 'ok' ? t('subPage.ready') : t('subPage.notReady')}
              </span>
            </div>
          </div>
          {announce && (
            <div className="tile edge row" role="note" style={{ padding: '12px 14px', gap: 10 }}>
              <Icon name="info" size={16} />
              <span className="f1">{announce}</span>
            </div>
          )}
          <section
            className="tile edge"
            style={{ display: 'flex', alignItems: 'center', gap: 18, flexWrap: 'wrap' }}
          >
            <Ring value={days === null ? 1 : Math.min(1, days / 30)}>
              <div>
                <div className="kpi num" style={{ fontSize: 38 }}>
                  {days === null ? '∞' : num.int(Math.max(0, days))}
                </div>
                <div className="lbl">{t('subscription.daysLeft')}</div>
              </div>
            </Ring>
            <div className="col f1" style={{ gap: 8, minWidth: 180 }}>
              <div>
                <div className="kpi num" style={{ fontSize: 26 }}>
                  <bdi>{SizeFormatter.sizeFormat(usedByte)}</bdi>
                </div>
                <div className="mut" style={{ fontSize: 13, fontWeight: 650 }}>
                  {totalByte > 0
                    ? t('subPage.usedOf', { total: SizeFormatter.sizeFormat(totalByte) })
                    : t('subscription.unlimited')}
                </div>
              </div>
              <Bar
                value={totalByte > 0 ? used : 0.06}
                large
                tone={used >= 1 ? 'bad' : used >= 0.85 ? 'warn' : undefined}
              />
              <div className="mut" style={{ fontSize: 12.5, fontWeight: 650 }}>
                {expireMs > 0
                  ? t('subPage.until', { date: IntlUtil.formatDate(expireMs, heroData.datepicker) })
                  : t('subscription.noExpiry')}
              </div>
            </div>
          </section>
          {firstApp && (
            <a className="btn pri lg full" href={firstApp.url} style={{ height: 60, fontSize: 17 }}>
              <Icon name="power" />
              {t('subPage.connect', { app: firstApp.name })}
            </a>
          )}
          {subUrl && (
            <div className="row" style={{ gap: 10 }}>
              <button
                type="button"
                className={`btn lg${copied === subUrl ? ' done' : ''}`}
                style={{ flex: 1 }}
                onClick={() => copy(subUrl)}
              >
                <Icon name={copied === subUrl ? 'check' : 'copy'} />
                {copied === subUrl ? t('subPage.copied') : t('subPage.copy')}
              </button>
              <button
                type="button"
                className="btn lg"
                style={{ flex: 1 }}
                aria-expanded={showQr}
                onClick={() => setShowQr(!showQr)}
              >
                <Icon name="qr" />
                {showQr ? t('subPage.hideQr') : t('subPage.showQr')}
              </button>
            </div>
          )}
          {copied === subUrl && subUrl && (
            <div
              className="mut"
              role="status"
              style={{ fontSize: 12.5, fontWeight: 650, textAlign: 'center' }}
            >
              {t('subPage.copiedHint')}
            </div>
          )}
          {showQr && subUrl && (
            <div className="qrbox" style={{ display: 'grid', placeItems: 'center' }}>
              <QRCode value={subUrl} size={220} bordered={false} color="#000" bgColor="#fff" />
            </div>
          )}
        </div>
        <div className="col" style={{ gap: 16 }}>
          {subUrl && (
            <section className="tile">
              <TileHead icon="app" title={t('subPage.help')} />
              <div className="lbl" style={{ marginBottom: 8 }}>
                {t('subPage.pick')}
              </div>
              <div className="row" style={{ gap: 6, marginBottom: 14 }}>
                {(['android', 'ios'] as const).map((p) => (
                  <button
                    key={p}
                    type="button"
                    className={`osbtn${platform === p ? ' on' : ''}`}
                    aria-pressed={platform === p}
                    onClick={() => setPlatform(p)}
                  >
                    {p === 'ios' ? 'iPhone' : 'Android'}
                  </button>
                ))}
              </div>
              <div className="li" style={{ border: 0, paddingTop: 0, alignItems: 'flex-start' }}>
                <span className="ico acc" style={{ fontWeight: 800 }}>
                  {num.int(1)}
                </span>
                <div className="f1">
                  <b>{t('subPage.step1')}</b>
                  <div className="row wrap" style={{ gap: 6, marginTop: 8 }}>
                    {apps[platform].map((app, i) => (
                      <a key={app.name} className="btn sm" href={app.url}>
                        {APP_ICONS[app.name] && (
                          <img
                            src={APP_ICONS[app.name].src}
                            alt=""
                            width={16}
                            height={16}
                            style={{ borderRadius: 4 }}
                          />
                        )}
                        {app.name}
                        {i === 0 && <span className="badge acc">{t('subPage.recommended')}</span>}
                      </a>
                    ))}
                  </div>
                </div>
              </div>
              <div className="li">
                <span className="ico acc" style={{ fontWeight: 800 }}>
                  {num.int(2)}
                </span>
                <b className="f1">{t('subPage.step2')}</b>
              </div>
              <div className="li">
                <span className="ico acc" style={{ fontWeight: 800 }}>
                  {num.int(3)}
                </span>
                <b className="f1">{t('subPage.step3')}</b>
              </div>
            </section>
          )}
          {(subJsonUrl || subClashUrl || links.length > 0) && (
            <details className="more-opts tile">
              <summary>{t('subPage.more')}</summary>
              <div className="list" style={{ marginTop: 10 }}>
                {[
                  ['JSON', subJsonUrl],
                  ['Clash', subClashUrl],
                ]
                  .filter(([, url]) => url)
                  .map(([label, url]) => (
                    <div className="li" key={label}>
                      <b className="f1">{label}</b>
                      <IconBtn
                        small
                        icon={copied === url ? 'check' : 'copy'}
                        label={`${t('subPage.copy')} ${label}`}
                        onClick={() => copy(url)}
                      />
                    </div>
                  ))}
                {links.map((link, i) => (
                  <div className="li" key={`${i}-${link.slice(0, 24)}`}>
                    <span
                      className="f1 mono mut"
                      style={{
                        fontSize: 12,
                        overflow: 'hidden',
                        textOverflow: 'ellipsis',
                        whiteSpace: 'nowrap',
                      }}
                    >
                      {decodeURIComponent(link.split('#')[1] || '') || link.split('://')[0]}
                    </span>
                    <IconBtn
                      small
                      icon={copied === link ? 'check' : 'copy'}
                      label={`${t('subPage.copy')} ${i + 1}`}
                      onClick={() => copy(link)}
                    />
                  </div>
                ))}
              </div>
            </details>
          )}
          <div className="mut" style={{ textAlign: 'center', fontSize: 12.5, fontWeight: 650 }}>
            {updateHours > 0 && (
              <div>{t('subscription.updateInterval', { hours: updateHours })}</div>
            )}
            {subSupportUrl ? (
              <a href={subSupportUrl} target="_blank" rel="noopener noreferrer">
                {t('subPage.need')}
              </a>
            ) : (
              t('subPage.need')
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
