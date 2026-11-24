import { useMemo, useState } from 'react';
import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router';
import { useQuery, useQueryClient } from '@tanstack/react-query';

import { keys } from '@/api/queryKeys';
import { useStatusQuery } from '@/api/queries/useStatusQuery';
import {
  useActivity,
  useAlerts,
  useMonthlyTraffic,
  useOnlineCount,
} from '@/api/queries/useDashboard';
import { useTunnelMutations, useTunnelsQuery, useTunnelStatus } from '@/api/queries/useTunnels';
import type { NodeRecord } from '@/api/queries/useNodesQuery';
import CountUp from '@/components/ui/CountUp';
import Icon from '@/components/dl/Icon';
import Mark from '@/components/dl/Mark';
import { IlloOffline } from '@/components/dl/Illustrations';
import {
  Badge,
  Bar,
  Btn,
  Empty,
  FlagDot,
  Meter,
  Pill,
  Spark,
  TileHead,
  Toggle,
} from '@/components/dl/primitives';
import type { Status } from '@/components/dl/primitives';
import { Modal, useToast } from '@/components/dl/overlays';
import { useActiveNode } from '@/components/dl/shell/data';
import { useThisServer } from '@/components/dl/shell/useThisServer';
import { useShellPrefs } from '@/components/dl/shell/shellPrefs';
import { nodeStatus } from '@/components/dl/shell/useFrameStatus';
import { fetchSlimInbounds } from '@/pages/inbounds/useInbounds';
import { countryName } from '@/pages/xray/outbounds/outbounds-tab-helpers';
import { HttpUtil, SizeFormatter } from '@/utils';
import { ActivityList } from './ActivityList';
import { AlertsTile } from './AlertsTile';
import { SpeedtestTile } from './SpeedtestTile';
import { tunnelRow } from './tunnelRow';
import type { NodeRef } from './tunnelRow';
import { fmtInt, useNum } from '@/components/dl/numbers';
import { accentScope } from '@/components/dl/accentScope';
import { useTheme } from '@/hooks/useTheme';
import { XRAY_RESTART_STEPS, runXrayRestart } from './xrayRestart';
import { useSparkHistory } from './useSparkHistory';
import { useOverviewHistory } from '../useOverviewHistory';

const days = (secs: number) => Math.max(0, Math.floor(secs / 86400));

function useNodeTraffic(): Map<number, number> {
  const q = useQuery({
    queryKey: keys.inbounds.slim(),
    queryFn: fetchSlimInbounds,
    staleTime: 30000,
  });
  return useMemo(() => {
    const m = new Map<number, number>();
    for (const raw of q.data ?? []) {
      const ib = raw as { nodeId?: number | null; up?: number; down?: number };
      const id = ib.nodeId ?? 0;
      m.set(id, (m.get(id) ?? 0) + (ib.up ?? 0) + (ib.down ?? 0));
    }
    return m;
  }, [q.data]);
}

function useXrayRestart() {
  const { t } = useTranslation();
  const toast = useToast();
  const qc = useQueryClient();
  const [step, setStep] = useState<number | null>(null);
  const start = async () => {
    const result = await runXrayRestart({
      restart: () => HttpUtil.post('/panel/api/server/restartXrayService'),
      readState: async () => {
        const msg = await HttpUtil.get<{ xray?: { state?: string } }>(
          '/panel/api/server/status',
          undefined,
          { silent: true },
        );
        return msg?.obj?.xray?.state ?? '';
      },
      onStep: setStep,
    });
    qc.invalidateQueries({ queryKey: keys.server.status() });
    setStep(null);
    if (result.ok) toast(t('dashboard.xray.restarted'));
    else toast(t('dashboard.xray.restartFailed', { error: result.error }), 'bad');
  };
  return { step, start };
}

