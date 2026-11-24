import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useQueryClient } from '@tanstack/react-query';
import { Alert, Radio } from 'antd';

import EngineRoutePicker from '@/components/engine/EngineRoutePicker';
import { engineRouteReady, PANEL_ROUTE, type EngineRoute } from '@/schemas/engineRoute';
import {
  TUNNELS_KEY,
  useEngineReleases,
  useTunnelEngine,
  useTunnelMutations,
  type EngineReleaseLookup,
  type ServerTunnel,
  type TunnelEngine,
  type TunnelStatus,
} from '@/api/queries/useTunnels';
import { tunnelRow, type NodeRef } from '@/pages/index/dashboard/tunnelRow';
import { FORWARDER_LABELS, isForwarder } from '@/pages/tunnels/forwardingEngines';
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
  onSpeedtest: () => void;
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
    row.role === 'portal' &&
    !row.managePort &&
    (row.mode !== 'direct' || row.engine === 'rakhsh' || isForwarder(row.engine))
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
    engine: isForwarder(row.engine)
      ? FORWARDER_LABELS[row.engine]
      : row.engine === 'rakhsh'
        ? t('pages.tunnels.engineRakhsh')
        : t('pages.tunnels.engineBackpack'),
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

// Only an enabled reverse entry tunnel has a loopback relay, which is what a speed test goes through.
function canSpeedtest(row: ServerTunnel): boolean {
  return (
    row.role === 'portal' &&
    row.enable &&
    !row.managePort &&
    row.mode !== 'direct' &&
    row.relayPort > 0
  );
}

function Actions({
  row,
  onEdit,
  onCopy,
  onDelete,
  onSpeedtest,
}: Pick<RowProps, 'row' | 'onEdit' | 'onCopy' | 'onDelete' | 'onSpeedtest'>) {
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
      {canSpeedtest(row) && (
        <IconBtn
          small
          icon="bolt"
          label={`${t('tunnels.speedtest.open')} ${row.name}`}
          onClick={onSpeedtest}
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
        <Actions
          row={p.row}
          onEdit={p.onEdit}
          onCopy={p.onCopy}
          onDelete={p.onDelete}
          onSpeedtest={p.onSpeedtest}
        />
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
        <Actions
          row={p.row}
          onEdit={p.onEdit}
          onCopy={p.onCopy}
          onDelete={p.onDelete}
          onSpeedtest={p.onSpeedtest}
        />
      </div>
    </div>
  );
}

// A node reads the list as its own panel is set, so the route only counts for this server.
function releaseLookup(
  engine: TunnelEngine,
  nodeId: number,
  route: EngineRoute,
): EngineReleaseLookup | null {
  if (!engineRouteReady(route)) return null;
  return { engine, nodeId, route: nodeId ? PANEL_ROUTE : route };
}

// One engine: its release and tunnel count, with install, update and the download route in Options.
export function EngineCard({
  engine,
  label,
  blurb,
  count,
  nodes,
  report,
  builtIn,
}: {
  engine: TunnelEngine;
  // An engine that is part of the panel has nothing to install or update.
  builtIn?: boolean;
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
  const [lookup, setLookup] = useState<EngineReleaseLookup | null>(null);
  const [picked, setPicked] = useState('');
  const releases = useEngineReleases(lookup);
  const nodeId = via ? Number(via) : 0;
  const installed = info?.installed ?? false;
  const list = releases.data;
  const pinned = info?.pinned ?? '';
  const chosen = list
    ? list.includes(picked)
      ? picked
      : list.includes(pinned)
        ? pinned
        : list[0]
    : '';
  // A typed link is looked up on request, so a release list is not asked for at every keystroke.
  const lookUp = (nextVia: string, nextRoute: EngineRoute) => {
    const next = releaseLookup(engine, nextVia ? Number(nextVia) : 0, nextRoute);
    if (next && JSON.stringify(next) === JSON.stringify(lookup)) void releases.refetch();
    else setLookup(next);
  };
  const changeVia = (next: string) => {
    setVia(next);
    setLookup(releaseLookup(engine, next ? Number(next) : 0, route));
  };
  const changeRoute = (next: EngineRoute) => {
    setRoute(next);
    setLookup(next.via === 'link' ? null : releaseLookup(engine, nodeId, next));
  };
  const openOptions = () => {
    setOpen(true);
    setLookup(route.via === 'link' ? null : releaseLookup(engine, nodeId, route));
  };
  const run = async () => {
    const version = chosen || undefined;
    const msg =
      installed || nodeId
        ? await updateEngine({ nodeId, engine, route, version })
        : await installEngine({ engine, route, version });
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
        {!builtIn && (
          <Btn
            size="sm"
            icon="settings"
            aria-label={t('tunnels.engineOptions', { engine: label })}
            onClick={openOptions}
          >
            {t('tunnels.options')}
          </Btn>
        )}
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
            onChange={changeVia}
            options={[
              { value: '', label: t('pages.index.thisServer') },
              ...nodes
                .filter((n) => n.enable !== false && !n.transitive)
                .map((n) => ({ value: String(n.id), label: n.name })),
            ]}
          />
          {/* A node downloads as its own panel is set, so the route is only offered for this server. */}
          {!nodeId && (
            <EngineRoutePicker id={`engine-route-${engine}`} value={route} onChange={changeRoute} />
          )}
          <div className="col" style={{ gap: 8 }}>
            <div className="row sp">
              <span className="mut" style={{ fontSize: 12 }}>
                {t('pages.tunnels.release')}
              </span>
              <Btn
                size="sm"
                icon="refresh"
                disabled={releases.isFetching || (!nodeId && !engineRouteReady(route))}
                onClick={() => lookUp(via, route)}
              >
                {t('pages.tunnels.releasesLookup')}
              </Btn>
            </div>
            {releases.isFetching && (
              <span className="mut" style={{ fontSize: 13 }}>
                {t('pages.tunnels.releasesLoading')}
              </span>
            )}
            {!releases.isFetching && releases.isError && (
              <Alert
                type="warning"
                showIcon
                title={t('pages.tunnels.releasesFailed', { engine: label })}
                description={releases.error.message}
              />
            )}
            {!releases.isFetching && !releases.isError && !list && (
              <span className="mut" style={{ fontSize: 13 }}>
                {t('pages.tunnels.releasesIdle')}
              </span>
            )}
            {!releases.isFetching && list && (
              <Radio.Group
                value={chosen}
                onChange={(e) => setPicked(e.target.value)}
                style={{ display: 'flex', flexDirection: 'column', gap: 6 }}
                options={list.map((version) => ({
                  value: version,
                  label: [
                    version,
                    version === pinned && t('pages.tunnels.releaseRecommended'),
                    version === info?.version && t('pages.tunnels.releaseInstalled'),
                  ]
                    .filter(Boolean)
                    .join(' · '),
                }))}
              />
            )}
          </div>
        </div>
      </FormModal>
    </section>
  );
}
