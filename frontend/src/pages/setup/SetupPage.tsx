import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router';
import { useQueryClient } from '@tanstack/react-query';

import { keys } from '@/api/queryKeys';
import Icon from '@/components/dl/Icon';
import Mark from '@/components/dl/Mark';
import { FlagDot } from '@/components/dl/primitives';
import { countryName } from '@/pages/xray/outbounds/outbounds-tab-helpers';
import { TOR_COUNTRIES } from '@/pages/xray/overrides/torCountries';
import { HttpUtil } from '@/utils';

const PROFILE_URL = '/panel/api/setting/profile';

// First run: name this server and say where it is. Nothing here is required and leaving skips it.
export default function SetupPage() {
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [name, setName] = useState(() => window.location.hostname);
  const [country, setCountry] = useState('');
  const [address, setAddress] = useState(() => window.location.hostname);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const countries = TOR_COUNTRIES.map((c) => c.toUpperCase())
    .map((code) => ({ code, label: countryName(code, i18n.language) }))
    .sort((a, b) => a.label.localeCompare(b.label));

  // The dashboard redirects here until it sees setupDone, so the refetch must land before leaving.
  async function save(profile: Record<string, string | boolean>) {
    setBusy(true);
    setError('');
    try {
      const saved = await HttpUtil.post(PROFILE_URL, { ...profile, setupDone: true });
      if (!saved?.success) throw new Error(saved?.msg || t('somethingWentWrong'));
      await queryClient.refetchQueries({ queryKey: keys.settings.all() });
      navigate('/');
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }

  function saveProfile() {
    const profile: Record<string, string> = {};
    if (name.trim()) profile.serverName = name.trim();
    if (country) profile.serverCountry = country;
    if (address.trim()) profile.serverAddress = address.trim();
    return save(profile);
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
        <div className="row" style={{ gap: 10 }}>
          <Mark size={34} motion="anim" />
          <b className="wm">Dalan</b>
        </div>
        <section className="tile edge" style={{ padding: 28, borderRadius: 24 }}>
          <div className="col wz-step" style={{ gap: 16 }}>
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
                <span id="setup-address-label">{t('setup.server.address')}</span>
                <input
                  className="inp mono"
                  dir="ltr"
                  aria-labelledby="setup-address-label"
                  aria-describedby="setup-address-hint"
                  value={address}
                  maxLength={253}
                  onChange={(e) => setAddress(e.target.value)}
                />
                <small id="setup-address-hint">{t('setup.server.addressHint')}</small>
              </div>
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
          </div>
        </section>
        <div className="row sp">
          <button type="button" className="btn ghost" disabled={busy} onClick={() => save({})}>
            {t('setup.later')}
          </button>
          <button
            type="button"
            className="btn pri lg"
            disabled={busy}
            aria-busy={busy}
            onClick={saveProfile}
          >
            {t('save')}
            <Icon name="chev" className="flipx" />
          </button>
        </div>
      </div>
    </div>
  );
}
