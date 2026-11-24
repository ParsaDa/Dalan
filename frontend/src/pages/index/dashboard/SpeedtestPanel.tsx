import { useState } from 'react';
import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

import { useSpeedtest } from '@/api/queries/useSpeedtest';
import Icon from '@/components/dl/Icon';
import { Btn, Spark } from '@/components/dl/primitives';
import { useNum } from '@/components/dl/numbers';

const CEILINGS = [50, 100, 250, 500, 1000, 2500, 5000, 10000];
const ARC = 'M 25 120 A 95 95 0 0 1 215 120';
const PHASES = ['ping', 'download', 'upload'] as const;

// The dial only ever widens during a visit, so the needle does not jump back when a faster phase ends.
function useCeiling(mbps: number): number {
  const [widest, setWidest] = useState(CEILINGS[0]);
  const fit = CEILINGS.find((c) => c >= mbps) ?? CEILINGS[CEILINGS.length - 1];
  if (fit > widest) setWidest(fit);
  return Math.max(widest, fit);
}

function Gauge({ value, ceiling, label }: { value: number; ceiling: number; label: string }) {
  const { dec } = useNum();
  const filled = Math.sqrt(Math.min(value / ceiling, 1)) * 100;
  return (
    <div style={{ position: 'relative', width: '100%', maxWidth: 300, flex: '0 0 auto' }}>
      <svg viewBox="0 0 240 140" width="100%" aria-hidden="true">
        <path
          d={ARC}
          pathLength={100}
          fill="none"
          stroke="var(--glass-b)"
          strokeWidth={14}
          strokeLinecap="round"
        />
        {filled > 0.5 && (
          <path
            d={ARC}
            pathLength={100}
            fill="none"
            stroke="var(--acc)"
            strokeWidth={14}
            strokeLinecap="round"
            strokeDasharray={`${filled} 100`}
            style={{ transition: 'stroke-dasharray var(--d-base) var(--e-out)' }}
          />
        )}
      </svg>
      <div
        style={{
          position: 'absolute',
          left: 0,
          right: 0,
          bottom: 6,
          textAlign: 'center',
          lineHeight: 1.1,
        }}
      >
        <div className="num" style={{ fontSize: 40, fontWeight: 750 }}>
          {dec(value, 1)}
        </div>
        <div className="mut" style={{ fontSize: 12.5 }}>
          Mbps
        </div>
        <div style={{ fontSize: 13, fontWeight: 650 }}>{label}</div>
      </div>
    </div>
  );
}

function Stat({ label, value, unit }: { label: string; value: ReactNode; unit: string }) {
  return (
    <div
      style={{
        flex: '1 1 110px',
        minWidth: 0,
        padding: 14,
        borderRadius: 'var(--r-md)',
        border: '1px solid var(--glass-b)',
      }}
    >
      <div className="mut" style={{ fontSize: 12.5 }}>
        {label}
      </div>
      <div className="num" style={{ fontSize: 24, fontWeight: 700 }}>
        {value}
      </div>
      <div className="mut" style={{ fontSize: 12 }}>
        {unit}
      </div>
    </div>
  );
}

export type SpeedtestRun = ReturnType<typeof useSpeedtest>;

export function SpeedtestRunButton({ run }: { run: SpeedtestRun }) {
  const { t } = useTranslation();
  const busy = (run.status?.running ?? false) || run.start.isPending;
  return (
    <Btn kind="pri" size="sm" icon="bolt" disabled={busy} onClick={() => run.start.mutate()}>
      {busy ? t('dashboard.speedtest.running') : t('dashboard.speedtest.run')}
    </Btn>
  );
}

export function SpeedtestPanel({ run }: { run: SpeedtestRun }) {
  const { t } = useTranslation();
  const { dec } = useNum();
  const { status, history } = run;

  const running = status?.running ?? false;
  const last = status?.result ?? history[0] ?? null;
  const gaugeValue = running ? status!.mbps : (last?.downMbps ?? 0);
  const ceiling = useCeiling(Math.max(gaugeValue, last?.upMbps ?? 0));
  const phaseIndex = PHASES.indexOf((status?.phase ?? '') as (typeof PHASES)[number]);
  const dash = <span aria-hidden="true">–</span>;
  const runs = [...history].reverse();

  return (
    <>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 24, alignItems: 'center' }}>
        <Gauge
          value={gaugeValue}
          ceiling={ceiling}
          label={
            running && status!.phase
              ? t(`dashboard.speedtest.phase.${status!.phase}`)
              : t('dashboard.speedtest.download')
          }
        />
        <div className="col" style={{ flex: '1 1 320px', gap: 14, minWidth: 0 }}>
          {running ? (
            <div className="row" style={{ gap: 16 }}>
              {PHASES.map((p, i) => (
                <span
                  key={p}
                  className={i === phaseIndex ? undefined : 'mut'}
                  style={{ fontWeight: i === phaseIndex ? 700 : 500 }}
                >
                  {i < phaseIndex && <Icon name="check" />} {t(`dashboard.speedtest.${p}`)}
                </span>
              ))}
            </div>
          ) : last ? (
            <div className="mut" style={{ fontSize: 12.5 }}>
              {t('dashboard.speedtest.server', {
                sponsor: last.server.sponsor,
                name: last.server.name,
                country: last.server.country,
              })}
            </div>
          ) : (
            <div className="mut">{t('dashboard.speedtest.empty')}</div>
          )}
          <div className="row" style={{ gap: 12, flexWrap: 'wrap' }}>
            <Stat
              label={t('dashboard.speedtest.ping')}
              value={
                running
                  ? status!.pingMs > 0
                    ? dec(status!.pingMs, 1)
                    : dash
                  : last
                    ? dec(last.pingMs, 1)
                    : dash
              }
              unit="ms"
            />
            <Stat
              label={t('dashboard.speedtest.jitter')}
              value={!running && last ? dec(last.jitterMs, 1) : dash}
              unit="ms"
            />
            <Stat
              label={t('dashboard.speedtest.upload')}
              value={!running && last ? dec(last.upMbps, 1) : dash}
              unit="Mbps"
            />
          </div>
          {!running && status?.error && (
            <div role="alert" style={{ color: 'var(--bad, #d9534f)', fontSize: 13 }}>
              {status.error}
            </div>
          )}
        </div>
      </div>
      {runs.length >= 2 && (
        <div style={{ marginTop: 18 }}>
          <div className="mut" style={{ fontSize: 12.5, marginBottom: 6 }}>
            {t('dashboard.speedtest.history', { n: runs.length })}
          </div>
          <Spark values={runs.map((r) => r.downMbps)} compare={runs.map((r) => r.upMbps)} h={56} />
        </div>
      )}
    </>
  );
}
