import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useQueryClient } from '@tanstack/react-query';

import EngineRoutePicker from '@/components/engine/EngineRoutePicker';
import { engineRouteReady, PANEL_ROUTE, type EngineRoute } from '@/schemas/engineRoute';
import {
  TUNNELS_KEY,
  useTunnelEngine,
  useTunnelMutations,
  type ServerTunnel,
  type TunnelEngine,
  type TunnelStatus,
} from '@/api/queries/useTunnels';
import { tunnelRow, type NodeRef } from '@/pages/index/dashboard/tunnelRow';
import Icon from '@/components/dl/Icon';
import { DropChip } from '@/components/dl/DropChip';
import { MoreMenu } from '@/components/dl/MoreMenu';
import { FormModal } from '@/components/dl/overlays';
import { Badge, Btn, FlagDot, IconBtn, Pill, Spark, Toggle } from '@/components/dl/primitives';
import { useNum } from '@/components/dl/numbers';
import { SizeFormatter } from '@/utils';

export type TunnelState = 'ok' | 'bad' | 'off' | 'none';
type Report = (msg: { success?: boolean; msg?: string } | null | undefined) => boolean;

export interface RowProps {
  row: ServerTunnel;
  nodes: readonly (NodeRef & { enable?: boolean; transitive?: boolean })[];
  status?: TunnelStatus;
  state: TunnelState;
  onToggle: (on: boolean) => void;
  onEdit: () => void;
  onCopy: () => void;
  onDelete: () => void;
}

// The arrow between two boxes of the flow tile; reverse points back at the entry, as the exit dials in.
export function FlowArrow({ reverse }: { reverse?: boolean }) {
  return (
    <svg width="70" height="16" className="flipx" aria-hidden="true">
      <path
        d={reverse ? 'M62 8H2' : 'M2 8h60'}
        stroke="var(--acc-line)"
        strokeWidth="2"
        strokeDasharray="4 5"
        strokeLinecap="round"
      >
        <animate
          attributeName="stroke-dashoffset"
          from={reverse ? '0' : '18'}
          to={reverse ? '18' : '0'}
          dur="1.2s"
          repeatCount="indefinite"
        />
      </path>
      <path
        d={reverse ? 'm8 3-6 5 6 5' : 'm58 3 6 5-6 5'}
        fill="none"
        stroke="var(--acc-line)"
        strokeWidth="2"
        strokeLinecap="round"
      />
    </svg>
  );
}

function useRowText(row: ServerTunnel, nodes: RowProps['nodes'], status?: TunnelStatus) {
  const { t } = useTranslation();
  const info = tunnelRow(row, nodes);
  const exit = status ? [status.exitLoc, status.exitIp].filter(Boolean).join(' · ') : '';
  const lanes = status?.targetLanes
    ? t('pages.tunnels.statusLanes', { up: status.lanesUp ?? 0, total: status.targetLanes })
    : '';
  const loss = status?.lossPct ? t('pages.tunnels.statusLoss', { pct: status.lossPct }) : '';
  // Xray counts traffic only where the tunnel is an Xray outbound: a reverse entry or a Rakhsh entry.
  const traffic =
    row.role === 'portal' && !row.managePort && (row.mode !== 'direct' || row.engine === 'rakhsh')
      ? `↑ ${SizeFormatter.sizeFormat(row.up)} · ↓ ${SizeFormatter.sizeFormat(row.down)}`
      : '';
  const routes =
    row.role !== 'portal'
      ? [row.exitOutbound || 'direct']
      : [...row.forwardPorts, ...row.routeInbounds];
  return {
    info,
    exit: exit ? `${t('pages.tunnels.statusExit')} ${exit}` : '',
    detail: [lanes, loss].filter(Boolean).join(' · '),
    traffic,
    routes,
    role: row.role === 'portal' ? t('pages.tunnels.rolePortal') : t('pages.tunnels.roleBridge'),
    engine:
      row.engine === 'rakhsh' ? t('pages.tunnels.engineRakhsh') : t('pages.tunnels.engineBackpack'),
    mode: row.mode === 'direct' ? t('pages.tunnels.modeDirect') : t('pages.tunnels.modeReverse'),
    layer: info.layer3 ? t('tunnels.layer3') : row.transport,
  };
}

function StatePill({ state, status }: { state: TunnelState; status?: TunnelStatus }) {
  const { t } = useTranslation();
  if (state === 'none') return <span className="mut">–</span>;
  const map = {
    ok: 'tunnels.filter.ok',
    bad: 'tunnels.filter.bad',
    off: 'tunnels.filter.off',
  } as const;
  return (
    <span title={state === 'bad' ? status?.error : undefined}>
      <Pill status={state}>
        {state === 'ok'
          ? t('pages.tunnels.statusConnected')
          : state === 'bad'
            ? t('pages.tunnels.statusDown')
            : t(map.off)}
      </Pill>
    </span>
  );
}