function RestartModal({ step, where }: { step: number | null; where: string }) {
  const { t } = useTranslation();
  const labels = [
    t('dashboard.xray.stepSend'),
    t('dashboard.xray.stepWait'),
    t('dashboard.xray.stepDone'),
  ];
  return (
    <Modal
      open={step !== null}
      onClose={() => {}}
      label={t('dashboard.xray.restarting')}
      width={460}
    >
      <div className="row">
        <Mark size={40} motion="loop" />
        <div>
          <h2 className="h2">{t('dashboard.xray.restarting')}</h2>
          <div className="mut">{t('dashboard.xray.restartingSub', { where })}</div>
        </div>
      </div>
      <Bar large value={((step ?? 0) + 1) / XRAY_RESTART_STEPS} />
      <div className="prog" aria-live="polite">
        {labels.map((l, i) => {
          const s = step ?? 0;
          const cls = i < s ? 'done' : i === s ? 'now' : 'todo';
          return (
            <div key={l} className={`pstep ${cls}`}>
              <span className="pi">{i < s && <Icon name="check" />}</span>
              <span className="f1">{l}</span>
            </div>
          );
        })}
      </div>
    </Modal>
  );
}

export function HealthTile({ restart }: { restart: () => void }) {
  const { t } = useTranslation();
  const toast = useToast();
  const qc = useQueryClient();
  const { status } = useStatusQuery();
  const [busy, setBusy] = useState(false);
  const [confirmStop, setConfirmStop] = useState(false);
  const running = status.xray.state === 'running';
  const st: Status = status.xray.state === 'error' ? 'bad' : running ? 'ok' : 'off';
  const highLabel = t('dashboard.high');
  const flip = async (on: boolean) => {
    setBusy(true);
    const msg = await HttpUtil.post(
      on ? '/panel/api/server/restartXrayService' : '/panel/api/server/stopXrayService',
    );
    await qc.invalidateQueries({ queryKey: keys.server.status() });
    setBusy(false);
    if (msg?.success) toast(t(on ? 'dashboard.xray.started' : 'dashboard.xray.stopped'));
  };
  const word = busy
    ? t(running ? 'dashboard.xray.stopping' : 'dashboard.xray.starting')
    : running
      ? t('dashboard.xray.runningFor', { n: fmtInt(days(status.appStats.uptime)) })
      : st === 'bad'
        ? status.xray.errorMsg || t('pages.index.xrayStatusError')
        : t('dashboard.xray.stoppedSub');
  return (
    <section className="tile edge s4" aria-label={t('dashboard.health.title')}>
      <TileHead
        icon="server"
        title={t('dashboard.health.title')}
        right={
          <Pill status={st}>
            {t(
              running
                ? 'shell.status.online'
                : st === 'bad'
                  ? 'pages.index.xrayStatusError'
                  : 'pages.index.xrayStatusStop',
            )}
          </Pill>
        }
      />
      <div className="col" style={{ gap: 16 }}>
        <Meter
          label={t('pages.index.cpu')}
          value={status.cpu.percent / 100}
          highLabel={highLabel}
          shown={status.metricAvailable('cpu') ? undefined : '–'}
        />
        <Meter
          label={t('dashboard.health.memory')}
          value={status.mem.percent / 100}
          highLabel={highLabel}
        />
        <Meter
          label={t('dashboard.health.disk')}
          value={status.disk.percent / 100}
          highLabel={highLabel}
        />
        <div className="sep" />
        <div className="row sp">
          <div className="f1">
            <div className="h3">{t('dashboard.xray.core')}</div>
            <div className="mut" style={{ fontSize: 12.5 }}>
              {word}
            </div>
          </div>
          <div className="row" style={{ gap: 8 }}>
            <Btn size="sm" icon="refresh" onClick={restart} disabled={!running}>
              {t('pages.index.restartXray')}
            </Btn>
            <Toggle
              on={running}
              busy={busy}
              label={t('dashboard.xray.core')}
              onChange={(on) => (on ? flip(true) : setConfirmStop(true))}
            />
          </div>
        </div>
      </div>
      <Modal
        open={confirmStop}
        onClose={() => setConfirmStop(false)}
        label={t('pages.index.stopXrayConfirmTitle')}
        width={440}
      >
        <h2 className="h2">{t('pages.index.stopXrayConfirmTitle')}</h2>
        <p className="mut" style={{ margin: 0 }}>
          {t('pages.index.stopXrayConfirmContent')}
        </p>
        <div className="mfoot">
          <Btn kind="ghost" onClick={() => setConfirmStop(false)}>
            {t('cancel')}
          </Btn>
          <Btn
            kind="dng"
            icon="power"
            onClick={() => {
              setConfirmStop(false);
              flip(false);
            }}
          >
            {t('pages.index.stopXrayConfirmOk')}
          </Btn>
        </div>
      </Modal>
    </section>
  );
}

