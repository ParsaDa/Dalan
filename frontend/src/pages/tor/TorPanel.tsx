import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';

import EngineRoutePicker from '@/components/engine/EngineRoutePicker';
import Icon from '@/components/dl/Icon';
import { Btn, FlagDot, Pill, Toggle } from '@/components/dl/primitives';
import { engineRouteReady, PANEL_ROUTE, type EngineRoute } from '@/schemas/engineRoute';
import { useTorBridges, useTorEngine, useTorExits, useTorMutations } from '@/api/queries/useTor';
import { countryName } from '@/pages/xray/outbounds/outbounds-tab-helpers';
import { TOR_COUNTRIES } from '@/pages/xray/overrides/torCountries';

interface RowResult {
  ok: boolean;
  text: string;
}

// Tor exits, the engine and bridges; the Tor and PIA page shows it, and so does the Tor dialog.
export default function TorPanel({ onChanged }: { onChanged?: () => void }) {
  const { t } = useTranslation();
  const { exits } = useTorExits();
  const engine = useTorEngine(true).data;
  const bridgesQuery = useTorBridges(true);
  const { add, remove, test, installEngine, installing, saveBridges, savingBridges } =
    useTorMutations();
  const [search, setSearch] = useState('');
  const [testing, setTesting] = useState<string | null>(null);
  const [results, setResults] = useState<Record<string, RowResult>>({});
  const [error, setError] = useState('');
  const [bridgesDraft, setBridgesDraft] = useState<string | null>(null);
  const [bridgesError, setBridgesError] = useState('');
  const [bridgesSaved, setBridgesSaved] = useState(false);
  const [route, setRoute] = useState<EngineRoute>(PANEL_ROUTE);

  const added = useMemo(() => new Map(exits.map((e) => [e.country, e])), [exits]);
  const countries = useMemo(() => {
    const needle = search.trim().toLowerCase();
    return TOR_COUNTRIES.map((code) => ({ code, name: countryName(code, 'en') }))
      .filter((c) => !needle || c.name.toLowerCase().includes(needle) || c.code.includes(needle))
      .sort(
        (a, b) =>
          Number(added.has(b.code)) - Number(added.has(a.code)) || a.name.localeCompare(b.name),
      );
  }, [search, added]);

  async function change(run: () => Promise<{ success?: boolean; msg?: string }>) {
    setError('');
    const msg = await run();
    if (!msg?.success) setError(msg?.msg || t('pages.tor.failed'));
    else onChanged?.();
  }

  async function testCountry(code: string) {
    setTesting(code);
    const answer = await test(code);
    setTesting(null);
    let outcome: RowResult;
    if ('error' in answer) outcome = { ok: false, text: answer.error };
    else if (answer.result.ok) {
      outcome = {
        ok: true,
        text: `${t('pages.tor.result.works', { ms: answer.result.latencyMs })} · ${t('pages.tor.result.exit', { loc: answer.result.exitLoc })}`,
      };
    } else {
      outcome = {
        ok: false,
        text: [answer.result.reason, answer.result.error].filter(Boolean).join(' — '),
      };
    }
    setResults((prev) => ({ ...prev, [code]: outcome }));
  }

  async function saveBridgeLines() {
    setBridgesError('');
    setBridgesSaved(false);
    const msg = await saveBridges(bridgesDraft ?? bridgesQuery.data ?? '');
    if (msg?.success) {
      setBridgesSaved(true);
      setBridgesDraft(null);
    } else setBridgesError(msg?.msg || t('pages.tor.failed'));
  }

  const pinned = engine?.pinned ?? '';
  const needsUpdate = !!engine?.installed && !!pinned && engine.version !== pinned;

  const statusText = (state?: string, ms?: number, progress?: number) =>
    state === 'connected'
      ? `${t('pages.tor.status.connected')} · ${ms} ms`
      : state === 'starting'
        ? `${t('pages.tor.status.starting')} ${progress ?? 0}%`
        : t('pages.tor.status.notConnected');

  return (
    <div className="col" style={{ gap: 12 }}>
      <div className="col" style={{ gap: 8 }}>
        <div className="tile row" style={{ padding: '10px 12px', gap: 10, fontSize: 13 }}>
          <Icon name="info" size={16} />
          <span className="f1">{t('pages.tor.notice.tcp')}</span>
        </div>
        <div
          className="tile edge st-warn row"
          style={{ padding: '10px 12px', gap: 10, fontSize: 13 }}
        >
          <Icon name="alert" size={16} />
          <span className="f1">{t('pages.tor.notice.exit')}</span>
        </div>
      </div>

      {engine && (
        <div className="row wrap sp" style={{ gap: 10 }}>
          <div className="row" style={{ gap: 8 }}>
            <Pill status={engine.installed ? 'ok' : 'warn'}>
              {engine.installed
                ? t('pages.tor.engine.installed')
                : t('pages.tor.engine.notInstalled')}
            </Pill>
            <span className="mut" style={{ fontSize: 12.5 }}>
              {engine.installed
                ? engine.version && t('pages.tor.engine.version', { version: engine.version })
                : t('pages.tor.engine.installHint')}
            </span>
          </div>
          {(!engine.installed || needsUpdate) && (
            <div className="row wrap" style={{ gap: 8 }}>
              <EngineRoutePicker id="tor-engine-route" value={route} onChange={setRoute} />
              <Btn
                kind="pri"
                size="sm"
                icon="down"
                aria-busy={installing}
                disabled={installing || !engineRouteReady(route)}
                onClick={() => change(() => installEngine(route))}
              >
                {engine.installed
                  ? t('pages.tor.engine.update', { version: pinned })
                  : t('pages.tor.engine.install', { version: pinned })}
              </Btn>
            </div>
          )}
        </div>
      )}

      {error && (
        <div
          className="tile edge st-bad row"
          role="alert"
          style={{ padding: '10px 12px', gap: 10, fontSize: 13 }}
        >
          <Icon name="alert" size={16} />
          <span className="f1">{error}</span>
          <button
            type="button"
            className="ib sm"
            aria-label={t('close')}
            onClick={() => setError('')}
          >
            <Icon name="x" />
          </button>
        </div>
      )}

      <label className="search" style={{ cursor: 'text' }}>
        <Icon name="search" />
        <input
          aria-label={t('pages.tor.search')}
          placeholder={t('pages.tor.search')}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          style={{
            border: 0,
            background: 'none',
            outline: 'none',
            color: 'var(--tx)',
            font: 'inherit',
            flex: 1,
            minWidth: 0,
          }}
        />
      </label>
      <div className="list tor-country-list" style={{ maxHeight: 420, overflowY: 'auto' }}>
        {countries.length === 0 && <div className="li mut">{t('pages.tor.empty')}</div>}
        {countries.map(({ code, name }) => {
          const exit = added.get(code);
          const result = results[code];
          return (
            <div key={code} className="li" data-country={code}>
              <FlagDot country={code.toUpperCase()} size={28} />
              <div className="f1" style={{ minWidth: 0 }}>
                <b>{name}</b>{' '}
                <span className="mut mono" style={{ fontSize: 11.5 }}>
                  {code}
                </span>
                <div className="mut num" style={{ fontSize: 12.5 }}>
                  {result ? (
                    <span style={{ color: result.ok ? 'var(--ok)' : 'var(--bad)' }}>
                      {result.text}
                    </span>
                  ) : exit ? (
                    <>
                      {statusText(
                        exit.status?.state,
                        exit.status?.latencyMs,
                        exit.status?.progress,
                      )}
                      {exit.status &&
                      !['connected', 'starting'].includes(exit.status.state) &&
                      exit.status.error
                        ? ` · ${exit.status.error}`
                        : ''}
                      {exit.status?.state === 'not_installed'
                        ? ` · ${t('pages.tor.reason.notInstalled')}`
                        : ''}
                    </>
                  ) : (
                    t('torPage.notAdded')
                  )}
                </div>
              </div>
              <Btn
                size="sm"
                icon="refresh"
                aria-busy={testing === code}
                disabled={testing !== null && testing !== code}
                onClick={() => testCountry(code)}
              >
                {t('pages.tor.test')}
              </Btn>
              <Toggle
                on={!!exit}
                label={t('torPage.use', { name })}
                onChange={(on) => change(() => (on ? add(code) : remove(exit!.id)))}
              />
            </div>
          );
        })}
      </div>

      <details className="more-opts">
        <summary>{t('pages.tor.advanced')}</summary>
        <div className="col" style={{ gap: 8, marginTop: 8 }}>
          <div className="mut" style={{ fontSize: 12.5 }}>
            {t('pages.tor.bridges.help')}
          </div>
          <textarea
            className="inp mono"
            rows={4}
            value={bridgesDraft ?? bridgesQuery.data ?? ''}
            onChange={(e) => setBridgesDraft(e.target.value)}
            aria-label={t('pages.tor.bridges.title')}
          />
          {bridgesError && (
            <div className="mut" role="alert" style={{ color: 'var(--bad)' }}>
              {bridgesError}
            </div>
          )}
          {bridgesSaved && <div style={{ color: 'var(--ok)' }}>{t('pages.tor.bridges.saved')}</div>}
          <div>
            <Btn size="sm" icon="check" disabled={savingBridges} onClick={saveBridgeLines}>
              {t('pages.tor.bridges.save')}
            </Btn>
          </div>
        </div>
      </details>
    </div>
  );
}