function Actions({
  row,
  onEdit,
  onCopy,
  onDelete,
}: Pick<RowProps, 'row' | 'onEdit' | 'onCopy' | 'onDelete'>) {
  const { t } = useTranslation();
  const qc = useQueryClient();
  return (
    <div className="row" style={{ gap: 4, justifyContent: 'flex-end' }}>
      {!row.managePort && (
        <IconBtn small icon="edit" label={`${t('edit')} ${row.name}`} onClick={onEdit} />
      )}
      {!row.managePort && row.enable && (
        <IconBtn
          small
          icon="refresh"
          label={`${t('tunnels.test')} ${row.name}`}
          onClick={() => qc.invalidateQueries({ queryKey: [...TUNNELS_KEY, 'status', row.id] })}
        />
      )}
      <MoreMenu
        label={`${t('pages.nodes.actions')} ${row.name}`}
        actions={[
          ...(row.role === 'portal'
            ? [
                {
                  key: 'copy',
                  label: t('pages.tunnels.copyLink'),
                  icon: 'copy' as const,
                  onSelect: onCopy,
                },
              ]
            : []),
          {
            key: 'del',
            label: `${t('delete')} ${row.name}`,
            icon: 'trash' as const,
            danger: true,
            onSelect: onDelete,
          },
        ]}
      />
    </div>
  );
}

function LatencySpark({ status }: { status?: TunnelStatus }) {
  const h = status?.history ?? [];
  if (h.length < 2) return null;
  return (
    <div style={{ width: 120 }}>
      <Spark values={h.map((v) => v || 0)} w={120} h={26} />
    </div>
  );
}

export function TunnelRow(p: RowProps) {
  const { t } = useTranslation();
  const num = useNum();
  const x = useRowText(p.row, p.nodes, p.status);
  const latency = p.status?.connected && p.status.latencyMs > 0 ? p.status.latencyMs : 0;
  return (
    <tr>
      <td>
        <div className="row">
          <Toggle
            on={p.row.enable}
            label={`${t('tunnels.enable')} ${p.row.name}`}
            onChange={p.onToggle}
          />
          <FlagDot country={x.info.country} size={26} />
          <div>
            <div style={{ fontWeight: 750 }}>{p.row.name}</div>
            <div className="mut" style={{ fontSize: 12 }}>
              {x.role} · <bdi className="mono">{`${p.row.address}:${p.row.port}`}</bdi>
            </div>
          </div>
        </div>
      </td>
      <td>
        <Badge>{x.engine}</Badge>
      </td>
      <td>
        {!p.row.managePort && <Badge acc={p.row.mode !== 'direct'}>{x.mode}</Badge>}{' '}
        {x.layer && <Badge>{x.layer}</Badge>}
      </td>
      <td>
        <span className="num" style={{ fontWeight: 700 }}>
          {latency ? <bdi>{t('tunnels.ms', { n: num.int(latency) })}</bdi> : '–'}
        </span>
        {x.detail && (
          <div className="mut" style={{ fontSize: 12 }}>
            {x.detail}
          </div>
        )}
      </td>
      <td>
        <LatencySpark status={p.status} />
      </td>
      <td>
        <StatePill state={p.state} status={p.status} />
        {x.exit && (
          <div className="mut" style={{ fontSize: 12, marginTop: 4 }}>
            {x.exit}
          </div>
        )}
      </td>
      <td>
        {x.traffic && (
          <bdi className="num" style={{ fontSize: 12.5 }}>
            {x.traffic}
          </bdi>
        )}
      </td>
      <td>
        <div className="row wrap" style={{ gap: 4 }}>
          {x.routes.map((r) => (
            <span key={r} className="badge mono">
              {r}
            </span>
          ))}
        </div>
      </td>
      <td>
        <Actions row={p.row} onEdit={p.onEdit} onCopy={p.onCopy} onDelete={p.onDelete} />
      </td>
    </tr>
  );
}