function TunnelLine({
  tunnel,
  nodes,
}: {
  tunnel: {
    id: number;
    name: string;
    engine: string;
    mode: string;
    enable: boolean;
    nodeId?: number | null;
    options?: Record<string, unknown> | null;
  };
  nodes: NodeRef[];
}) {
  const { t } = useTranslation();
  const toast = useToast();
  const { setEnable } = useTunnelMutations();
  const [busy, setBusy] = useState(false);
  const probe = useTunnelStatus(tunnel.id, tunnel.enable);
  const row = tunnelRow(tunnel, nodes);
  const st: Status = !tunnel.enable
    ? 'off'
    : probe.data
      ? probe.data.connected
        ? 'ok'
        : 'bad'
      : 'off';
  const word = !tunnel.enable
    ? t('shell.status.paused')
    : probe.data
      ? t(probe.data.connected ? 'dashboard.tunnels.up' : 'dashboard.tunnels.down')
      : t('dashboard.tunnels.testing');
  const detail = [
    t(row.mode === 'direct' ? 'dashboard.tunnels.direct' : 'dashboard.tunnels.reverse'),
    row.layer3 ? t('dashboard.tunnels.layer3') : '',
    probe.data?.connected ? t('shell.ms', { n: fmtInt(probe.data.latencyMs) }) : '',
  ]
    .filter(Boolean)
    .join(' · ');
  const route = t('dashboard.tunnels.route', { to: row.to });
  return (
    <div className="li">
      <FlagDot country={row.country} />
      <div className="f1">
        <div style={{ fontWeight: 750 }}>{route}</div>
        <div className="mut" style={{ fontSize: 12 }}>
          <Badge>{tunnel.engine === 'rakhsh' ? 'Rakhsh' : 'BackPack'}</Badge> {detail}
        </div>
      </div>
      <Pill status={st}>{word}</Pill>
      <Toggle
        on={tunnel.enable}
        busy={busy}
        label={t('dashboard.tunnels.toggle', { name: route })}
        onChange={async (enable) => {
          setBusy(true);
          const msg = await setEnable({ id: tunnel.id, enable }).finally(() => setBusy(false));
          if (msg?.success)
            toast(
              t(enable ? 'dashboard.tunnels.started' : 'dashboard.tunnels.paused', { name: route }),
            );
        }}
      />
    </div>
  );
}

function TunnelsTile({ nodes }: { nodes: NodeRef[] }) {
  const { t } = useTranslation();
  const { tunnels, fetched: loaded } = useTunnelsQuery();
  const on = tunnels.filter((x) => x.enable).length;
  return (
    <section className="tile edge s4" aria-label={t('dashboard.tunnels.title')}>
      <TileHead
        icon="tunnel"
        title={t('dashboard.tunnels.title')}
        right={
          <Pill tone="neutral">
            {t('dashboard.tunnels.count', { on: fmtInt(on), n: fmtInt(tunnels.length) })}
          </Pill>
        }
      />
      {loaded && tunnels.length === 0 ? (
        <Empty
          title={t('dashboard.tunnels.none')}
          actions={
            <Link className="btn sm pri" to="/tunnels">
              {t('dashboard.tunnels.add')}
            </Link>
          }
        />
      ) : (
        <div className="list">
          {tunnels.slice(0, 5).map((tu) => (
            <TunnelLine key={tu.id} tunnel={tu} nodes={nodes} />
          ))}
        </div>
      )}
    </section>
  );
}

