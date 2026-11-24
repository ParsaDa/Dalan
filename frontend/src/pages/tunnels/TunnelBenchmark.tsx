import { useMemo, useState } from 'react';
import { Input, Select } from 'antd';
import { useTranslation } from 'react-i18next';

import { useBenchmark } from '@/api/queries/useBenchmark';
import { Btn, Chip, Seg } from '@/components/dl/primitives';
import { Modal } from '@/components/dl/overlays';
import type { Candidate, ForwardingEngine, MethodResult } from '@/generated/zod';
import { benchmarkCandidates, type BenchmarkDirection } from './benchmarkCandidates';
import { FORWARDER_ENGINES, FORWARDER_LABELS, isBuiltIn, isForwarder } from './forwardingEngines';

export interface BenchmarkNode {
  id: number;
  name: string;
  enable: boolean;
}

export interface BenchmarkPick {
  candidate: Candidate;
  nodeId: number;
  address: string;
}

// Kernel carriers need root and the other end's address, so a benchmark does not try them.
const ENGINE_IDS = ['backpack', 'rakhsh', ...FORWARDER_ENGINES.filter((id) => !isBuiltIn(id))];

function Body({
  nodes,
  catalog,
  onUse,
  onClose,
}: {
  nodes: BenchmarkNode[];
  catalog: ForwardingEngine[];
  onUse: (pick: BenchmarkPick) => void;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const { status, start, cancel } = useBenchmark();
  const usable = nodes.filter((n) => n.enable);
  const [nodeId, setNodeId] = useState<number | undefined>(usable[0]?.id);
  const [address, setAddress] = useState('');
  const [objective, setObjective] = useState<'speed' | 'balanced'>('balanced');
  const [direction, setDirection] = useState<BenchmarkDirection>('reverse');
  const [engines, setEngines] = useState<string[]>(['backpack']);

  const engineLabel = (id: string) =>
    id === 'backpack'
      ? t('pages.tunnels.engineBackpack')
      : id === 'rakhsh'
        ? t('pages.tunnels.engineRakhsh')
        : isForwarder(id)
          ? FORWARDER_LABELS[id]
          : id;
  const methodLabel = (c: Candidate) => `${engineLabel(c.engine)} · ${c.transport} · ${c.mode}`;

  const candidates = useMemo(
    () => benchmarkCandidates(engines, direction, catalog),
    [engines, direction, catalog],
  );
  const needsAddress = candidates.some((c) => c.mode === 'reverse');
  const running = status?.running ?? false;
  const ready =
    nodeId !== undefined && candidates.length > 0 && (!needsAddress || address.trim() !== '');

  const toggleEngine = (id: string) =>
    setEngines((now) => (now.includes(id) ? now.filter((e) => e !== id) : [...now, id]));

  const results = status?.results ?? [];
  const best = status?.best;
  const sameAsBest = (r: MethodResult) =>
    best !== undefined &&
    best !== null &&
    r.engine === best.engine &&
    r.transport === best.transport &&
    r.mode === best.mode;

  return (
    <div className="col" style={{ gap: 16, flex: '1 1 auto', minHeight: 0 }}>
      <h2 className="h2">{t('tunnels.benchmark.title')}</h2>
      <p className="mut" style={{ margin: 0, fontSize: 13 }}>
        {t('tunnels.benchmark.hint')}
      </p>

      <div
        data-testid="benchmark-scroll"
        className="col"
        style={{ gap: 16, overflowY: 'auto', flex: '1 1 auto', minHeight: 0, paddingRight: 4 }}
      >
        <div className="col" style={{ gap: 10 }}>
          <label className="col" style={{ gap: 4 }}>
            <span>{t('tunnels.benchmark.node')}</span>
            <Select
              aria-label={t('tunnels.benchmark.node')}
              disabled={running}
              value={nodeId}
              onChange={setNodeId}
              options={usable.map((n) => ({ value: n.id, label: n.name }))}
            />
          </label>
          <label className="col" style={{ gap: 4 }}>
            <span>{t('tunnels.benchmark.address')}</span>
            <Input
              aria-label={t('tunnels.benchmark.address')}
              disabled={running}
              value={address}
              onChange={(e) => setAddress(e.target.value)}
            />
            <span className="mut" style={{ fontSize: 12 }}>
              {t('tunnels.benchmark.addressHint')}
            </span>
          </label>
          <Seg
            label={t('tunnels.benchmark.objective')}
            value={objective}
            onChange={setObjective}
            options={[
              { value: 'balanced', label: t('tunnels.benchmark.balanced') },
              { value: 'speed', label: t('tunnels.benchmark.speed') },
            ]}
          />
          <Seg
            label={t('tunnels.benchmark.direction')}
            value={direction}
            onChange={setDirection}
            options={[
              { value: 'reverse', label: t('pages.tunnels.modeReverse') },
              { value: 'direct', label: t('pages.tunnels.modeDirect') },
              { value: 'both', label: t('tunnels.benchmark.both') },
            ]}
          />
          <div className="row wrap" role="group" aria-label={t('tunnels.benchmark.engines')}>
            {ENGINE_IDS.map((id) => (
              <Chip key={id} on={engines.includes(id)} onClick={() => toggleEngine(id)}>
                {engineLabel(id)}
              </Chip>
            ))}
          </div>
          <span className="mut" style={{ fontSize: 12 }}>
            {t('tunnels.benchmark.methods', { count: candidates.length })}
          </span>
        </div>

        {results.length > 0 && (
          <table style={{ width: '100%', fontSize: 13 }}>
            <thead style={{ position: 'sticky', top: 0, background: 'var(--surf)' }}>
              <tr>
                <th>{t('tunnels.benchmark.colMethod')}</th>
                <th>{t('tunnels.benchmark.colDown')}</th>
                <th>{t('tunnels.benchmark.colUp')}</th>
                <th>{t('tunnels.benchmark.colPing')}</th>
                <th>{t('tunnels.benchmark.colScore')}</th>
                <th aria-label={t('tunnels.benchmark.use')} />
              </tr>
            </thead>
            <tbody>
              {results.map((r) => (
                <tr key={`${r.engine}/${r.transport}/${r.mode}`}>
                  <td>
                    {methodLabel(r)}
                    {!running && sameAsBest(r) && <strong> · {t('tunnels.benchmark.best')}</strong>}
                    {!r.ok && <div className="st-bad">{r.error}</div>}
                    {r.leftover && (
                      <div className="st-warn">
                        {t('tunnels.benchmark.leftover', { what: r.leftover })}
                      </div>
                    )}
                  </td>
                  <td className="num">{r.ok ? r.downMbps.toFixed(1) : '–'}</td>
                  <td className="num">{r.ok ? r.upMbps.toFixed(1) : '–'}</td>
                  <td className="num">{r.ok ? r.pingMs.toFixed(0) : '–'}</td>
                  <td className="num">{r.ok ? r.score.toFixed(0) : '–'}</td>
                  <td>
                    {r.ok && nodeId !== undefined && !running && (
                      <Btn
                        size="sm"
                        onClick={() => onUse({ candidate: r, nodeId, address: address.trim() })}
                      >
                        {t('tunnels.benchmark.use')}
                      </Btn>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <div className="col" style={{ gap: 8 }}>
        {start.error && <span className="st-bad">{(start.error as Error).message}</span>}
        {running && status && (
          <div className="col" style={{ gap: 4 }}>
            <span>
              {t('tunnels.benchmark.progress', { done: status.done, total: status.total })}
            </span>
            {status.current && (
              <span className="mut" style={{ fontSize: 12 }}>
                {methodLabel(status.current)} · {t(`tunnels.benchmark.phase.${status.phase}`)}
              </span>
            )}
          </div>
        )}
        <div className="row" style={{ justifyContent: 'flex-end', gap: 8 }}>
          {running ? (
            <Btn onClick={() => cancel.mutate()}>{t('tunnels.benchmark.stop')}</Btn>
          ) : (
            <Btn
              kind="pri"
              disabled={!ready || start.isPending}
              onClick={() =>
                nodeId !== undefined &&
                start.mutate({ nodeId, address: address.trim(), objective, candidates })
              }
            >
              {t('tunnels.benchmark.start')}
            </Btn>
          )}
          <Btn onClick={onClose}>{t('close')}</Btn>
        </div>
      </div>
    </div>
  );
}

export function TunnelBenchmark({
  open,
  nodes,
  catalog,
  onUse,
  onClose,
}: {
  open: boolean;
  nodes: BenchmarkNode[];
  catalog: ForwardingEngine[];
  onUse: (pick: BenchmarkPick) => void;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  return (
    <Modal
      open={open}
      onClose={onClose}
      label={t('tunnels.benchmark.open')}
      width={760}
      style={{ maxHeight: 'calc(100dvh - 48px)' }}
    >
      {open && <Body nodes={nodes} catalog={catalog} onUse={onUse} onClose={onClose} />}
    </Modal>
  );
}
