import { useEffect, useMemo, useState, type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { useLocation, useNavigate } from 'react-router';
import { message } from 'antd';

import { HttpUtil, PromiseUtil } from '@/utils';
import { setMessageInstance } from '@/utils/messageBus';
import { useAllSettings } from '@/api/queries/useAllSettings';
import { AllSettingSchema } from '@/schemas/setting';
import GeneralTab from './GeneralTab';
import SecurityTab from './SecurityTab';
import TelegramTab from './TelegramTab';
import EmailTab from './EmailTab';
import DiscordTab from './DiscordTab';
import SubscriptionGeneralTab from './SubscriptionGeneralTab';
import SubscriptionFormatsTab from './SubscriptionFormatsTab';
import SubscriptionBalancersTab from './SubscriptionBalancersTab';
import AppearanceSettings from './AppearanceSettings';
import BackupsSection from './maintenance/BackupsSection';
import UpdatesSection from './maintenance/UpdatesSection';
import Icon from '@/components/dl/Icon';
import { Modal as DlModal } from '@/components/dl/overlays';
import { Btn, IconBtn, Seg, Sk, TileHead } from '@/components/dl/primitives';
import './SettingsPage.css';

interface ApiMsg {
  success?: boolean;
}

const SECTIONS = [
  { slug: 'general', icon: 'settings', key: 'settingsPage.nav.general' },
  { slug: 'appearance', icon: 'fx', key: 'settingsPage.nav.appearance' },
  { slug: 'subscription', icon: 'link', key: 'settingsPage.nav.subscription' },
  { slug: 'security', icon: 'lock', key: 'settingsPage.nav.security' },
  { slug: 'backups', icon: 'server', key: 'settingsPage.nav.backups' },
  { slug: 'notifications', icon: 'bell', key: 'settingsPage.nav.notifications' },
  { slug: 'developer', icon: 'code', key: 'settingsPage.nav.developer' },
  { slug: 'updates', icon: 'refresh', key: 'settingsPage.nav.updates' },
] as const;
const OLD_SLUGS: Record<string, string> = {
  telegram: 'notifications',
  email: 'notifications',
  discord: 'notifications',
  'subscription-formats': 'subscription',
  'subscription-balancers': 'subscription',
};

function isIp(h: string): boolean {
  if (typeof h !== 'string') return false;
  const v4 = h.split('.');
  if (v4.length === 4 && v4.every((p) => /^\d{1,3}$/.test(p) && Number(p) <= 255)) return true;
  if (!h.includes(':') || h.includes(':::')) return false;
  const parts = h.split('::');
  if (parts.length > 2) return false;
  const split = (s: string) => (s ? s.split(':').filter(Boolean) : []);
  const head = split(parts[0]);
  const tail = split(parts[1]);
  const valid = (seg: string) => /^[0-9a-fA-F]{1,4}$/.test(seg);
  if (![...head, ...tail].every(valid)) return false;
  const groups = head.length + tail.length;
  return parts.length === 2 ? groups < 8 : groups === 8;
}

export default function SettingsPage() {
  const { t } = useTranslation();
  const [messageApi, messageContextHolder] = message.useMessage();

  useEffect(() => {
    setMessageInstance(messageApi);
  }, [messageApi]);

  const {
    allSetting,
    updateSetting,
    fetched,
    spinning,
    setSpinning,
    saveDisabled,
    saveAll,
    savePayload,
  } = useAllSettings();

  const [entryHost] = useState(() => window.location.hostname);
  const [entryPort] = useState(() => window.location.port);
  const [entryIsIP] = useState(() => isIp(window.location.hostname));

  const [alertVisible, setAlertVisible] = useState(true);
  const location = useLocation();
  const navigate = useNavigate();
  const hash = location.hash.replace(/^#/, '');
  const slug = OLD_SLUGS[hash] ?? hash;
  const activeSlug = SECTIONS.some((x) => x.slug === slug) ? slug : 'general';
  const [subTab, setSubTab] = useState<'general' | 'formats' | 'balancers'>(
    hash === 'subscription-formats'
      ? 'formats'
      : hash === 'subscription-balancers'
        ? 'balancers'
        : 'general',
  );
  const [notifyTab, setNotifyTab] = useState<'telegram' | 'email' | 'discord'>(
    hash === 'email' || hash === 'discord' ? hash : 'telegram',
  );

  function rebuildUrlAfterRestart(): string {
    const { webDomain, webPort, webBasePath, webCertFile, webKeyFile } = allSetting;
    const newProtocol = webCertFile || webKeyFile ? 'https:' : 'http:';

    let base = webBasePath ? webBasePath.replace(/^\//, '') : '';
    if (base && !base.endsWith('/')) base += '/';

    if (!entryIsIP) {
      const url = new URL(window.location.href);
      url.pathname = `/${base}panel/settings`;
      url.protocol = newProtocol;
      return url.toString();
    }

    let finalHost = entryHost;
    let finalPort = entryPort || '';
    if (webDomain && isIp(webDomain)) finalHost = webDomain;
    if (webPort && Number(webPort) !== Number(entryPort)) finalPort = String(webPort);

    const url = new URL(`${newProtocol}//${finalHost}`);
    if (finalPort) url.port = finalPort;
    url.pathname = `/${base}panel/settings`;
    return url.toString();
  }

  async function onSave() {
    const result = AllSettingSchema.safeParse(allSetting);
    if (!result.success) {
      const issue = result.error.issues[0];
      const fieldPath = issue?.path.join('.') ?? 'value';
      const msgKey = issue?.message ?? 'somethingWentWrong';
      messageApi.error(`${fieldPath}: ${t(msgKey, { defaultValue: msgKey })}`);
      return;
    }
    await saveAll();
  }

  const [restartOpen, setRestartOpen] = useState(false);
  async function restartPanel() {
    setRestartOpen(false);
    setSpinning(true);
    try {
      const msg = (await HttpUtil.post('/panel/api/setting/restartPanel')) as ApiMsg;
      if (!msg?.success) return;
      await PromiseUtil.sleep(5000);
      window.location.replace(rebuildUrlAfterRestart());
    } finally {
      setSpinning(false);
    }
  }

  const confAlerts = useMemo<string[]>(() => {
    const out: string[] = [];
    if (window.location.protocol !== 'https:') {
      out.push(t('pages.settings.warnHttp'));
    }
    if (allSetting.webPort === 2053) {
      out.push(t('pages.settings.warnDefaultPort'));
    }
    const segs = window.location.pathname.split('/').length < 4;
    if (segs && allSetting.webBasePath === '/') {
      out.push(t('pages.settings.warnDefaultBasePath'));
    }
    if (allSetting.subEnable) {
      let subPath = allSetting.subPath;
      if (allSetting.subURI) {
        try {
          subPath = new URL(allSetting.subURI).pathname;
        } catch {
          /* noop */
        }
      }
      if (subPath === '/sub/') {
        out.push(t('pages.settings.warnDefaultSubPath'));
      }
    }
    if (allSetting.subJsonEnable) {
      let p = allSetting.subJsonPath;
      if (allSetting.subJsonURI) {
        try {
          p = new URL(allSetting.subJsonURI).pathname;
        } catch {
          /* noop */
        }
      }
      if (p === '/json/') {
        out.push(t('pages.settings.warnDefaultJsonPath'));
      }
    }
    return out;
  }, [allSetting, t]);

  const tile = (body: ReactNode) => <section className="tile">{body}</section>;
  const categoryBody = (() => {
    switch (activeSlug) {
      case 'appearance':
        return <AppearanceSettings />;
      case 'security':
        return tile(
          <SecurityTab
            allSetting={allSetting}
            updateSetting={updateSetting}
            saveSetting={savePayload}
          />,
        );
      case 'notifications':
        return tile(
          <>
            <Seg
              label={t('settingsPage.nav.notifications')}
              value={notifyTab}
              onChange={setNotifyTab}
              options={[
                { value: 'telegram', label: 'Telegram' },
                { value: 'email', label: t('settingsPage.email') },
                { value: 'discord', label: 'Discord' },
              ]}
            />
            <div style={{ marginTop: 14 }}>
              {notifyTab === 'telegram' ? (
                <TelegramTab allSetting={allSetting} updateSetting={updateSetting} />
              ) : notifyTab === 'email' ? (
                <EmailTab allSetting={allSetting} updateSetting={updateSetting} />
              ) : (
                <DiscordTab allSetting={allSetting} updateSetting={updateSetting} />
              )}
            </div>
          </>,
        );
      case 'subscription':
        return tile(
          <>
            <Seg
              label={t('settingsPage.nav.subscription')}
              value={subTab}
              onChange={setSubTab}
              options={[
                { value: 'general', label: t('settingsPage.nav.general') },
                { value: 'formats', label: t('settingsPage.formats') },
                { value: 'balancers', label: t('settingsPage.balancers') },
              ]}
            />
            <div style={{ marginTop: 14 }}>
              {subTab === 'general' ? (
                <SubscriptionGeneralTab allSetting={allSetting} updateSetting={updateSetting} />
              ) : subTab === 'formats' ? (
                <SubscriptionFormatsTab allSetting={allSetting} updateSetting={updateSetting} />
              ) : (
                <SubscriptionBalancersTab allSetting={allSetting} updateSetting={updateSetting} />
              )}
            </div>
          </>,
        );
      case 'backups':
        return <BackupsSection />;
      case 'updates':
        return <UpdatesSection />;
      case 'developer':
        return (
          <section className="tile">
            <TileHead icon="code" title={t('settingsPage.nav.developer')} />
            <p className="mut" style={{ margin: '0 0 12px' }}>
              {t('settingsPage.developerHint')}
            </p>
            <div className="row wrap" style={{ gap: 8 }}>
              <Btn kind="pri" icon="code" onClick={() => navigate('/api-docs')}>
                {t('menu.apiDocs')}
              </Btn>
              <Btn icon="lock" onClick={() => navigate('/settings#security')}>
                {t('pages.nodes.apiToken')}
              </Btn>
            </div>
          </section>
        );
      default:
        return tile(<GeneralTab allSetting={allSetting} updateSetting={updateSetting} />);
    }
  })();

  return (
    <div className="col stag" style={{ gap: 18 }}>
      {messageContextHolder}
      <div className="pagehead">
        <div>
          <h2 className="h2" style={{ fontSize: 28 }}>
            {t('menu.settings')}
          </h2>
          <p>{t('settingsPage.subtitle')}</p>
        </div>
        <div className="row wrap">
          <Btn
            kind="dng"
            icon="power"
            disabled={!saveDisabled || spinning}
            onClick={() => setRestartOpen(true)}
          >
            {t('pages.settings.restartPanel')}
          </Btn>
          <Btn kind="pri" icon="check" disabled={saveDisabled || spinning} onClick={onSave}>
            {t('pages.settings.save')}
          </Btn>
        </div>
      </div>
      {confAlerts.length > 0 && alertVisible && (
        <section className="tile edge st-bad" role="alert">
          <div className="row sp" style={{ alignItems: 'flex-start' }}>
            <div className="row" style={{ alignItems: 'flex-start' }}>
              <span className="ico bad">
                <Icon name="alert" />
              </span>
              <div>
                <b>{t('pages.settings.securityWarnings')}</b>
                <div className="mut" style={{ fontSize: 13 }}>
                  {t('pages.settings.panelExposed')}
                </div>
                <ul style={{ margin: '6px 0 0', paddingInlineStart: 18 }}>
                  {confAlerts.map((msg, i) => (
                    <li key={i}>{msg}</li>
                  ))}
                </ul>
              </div>
            </div>
            <IconBtn small icon="x" label={t('close')} onClick={() => setAlertVisible(false)} />
          </div>
        </section>
      )}
      {!saveDisabled && (
        <div
          className="tile edge st-warn row"
          role="status"
          style={{ padding: '10px 14px', gap: 10 }}
        >
          <Icon name="info" size={16} />
          <span className="f1" style={{ fontWeight: 650 }}>
            {t('pages.settings.infoDesc')}
          </span>
        </div>
      )}
      <div className="grid">
        <nav
          className="tile s3"
          style={{ padding: 10, alignSelf: 'start' }}
          aria-label={t('settingsPage.sections')}
        >
          {SECTIONS.map((x) => (
            <a
              key={x.slug}
              className={`nav${activeSlug === x.slug ? ' on' : ''}`}
              href={`#${x.slug}`}
              aria-current={activeSlug === x.slug ? 'page' : undefined}
              onClick={(e) => {
                e.preventDefault();
                navigate(`/settings#${x.slug}`);
              }}
            >
              <Icon name={x.icon} />
              <span className="nl">{t(x.key)}</span>
            </a>
          ))}
        </nav>
        <div className="s9 col" style={{ gap: 18 }}>
          {!fetched ? (
            <section className="tile">
              <Sk h={260} />
            </section>
          ) : (
            categoryBody
          )}
        </div>
      </div>
      <DlModal
        open={restartOpen}
        onClose={() => setRestartOpen(false)}
        label={t('pages.settings.restartPanel')}
        width={460}
      >
        <h2 className="h3">{t('pages.settings.restartPanel')}</h2>
        <p className="mut" style={{ margin: 0 }}>
          {t('pages.settings.restartPanelDesc')}
        </p>
        <div className="mfoot">
          <Btn kind="ghost" onClick={() => setRestartOpen(false)}>
            {t('cancel')}
          </Btn>
          <Btn kind="dng" icon="power" onClick={restartPanel}>
            {t('pages.settings.restartPanel')}
          </Btn>
        </div>
      </DlModal>
    </div>
  );
}