function UsageTile({
  online,
  rate,
  spark,
  users,
}: {
  online: number | null;
  rate: number | null;
  spark: number[];
  users?: number;
}) {
  const { t } = useTranslation();
  const num = useNum();
  const traffic = useMonthlyTraffic();
  return (
    <section className="tile edge s4" aria-label={t('dashboard.usage.title')}>
      <TileHead
        icon="users"
        title={t('dashboard.usage.title')}
        right={
          <Pill tone="acc" icon="activity">
            {t('dashboard.live')}
          </Pill>
        }
      />
      <div className="row" style={{ alignItems: 'flex-end', gap: 22 }}>
        <div>
          <div className="kpi">
            {online === null ? '–' : <CountUp value={online} format={(v) => num.int(v)} />}
          </div>
          <div className="lbl" style={{ marginTop: 6 }}>
            {t('dashboard.usage.onlineNow')}
          </div>
        </div>
        {rate !== null ? (
          <div>
            <div className="kpi" style={{ fontSize: 28 }}>
              <CountUp value={rate} format={(v) => num.dec(v)} />
              <span className="u">{t('dashboard.usage.mbits')}</span>
            </div>
            <div className="lbl" style={{ marginTop: 6 }}>
              {t('dashboard.usage.throughput')}
            </div>
          </div>
        ) : users !== undefined ? (
          <div>
            <div className="kpi" style={{ fontSize: 28 }}>
              {num.int(users)}
            </div>
            <div className="lbl" style={{ marginTop: 6 }}>
              {t('dashboard.usage.clients')}
            </div>
          </div>
        ) : null}
      </div>
      {spark.length > 1 && (
        <div style={{ margin: '16px -4px 4px' }}>
          <Spark values={spark} w={300} h={64} />
        </div>
      )}
      {rate !== null && traffic && traffic.capGB > 0 && (
        <Meter
          label={t('dashboard.usage.monthly')}
          value={traffic.bytes / (traffic.capGB * 1024 ** 3)}
          shown={SizeFormatter.sizeFormat(traffic.bytes)}
          note={t('dashboard.usage.plan', {
            cap: fmtInt(traffic.capGB),
            n: fmtInt(traffic.daysLeft),
          })}
        />
      )}
    </section>
  );
}

function ActivityTile({ nodes, className = 's7' }: { nodes: NodeRef[]; className?: string }) {
  const { t } = useTranslation();
  const { items, loaded } = useActivity({ page: 1, pageSize: 3 });
  return (
    <section className={`tile ${className}`} aria-label={t('dashboard.activity.title')}>
      <TileHead
        icon="activity"
        title={t('dashboard.activity.title')}
        right={
          <Link className="btn ghost sm" to="/activity">
            {t('dashboard.activity.viewAll')}
            <Icon name="chev" className="flipx" />
          </Link>
        }
      />
      {loaded && items.length === 0 ? (
        <p className="mut" style={{ margin: 0, fontSize: 13 }}>
          {t('dashboard.activity.empty')}
        </p>
      ) : (
        <ActivityList items={items} nodes={nodes} />
      )}
    </section>
  );
}

