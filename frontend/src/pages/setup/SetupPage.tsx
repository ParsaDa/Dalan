import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router';
import { useQueryClient } from '@tanstack/react-query';
import { QRCode } from 'antd';

import { useAllSettings } from '@/api/queries/useAllSettings';
import Icon from '@/components/dl/Icon';
import Mark from '@/components/dl/Mark';
import { fmtInt } from '@/components/dl/numbers';
import { FlagDot } from '@/components/dl/primitives';
import { formValuesToWirePayload } from '@/lib/xray/inbound-form-adapter';
import { countryName } from '@/pages/xray/outbounds/outbounds-tab-helpers';
import { TOR_COUNTRIES } from '@/pages/xray/overrides/torCountries';
import { ClipboardManager, HttpUtil } from '@/utils';
import { buildPresetInbound, type Preset } from './presets';

const GB = 1024 ** 3;
const DAY = 86_400_000;
const PRESETS: { id: Preset | 'game'; rec?: boolean }[] = [
  { id: 'reality', rec: true },
  { id: 'cdn' },
  { id: 'game' },
];

interface Done {
  email: string;
  link: string;
}

// The first-run wizard: name this server, pick how people connect, add the first user.
export default function SetupPage() {
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { allSetting, savePayload } = useAllSettings();
  const [step, setStep] = useState(0);
  const [name, setName] = useState(() => window.location.hostname);
  const [country, setCountry] = useState('');
  const [preset, setPreset] = useState<Preset>('reality');
  const [user, setUser] = useState('');
  const [traffic, setTraffic] = useState<'50' | '0'>('50');
  const [days, setDays] = useState<'30' | '0'>('30');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [done, setDone] = useState<Done | null>(null);
  const [copied, setCopied] = useState(false);
  const steps = [t('setup.step.server'), t('setup.step.connection'), t('setup.step.user')];
  const countries = TOR_COUNTRIES.map((c) => c.toUpperCase())
    .map((code) => ({ code, label: countryName(code, i18n.language) }))
    .sort((a, b) => a.label.localeCompare(b.label));

  async function finish() {
    setBusy(true);
    setError('');
    try {
      const email = user.trim() || 'user1';
      let keys: { privateKey: string; publicKey: string } | undefined;
      if (preset === 'reality') {
        const k = await HttpUtil.get<{ privateKey: string; publicKey: string }>(
          '/panel/api/server/getNewX25519Cert',
        );
        if (!k?.success || !k.obj) throw new Error(k?.msg || t('setup.keysFailed'));
        keys = k.obj;
      }
      const values = buildPresetInbound(preset, {
        email,
        keys,
        totalGB: traffic === '0' ? 0 : Number(traffic) * GB,
        expiryTime: days === '0' ? 0 : Date.now() + Number(days) * DAY,
      });
      const added = await HttpUtil.post(
        '/panel/api/inbounds/add',
        formValuesToWirePayload(values, { omitClients: false }),
      );
      if (!added?.success) throw new Error(added?.msg || t('somethingWentWrong'));
      const saved = await savePayload({
        ...allSetting,
        serverName: name.trim(),
        serverCountry: country,
        setupDone: true,
      });
      if (!(saved as { success?: boolean } | undefined)?.success)
        throw new Error(t('somethingWentWrong'));
      const defaults = await HttpUtil.post<{ subEnable?: boolean; subURI?: string }>(
        '/panel/api/setting/defaultSettings',
      );
      const subId = (values.settings as { clients: { subId?: string }[] }).clients[0]?.subId ?? '';
      const link =
        defaults?.obj?.subEnable && defaults.obj.subURI && subId ? defaults.obj.subURI + subId : '';
      queryClient.invalidateQueries();
      setDone({ email, link });
      setStep(3);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div
      style={{ minHeight: '100vh', display: 'grid', placeItems: 'center', padding: '40px 20px' }}
    >
      <div className="bgfx" aria-hidden="true">
        <div className="glow g1" />
        <div className="glow g2" />
      </div>
      <div className="col" style={{ width: 'min(720px, 100%)', gap: 22, position: 'relative' }}>
        <div className="row sp wrap">
          <div className="row" style={{ gap: 10 }}>
            <Mark size={34} motion="anim" />
            <b className="wm">Dalan</b>
          </div>
          <span className="mut" style={{ fontWeight: 650 }}>
            {t('setup.time')}
          </span>
        </div>
        <ol
          className="stepper"
          aria-label={t('setup.progress')}
          style={{ listStyle: 'none', margin: 0, padding: 0 }}
        >
          {steps.map((label, i) => (
            <li key={label} style={{ display: 'contents' }}>
              <div
                className={`st${i < step ? ' done' : i === step ? ' on' : ''}`}
                aria-current={i === step ? 'step' : undefined}
              >
                <i>{i < step ? <Icon name="check" size={14} /> : fmtInt(i + 1)}</i>
                <span>{label}</span>
              </div>
              {i < steps.length - 1 && <div className={`ln${i < step ? ' done' : ''}`} />}
            </li>
          ))}
        </ol>
        <section className="tile edge" style={{ padding: 28, borderRadius: 24 }}>
          <div key={step} className="col wz-step" style={{ gap: 16 }}>
            {step === 0 && (
              <>
                <div>
                  <h1 className="h2" style={{ fontSize: 26 }}>
                    {t('setup.server.title')}
                  </h1>
                  <p className="mut" style={{ margin: '6px 0 0' }}>
                    {t('setup.server.hint')}
                  </p>
                </div>
                <div className="grid" style={{ gap: 14 }}>
                  <label className="field s6">
                    <span>{t('setup.server.name')}</span>
                    <input
                      className="inp"
                      value={name}
                      maxLength={64}
                      onChange={(e) => setName(e.target.value)}
                    />
                  </label>
                  <label className="field s6">
                    <span>{t('setup.server.country')}</span>
                    <div className="row" style={{ gap: 10 }}>
                      <FlagDot country={country} size={22} />
                      <select
                        className="inp f1"
                        value={country}
                        onChange={(e) => setCountry(e.target.value)}
                      >
                        <option value="">{t('setup.server.countryNone')}</option>
                        {countries.map((c) => (
                          <option key={c.code} value={c.code}>
                            {c.label}
                          </option>
                        ))}
                      </select>
                    </div>
                  </label>
                  <div className="field s12">
                    <span>{t('setup.server.address')}</span>
                    <div className="inp mono" style={{ display: 'flex', alignItems: 'center' }}>
                      <bdi>{window.location.hostname}</bdi>
                    </div>
                    <small>{t('setup.server.addressHint')}</small>
                  </div>
                </div>
              </>
            )}
            {step === 1 && (
              <>
                <div>
                  <h1 className="h2" style={{ fontSize: 26 }}>
                    {t('setup.connect.title')}
                  </h1>
                  <p className="mut" style={{ margin: '6px 0 0' }}>
                    {t('setup.connect.hint')}
                  </p>
                </div>
                <div
                  className="col"
                  style={{ gap: 10 }}
                  role="radiogroup"
                  aria-label={t('setup.connect.title')}
                >
                  {PRESETS.map((p) => {
                    const on = p.id === preset;
                    const disabled = p.id === 'game';
                    return (
                      <button
                        key={p.id}
                        type="button"
                        role="radio"
                        aria-checked={on}
                        disabled={disabled}
                        className={`tile${on ? ' edge' : ''}`}
                        onClick={() => !disabled && setPreset(p.id as Preset)}
                        style={{
                          textAlign: 'start',
                          font: 'inherit',
                          color: 'inherit',
                          cursor: disabled ? 'not-allowed' : 'pointer',
                          display: 'flex',
                          gap: 12,
                          alignItems: 'flex-start',
                          opacity: disabled ? 0.6 : 1,
                          outline: on ? '2px solid var(--acc-line)' : undefined,
                          outlineOffset: -1,
                        }}
                      >
                        <span className="ico acc">
                          <Icon name="inbound" />
                        </span>
                        <span className="f1">
                          <b style={{ display: 'block' }}>{t(`setup.preset.${p.id}.title`)}</b>
                          <span className="mut" style={{ fontSize: 13 }}>
                            {t(`setup.preset.${p.id}.desc`)}
                          </span>
                        </span>
                        {p.rec && <span className="badge acc">{t('subPage.recommended')}</span>}
                      </button>
                    );
                  })}
                </div>
              </>
            )}
            {step === 2 && (
              <>
                <div>
                  <h1 className="h2" style={{ fontSize: 26 }}>
                    {t('setup.user.title')}
                  </h1>
                  <p className="mut" style={{ margin: '6px 0 0' }}>
                    {t('setup.user.hint')}
                  </p>
                </div>
                <div className="grid" style={{ gap: 14 }}>
                  <label className="field s12">
                    <span>{t('setup.user.name')}</span>
                    <input
                      className="inp"
                      placeholder={t('setup.user.placeholder')}
                      value={user}
                      onChange={(e) => setUser(e.target.value)}
                    />
                  </label>
                  <label className="field s6">
                    <span>{t('setup.user.traffic')}</span>
                    <select
                      className="inp"
                      value={traffic}
                      onChange={(e) => setTraffic(e.target.value as '50' | '0')}
                    >
                      <option value="50">{t('setup.user.gb50')}</option>
                      <option value="0">{t('clientsPage.unlimited')}</option>
                    </select>
                  </label>
                  <label className="field s6">
                    <span>{t('setup.user.valid')}</span>
                    <select
                      className="inp"
                      value={days}
                      onChange={(e) => setDays(e.target.value as '30' | '0')}
                    >
                      <option value="30">{t('setup.user.days30')}</option>
                      <option value="0">{t('setup.user.never')}</option>
                    </select>
                  </label>
                </div>
                {error && (
                  <div
                    className="tile edge st-bad row"
                    role="alert"
                    style={{ padding: '10px 12px', gap: 10 }}
                  >
                    <Icon name="alert" size={16} />
                    <span className="f1">{error}</span>
                  </div>
                )}
              </>
            )}
            {step === 3 && done && (
              <div className="col" style={{ gap: 18, alignItems: 'center', textAlign: 'center' }}>
                <Mark size={70} motion="anim" />
                <div>
                  <h1 className="h2" style={{ fontSize: 28 }}>
                    {t('setup.live.title')}
                  </h1>
                  <p className="mut" style={{ margin: '6px 0 0' }}>
                    {done.link
                      ? t('setup.live.hint', { name: done.email })
                      : t('setup.live.noSub', { name: done.email })}
                  </p>
                </div>
                {done.link && (
                  <>
                    <div style={{ background: '#fff', borderRadius: 20, padding: 14 }}>
                      <QRCode
                        value={done.link}
                        size={170}
                        bordered={false}
                        color="#000"
                        bgColor="#fff"
                      />
                    </div>
                    <div className="row" style={{ gap: 8, width: '100%' }}>
                      <div
                        className="inp mono f1"
                        style={{
                          display: 'flex',
                          alignItems: 'center',
                          overflow: 'hidden',
                          whiteSpace: 'nowrap',
                          fontSize: 12,
                        }}
                      >
                        <bdi>{done.link}</bdi>
                      </div>
                      <button
                        type="button"
                        className="btn pri"
                        onClick={async () => setCopied(await ClipboardManager.copyText(done.link))}
                      >
                        <Icon name={copied ? 'check' : 'copy'} />
                        {copied ? t('subPage.copied') : t('subPage.copy')}
                      </button>
                    </div>
                  </>
                )}
                <button type="button" className="btn" onClick={() => navigate('/')}>
                  <Icon name="dash" />
                  {t('errorPage.home')}
                </button>
              </div>
            )}
          </div>
        </section>
        {step < 3 && (
          <div className="row sp">
            <button
              type="button"
              className="btn ghost"
              disabled={step === 0 || busy}
              onClick={() => setStep(step - 1)}
            >
              <Icon name="back" className="flipx" />
              {t('setup.back')}
            </button>
            <div className="row" style={{ gap: 8 }}>
              {step === 0 && (
                <button
                  type="button"
                  className="btn ghost"
                  disabled={busy}
                  onClick={async () => {
                    await savePayload({ ...allSetting, setupDone: true });
                    navigate('/');
                  }}
                >
                  {t('setup.later')}
                </button>
              )}
              <button
                type="button"
                className="btn pri lg"
                disabled={busy || (step === 0 && !name.trim())}
                aria-busy={busy}
                onClick={() => (step === 2 ? finish() : setStep(step + 1))}
              >
                {step === 2 ? t('setup.finish') : t('setup.next')}
                <Icon name="chev" className="flipx" />
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
