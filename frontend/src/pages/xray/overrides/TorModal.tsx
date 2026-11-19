import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Alert, Button, Collapse, Input, Modal, Tag } from 'antd';
import { ExperimentOutlined } from '@ant-design/icons';

import { useTorBridges, useTorEngine, useTorExits, useTorMutations } from '@/api/queries/useTor';
import { countryFlag, countryName } from '../outbounds/outbounds-tab-helpers';
import TorStatusChip from '../outbounds/TorStatusChip';
import { TOR_COUNTRIES } from './torCountries';
import './TorModal.css';

interface TorModalProps {
  open: boolean;
  onClose: () => void;
  onChanged?: () => void;
}

interface RowResult {
  ok: boolean;
  text: string;
}

export default function TorModal({ open, onClose, onChanged }: TorModalProps) {
  const { t } = useTranslation();
  const { exits } = useTorExits();
  const engine = useTorEngine(open).data;
  const bridgesQuery = useTorBridges(open);
  const { add, remove, test, installEngine, installing, saveBridges, savingBridges } =
    useTorMutations();
  const [search, setSearch] = useState('');
  const [testing, setTesting] = useState<string | null>(null);
  const [results, setResults] = useState<Record<string, RowResult>>({});
  const [error, setError] = useState('');
  const [bridgesDraft, setBridgesDraft] = useState<string | null>(null);
  const [bridgesError, setBridgesError] = useState('');
  const [bridgesSaved, setBridgesSaved] = useState(false);

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

  return (
    <Modal open={open} title={t('pages.tor.title')} footer={null} onCancel={onClose} width={680}>
      <div className="tor-notices">
        <Alert type="info" showIcon title={t('pages.tor.notice.tcp')} />
        <Alert type="warning" showIcon title={t('pages.tor.notice.exit')} />
      </div>

      {engine && (
        <div className="tor-engine">
          <div>
            <Tag color={engine.installed ? 'success' : 'warning'}>
              {engine.installed
                ? t('pages.tor.engine.installed')
                : t('pages.tor.engine.notInstalled')}
            </Tag>
            {engine.installed ? (
              engine.version && (
                <span className="tor-engine-hint">
                  {t('pages.tor.engine.version', { version: engine.version })}
                </span>
              )
            ) : (
              <div className="tor-engine-hint">{t('pages.tor.engine.installHint')}</div>
            )}
          </div>
          {(!engine.installed || needsUpdate) && (
            <Button
              type="primary"
              loading={installing}
              onClick={() => change(() => installEngine())}
            >
              {engine.installed
                ? t('pages.tor.engine.update', { version: pinned })
                : t('pages.tor.engine.install', { version: pinned })}
            </Button>
          )}
        </div>
      )}

      {error && <Alert type="error" showIcon closable title={error} onClose={() => setError('')} />}

      <Input.Search
        allowClear
        aria-label={t('pages.tor.search')}
        value={search}
        placeholder={t('pages.tor.search')}
        onChange={(e) => setSearch(e.target.value)}
        style={{ marginTop: error ? 12 : 0 }}
      />
      <div className="tor-country-list">
        {countries.length === 0 && <div className="tor-row">{t('pages.tor.empty')}</div>}
        {countries.map(({ code, name }) => {
          const exit = added.get(code);
          const result = results[code];
          return (
            <div key={code} className="tor-row" data-country={code}>
              <div className="tor-row-name">
                <span className="tor-row-flag" aria-hidden>
                  {countryFlag(code)}
                </span>
                <span>{name}</span>
                <span className="tor-row-code">{code}</span>
              </div>
              <div className="tor-row-info">
                {result ? (
                  <span className={result.ok ? 'tor-row-ok' : 'tor-row-bad'}>{result.text}</span>
                ) : (
                  exit && <TorStatusChip status={exit.status} />
                )}
              </div>
              <div className="tor-row-actions">
                <Button
                  size="small"
                  icon={<ExperimentOutlined aria-hidden />}
                  loading={testing === code}
                  disabled={testing !== null && testing !== code}
                  onClick={() => testCountry(code)}
                >
                  {t('pages.tor.test')}
                </Button>
                {exit ? (
                  <Button size="small" danger onClick={() => change(() => remove(exit.id))}>
                    {t('pages.tor.remove')}
                  </Button>
                ) : (
                  <Button size="small" type="primary" onClick={() => change(() => add(code))}>
                    {t('pages.tor.add')}
                  </Button>
                )}
              </div>
            </div>
          );
        })}
      </div>

      <Collapse
        ghost
        style={{ marginTop: 12 }}
        items={[
          {
            key: 'advanced',
            label: t('pages.tor.advanced'),
            children: (
              <>
                <div className="tor-engine-hint">{t('pages.tor.bridges.help')}</div>
                <Input.TextArea
                  rows={4}
                  value={bridgesDraft ?? bridgesQuery.data ?? ''}
                  onChange={(e) => setBridgesDraft(e.target.value)}
                  style={{ marginTop: 8, fontFamily: 'monospace' }}
                  aria-label={t('pages.tor.bridges.title')}
                />
                {bridgesError && (
                  <Alert style={{ marginTop: 8 }} type="error" showIcon title={bridgesError} />
                )}
                {bridgesSaved && <div className="tor-row-ok">{t('pages.tor.bridges.saved')}</div>}
                <Button style={{ marginTop: 8 }} loading={savingBridges} onClick={saveBridgeLines}>
                  {t('pages.tor.bridges.save')}
                </Button>
              </>
            ),
          },
        ]}
      />
    </Modal>
  );
}