function NodeCard({
  node,
  traffic,
  behind,
  onOpen,
}: {
  node: NodeRecord;
  traffic: number;
  behind: NodeRecord[];
  onOpen: () => void;
}) {
  const { t } = useTranslation();
  const { isDark } = useTheme();
  const scope = accentScope(node.accent ?? '', node.country ?? '', isDark);
  const st = nodeStatus(node);
  const offline = st === 'bad';
  const stw = t(
    st === 'ok'
      ? 'shell.status.online'
      : st === 'warn'
        ? 'shell.status.highLoadShort'
        : st === 'bad'
          ? 'shell.status.offline'
          : 'shell.status.paused',
  );
  return (
    <button
      type="button"
      className={`tile edge${offline ? ' st-bad' : st === 'warn' ? ' st-warn' : ''} ${scope.className}`}
      onClick={onOpen}
      style={{
        textAlign: 'start',
        font: 'inherit',
        color: 'inherit',
        cursor: 'pointer',
        ...scope.style,
      }}
    >
      <div className="row" style={{ marginBottom: 14 }}>
        <FlagDot country={node.country} size={34} />
        <div className="f1">
          <div className="h3">{node.name}</div>
          <div className="mut num" style={{ fontSize: 12.5 }}>
            {offline
              ? t('dashboard.nodes.lastSeen', {
                  ago: node.lastHeartbeat ? relShort(t, node.lastHeartbeat) : '–',
                })
              : t('dashboard.nodes.latency', { n: fmtInt(node.latencyMs ?? 0) })}
          </div>
        </div>
        <Pill status={st}>{stw}</Pill>
      </div>
      <div className="col" style={{ gap: 10 }}>
        <Meter
          small
          label={t('pages.index.cpu')}
          value={(node.cpuPct ?? 0) / 100}
          shown={offline ? '–' : undefined}
        />
        <Meter
          small
          label={t('dashboard.health.memory')}
          value={(node.memPct ?? 0) / 100}
          shown={offline ? '–' : undefined}
        />
        <div className="row sp" style={{ fontSize: 12.5, fontWeight: 650 }}>
          <span className="mut">
            {t('dashboard.nodes.traffic')}{' '}
            <b className="num" style={{ color: 'var(--tx)' }}>
              <bdi>{SizeFormatter.sizeFormat(traffic)}</bdi>
            </b>
          </span>
          <span className="mut">
            Xray <b style={{ color: 'var(--tx)' }}>{xrayWord(t, node.xrayState, offline)}</b>
          </span>
        </div>
        {behind.length > 0 && (
          <div
            className="mut"
            style={{ fontSize: 12, fontWeight: 650, display: 'flex', gap: 6, alignItems: 'center' }}
          >
            <Icon name="nodes" size={14} />
            {t('dashboard.nodes.behind', {
              n: fmtInt(behind.length),
              names: behind.map((b) => b.name).join(', '),
            })}
          </div>
        )}
      </div>
    </button>
  );
}

function relShort(t: ReturnType<typeof useTranslation>['t'], sec: number) {
  const s = Math.max(0, Date.now() / 1000 - sec);
  if (s < 3600) return t('dashboard.ago.min', { n: fmtInt(Math.max(1, Math.floor(s / 60))) });
  if (s < 86400) return t('dashboard.ago.hour', { n: fmtInt(Math.floor(s / 3600)) });
  return t('dashboard.ago.day', { n: fmtInt(Math.floor(s / 86400)) });
}

function xrayWord(
  t: ReturnType<typeof useTranslation>['t'],
  state: string | undefined,
  offline: boolean,
) {
  if (offline) return t('dashboard.nodes.unreachable');
  if (state === 'running') return t('pages.index.xrayStatusRunning');
  if (state === 'error') return t('pages.index.xrayStatusError');
  return t('pages.index.xrayStatusStop');
}