// The phone layout of one tunnel: the same facts as a table row, stacked.
export function TunnelCard(p: RowProps) {
  const { t } = useTranslation();
  const num = useNum();
  const x = useRowText(p.row, p.nodes, p.status);
  const latency = p.status?.connected && p.status.latencyMs > 0 ? p.status.latencyMs : 0;
  return (
    <div className="li" style={{ flexDirection: 'column', alignItems: 'stretch', gap: 8 }}>
      <div className="row" style={{ gap: 10 }}>
        <Toggle
          on={p.row.enable}
          label={`${t('tunnels.enable')} ${p.row.name}`}
          onChange={p.onToggle}
        />
        <FlagDot country={x.info.country} size={26} />
        <b className="f1" style={{ minWidth: 0, overflowWrap: 'anywhere' }}>
          {p.row.name}
        </b>
        <StatePill state={p.state} status={p.status} />
      </div>
      <div className="row wrap" style={{ gap: 4 }}>
        <Badge>{x.engine}</Badge>
        {!p.row.managePort && <Badge acc={p.row.mode !== 'direct'}>{x.mode}</Badge>}
        {x.layer && <Badge>{x.layer}</Badge>}
        {latency > 0 && (
          <span className="badge num">
            <bdi>{t('tunnels.ms', { n: num.int(latency) })}</bdi>
          </span>
        )}
      </div>
      <div className="mut" style={{ fontSize: 12 }}>
        {[x.role, x.exit, x.detail].filter(Boolean).join(' · ')}
      </div>
      <div className="row sp">
        {x.traffic ? (
          <bdi className="num mut" style={{ fontSize: 12 }}>
            {x.traffic}
          </bdi>
        ) : (
          <span />
        )}
        <Actions row={p.row} onEdit={p.onEdit} onCopy={p.onCopy} onDelete={p.onDelete} />
      </div>
    </div>
  );
}

// One engine: its release and tunnel count, with install, update and the download route in Options.
export function EngineCard({
  engine,
  label,
  blurb,
  count,
  nodes,
  report,
}: {
  engine: TunnelEngine;
  label: string;
  blurb: string;
  count: number;
  nodes: RowProps['nodes'];
  report: Report;
}) {
  const { t } = useTranslation();
  const num = useNum();
  const { data: info } = useTunnelEngine(engine);
  const { installEngine, updateEngine, installingEngine } = useTunnelMutations();
  const [open, setOpen] = useState(false);
  const [route, setRoute] = useState<EngineRoute>(PANEL_ROUTE);
  const [via, setVia] = useState('');
  const nodeId = via ? Number(via) : 0;
  const installed = info?.installed ?? false;
  const run = async () => {
    const msg =
      installed || nodeId
        ? await updateEngine({ nodeId, engine, route })
        : await installEngine({ engine, route });
    if (report(msg)) setOpen(false);
  };
  return (
    <section className={`tile edge s4${info && !installed ? ' st-warn' : ''}`}>
      <div className="row sp" style={{ marginBottom: 10 }}>
        <div className="row">
          <span className="ico acc">
            <Icon name="engine" />
          </span>
          <div>
            <div className="h3">{label}</div>
            <div className="mut num" style={{ fontSize: 12 }}>
              {info
                ? installed
                  ? t('pages.tunnels.engineVersion', {
                      engine: label,
                      version: info.version || t('pages.tunnels.engineVersionUnknown'),
                    })
                  : t('pages.tunnels.engineMissing', { engine: label })
                : '…'}
            </div>
          </div>
        </div>
        {info && (
          <Pill status={installed ? 'ok' : 'warn'}>
            {installed ? t('tunnels.installed') : t('tunnels.notInstalled')}
          </Pill>
        )}
      </div>
      <p className="mut" style={{ margin: '0 0 12px', fontSize: 13 }}>
        {info && !installed && count > 0
          ? t('pages.tunnels.engineMissingHint', { engine: label })
          : blurb}
      </p>
      <div className="row sp">
        <span className="badge">{t('tunnels.count', { n: num.int(count) })}</span>
        <Btn
          size="sm"
          icon="settings"
          aria-label={t('tunnels.engineOptions', { engine: label })}
          onClick={() => setOpen(true)}
        >
          {t('tunnels.options')}
        </Btn>
      </div>
      <FormModal
        open={open}
        onClose={() => setOpen(false)}
        title={t('tunnels.engineOptions', { engine: label })}
        art={
          <span className="ico acc">
            <Icon name="engine" />
          </span>
        }
        width={520}
        footer={
          <>
            <Btn kind="ghost" onClick={() => setOpen(false)}>
              {t('cancel')}
            </Btn>
            <Btn
              kind="pri"
              icon="down"
              disabled={installingEngine || (!nodeId && !engineRouteReady(route))}
              onClick={run}
            >
              {installed
                ? t('pages.tunnels.updateEngine')
                : t('pages.tunnels.installEngine', { engine: label })}
            </Btn>
          </>
        }
      >
        <div className="col" style={{ gap: 14 }}>
          <DropChip
            icon="nodes"
            label={t('tunnels.downloadOn')}
            value={via}
            onChange={setVia}
            options={[
              { value: '', label: t('pages.index.thisServer') },
              ...nodes
                .filter((n) => n.enable !== false && !n.transitive)
                .map((n) => ({ value: String(n.id), label: n.name })),
            ]}
          />
          {/* A node downloads as its own panel is set, so the route is only offered for this server. */}
          {!nodeId && (
            <EngineRoutePicker id={`engine-route-${engine}`} value={route} onChange={setRoute} />
          )}
        </div>
      </FormModal>
    </section>
  );
}