function NodeView({ node, nodes }: { node: NodeRecord; nodes: NodeRecord[] }) {
  const { t, i18n } = useTranslation();
  const { select } = useActiveNode();
  const traffic = useNodeTraffic();
  const { tunnels } = useTunnelsQuery();
  const here = useThisServer();
  const st = nodeStatus(node);
  const behind = nodes.filter((n) => n.transitive && n.parentGuid && n.parentGuid === node.guid);
  const link = tunnels.find((x) => x.nodeId === node.id);
  const spark = useSparkHistory(`node:${node.id}`, node.onlineCount ?? 0);
  return (
    <div className="col" style={{ gap: 18 }}>
      <div className="crumb">
        <Btn kind="ghost" size="sm" style={{ padding: '0 6px' }} onClick={() => select(null)}>
          {here.name}
        </Btn>
        <Icon name="chev" className="flipx" />
        <span style={{ color: 'var(--tx)' }}>{node.name}</span>
        <Badge acc>{t('dashboard.nodes.managed')}</Badge>
      </div>
      {st === 'bad' ? (
        <section
          className="tile edge st-bad"
          style={{ padding: 40 }}
          aria-label={t('dashboard.nodes.offlineTitle', { name: node.name })}
        >
          <Empty
            art={<IlloOffline />}
            title={t('dashboard.nodes.offlineTitle', { name: node.name })}
            actions={
              <>
                <Link className="btn pri" to="/nodes">
                  <Icon name="refresh" />
                  {t('dashboard.nodes.reconnect')}
                </Link>
                <Link className="btn" to="/logs">
                  <Icon name="logs" />
                  {t('dashboard.nodes.lastLogs')}
                </Link>
              </>
            }
          >
            {t('dashboard.nodes.offlineBody', {
              ago: node.lastHeartbeat ? relShort(t, node.lastHeartbeat) : '–',
            })}
          </Empty>
        </section>
      ) : (
        <div className="grid">
          <section className="tile edge s4" aria-label={t('dashboard.health.title')}>
            <TileHead
              icon="server"
              title={t('dashboard.health.title')}
              right={
                <Pill status={st}>
                  {t(st === 'warn' ? 'shell.status.highLoadShort' : 'shell.status.online')}
                </Pill>
              }
            />
            <div className="col" style={{ gap: 16 }}>
              <Meter
                label={t('pages.index.cpu')}
                value={(node.cpuPct ?? 0) / 100}
                highLabel={t('dashboard.high')}
              />
              <Meter
                label={t('dashboard.health.memory')}
                value={(node.memPct ?? 0) / 100}
                highLabel={t('dashboard.high')}
              />
              <div className="sep" />
              <div className="row sp">
                <div>
                  <div className="h3">{t('dashboard.xray.core')}</div>
                  <div className="mut" style={{ fontSize: 12.5 }}>
                    {xrayWord(t, node.xrayState, false)}
                    {node.uptimeSecs
                      ? ` · ${t('dashboard.xray.uptime', { n: fmtInt(days(node.uptimeSecs)) })}`
                      : ''}
                  </div>
                </div>
                <Link className="btn sm" to="/nodes">
                  <Icon name="refresh" />
                  {t('dashboard.nodes.manage')}
                </Link>
              </div>
            </div>
          </section>
          <UsageTile
            online={node.onlineCount ?? 0}
            rate={null}
            spark={spark}
            users={node.clientCount ?? 0}
          />
          <section className="tile edge s4" aria-label={t('dashboard.nodes.thisNode')}>
            <TileHead
              icon="nodes"
              title={t('dashboard.nodes.thisNode')}
              right={
                <Pill tone="neutral">
                  {t('dashboard.nodes.latency', { n: fmtInt(node.latencyMs ?? 0) })}
                </Pill>
              }
            />
            <dl className="kv">
              <dt>{t('dashboard.nodes.country')}</dt>
              <dd>{node.country ? countryName(node.country, i18n.language) : '–'}</dd>
              <dt>{t('dashboard.nodes.link')}</dt>
              <dd>
                {link
                  ? `${link.engine === 'rakhsh' ? 'Rakhsh' : 'BackPack'} · ${t(link.mode === 'direct' ? 'dashboard.tunnels.direct' : 'dashboard.tunnels.reverse')}`
                  : t('dashboard.nodes.panelLink')}
              </dd>
              <dt>{t('dashboard.nodes.inbounds')}</dt>
              <dd className="num">{node.inboundCount ?? 0}</dd>
              <dt>{t('dashboard.usage.clients')}</dt>
              <dd className="num">{node.clientCount ?? 0}</dd>
              <dt>{t('dashboard.nodes.traffic')}</dt>
              <dd className="num">
                <bdi>{SizeFormatter.sizeFormat(traffic.get(node.id) ?? 0)}</bdi>
              </dd>
              <dt>{t('dashboard.nodes.version')}</dt>
              <dd className="num">{node.panelVersion || '–'}</dd>
            </dl>
          </section>
          {behind.length > 0 && (
            <section
              className="tile s12"
              aria-label={t('dashboard.nodes.behindTitle', { name: node.name })}
            >
              <TileHead
                icon="nodes"
                title={t('dashboard.nodes.behindTitle', { name: node.name })}
                right={
                  <span className="mut" style={{ fontSize: 12.5, fontWeight: 650 }}>
                    {t('dashboard.nodes.behindSub', { name: node.name })}
                  </span>
                }
              />
              <div className="col" style={{ gap: 14 }}>
                {behind.map((b) => {
                  const bs = nodeStatus(b);
                  return (
                    <div key={b.id} className="row wrap" style={{ gap: 16 }}>
                      <div className="row" style={{ gap: 12, minWidth: 260 }}>
                        <FlagDot country={b.country} size={34} />
                        <div>
                          <div className="h3">{b.name}</div>
                          <div className="mut num" style={{ fontSize: 12.5 }}>
                            {t('dashboard.nodes.via', {
                              name: node.name,
                              n: fmtInt(b.latencyMs ?? 0),
                            })}
                          </div>
                        </div>
                        <Pill status={bs}>
                          {t(
                            bs === 'bad'
                              ? 'shell.status.offline'
                              : bs === 'warn'
                                ? 'shell.status.highLoadShort'
                                : 'shell.status.online',
                          )}
                        </Pill>
                      </div>
                      <div className="f1" style={{ minWidth: 220 }}>
                        <Meter small label={t('pages.index.cpu')} value={(b.cpuPct ?? 0) / 100} />
                      </div>
                    </div>
                  );
                })}
              </div>
            </section>
          )}
        </div>
      )}
    </div>
  );
}

export default function DashboardPage({ tools }: { tools?: ReactNode }) {
  const { t } = useTranslation();
  const { status, fetched } = useStatusQuery();
  const { nodes, direct, active, select } = useActiveNode();
  const history = useOverviewHistory(status, fetched);
  const { alerts } = useAlerts();
  const online = useOnlineCount();
  const traffic = useNodeTraffic();
  const xray = useXrayRestart();
  const rate = ((status.netIO.up + status.netIO.down) * 8) / 1e6;
  const spark = history.series.netUp.map(
    (up, i) => ((up + (history.series.netDown[i] ?? 0)) * 8) / 1e6,
  );
  const refs: NodeRef[] = useMemo(
    () => nodes.map((n) => ({ id: n.id, name: n.name ?? `#${n.id}`, country: n.country })),
    [nodes],
  );
  const chained = nodes.filter((n) => n.transitive).length;
  const { simple } = useShellPrefs();
  return (
    <>
      <div
        key={active?.id ?? 'this'}
        className="stag"
        style={{ display: 'flex', flexDirection: 'column', gap: 18 }}
      >
        {active ? (
          <NodeView node={active} nodes={nodes} />
        ) : (
          <div className="grid">
            {!simple && tools}
            <HealthTile restart={xray.start} />
            <TunnelsTile nodes={refs} />
            <UsageTile online={online} rate={rate} spark={spark} />
            <AlertsTile
              alerts={alerts}
              nodes={refs}
              onOpenNode={select}
              onRestartXray={xray.start}
            />
            <ActivityTile nodes={refs} />
            <SpeedtestTile />
            {direct.length > 0 && (
              <section
                className="s12"
                aria-label={t('dashboard.nodes.connected')}
                style={{ display: 'flex', flexDirection: 'column', gap: 14, marginTop: 6 }}
              >
                <div className="row sp">
                  <h2 className="h2">{t('dashboard.nodes.connected')}</h2>
                  <span className="mut" style={{ fontWeight: 650 }}>
                    {t('dashboard.nodes.counts', {
                      direct: fmtInt(direct.length),
                      chained: fmtInt(chained),
                    })}
                  </span>
                </div>
                <div
                  style={{
                    display: 'grid',
                    gridTemplateColumns: 'repeat(auto-fill, minmax(250px, 1fr))',
                    gap: 16,
                  }}
                >
                  {direct.map((n) => (
                    <NodeCard
                      key={n.id}
                      node={n}
                      traffic={traffic.get(n.id) ?? 0}
                      behind={nodes.filter(
                        (b) => b.transitive && b.parentGuid && b.parentGuid === n.guid,
                      )}
                      onOpen={() => select(n.id)}
                    />
                  ))}
                </div>
              </section>
            )}
          </div>
        )}
      </div>
      <RestartModal step={xray.step} where={active?.name ?? t('pages.index.thisServer')} />
    </>
  );
}
