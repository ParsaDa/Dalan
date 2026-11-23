import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router';
import { useQuery } from '@tanstack/react-query';

import { useNodesQuery } from '@/api/queries/useNodesQuery';
import type { NodeRecord } from '@/api/queries/useNodesQuery';
import { useNodeMutations } from '@/api/queries/useNodeMutations';
import { useTunnelsQuery } from '@/api/queries/useTunnels';
import Icon from '@/components/dl/Icon';
import { IlloOffline } from '@/components/dl/Illustrations';
import { MoreMenu } from '@/components/dl/MoreMenu';
import { Modal, useToast } from '@/components/dl/overlays';
import { Badge, Btn, Empty, FlagDot, Meter, Pill, Sk } from '@/components/dl/primitives';
import { fmtInt } from '@/components/dl/numbers';
import { accentScope } from '@/components/dl/accentScope';
import { useTheme } from '@/hooks/useTheme';
import { useActiveNode } from '@/components/dl/shell/data';
import { nodeStatus } from '@/components/dl/shell/useFrameStatus';
import { fetchSlimInbounds } from '@/pages/inbounds/useInbounds';
import { keys } from '@/api/queryKeys';
import { countryName } from '@/pages/xray/outbounds/outbounds-tab-helpers';
import { formatPanelVersion } from '@/lib/panel-version';
import { HttpUtil, SizeFormatter } from '@/utils';
import type { PanelUpdateInfo } from '../index/PanelUpdateModal';
import { EnrolNodeDialog } from './EnrolNode';
import NodeFormModal from './NodeFormModal';
import NodeHistoryPanel from './NodeHistoryPanel';
import ReenrolNotices from './ReenrolNotices';

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
      m.set(ib.nodeId ?? 0, (m.get(ib.nodeId ?? 0) ?? 0) + (ib.up ?? 0) + (ib.down ?? 0));
    }
    return m;
  }, [q.data]);
}

function since(t: ReturnType<typeof useTranslation>['t'], sec: number): string {
  if (!sec) return t('pages.nodes.never');
  const s = Math.max(0, Date.now() / 1000 - sec);
  if (s < 60) return t('pages.nodes.justNow');
  if (s < 3600) return t('dashboard.ago.min', { n: fmtInt(Math.floor(s / 60)) });
  if (s < 86400) return t('dashboard.ago.hour', { n: fmtInt(Math.floor(s / 3600)) });
  return t('dashboard.ago.day', { n: fmtInt(Math.floor(s / 86400)) });
}

interface CardProps {
  node: NodeRecord;
  behind: NodeRecord[];
  traffic: number;
  link?: string;
  latest: string;
  onOpen: () => void;
  onEdit: () => void;
  onProbe: () => void;
  onUpdate: () => void;
  onHistory: () => void;
  onToggle: () => void;
  onDelete: () => void;
}

export function NodeCard({
  node,
  behind,
  traffic,
  link,
  latest,
  onOpen,
  onEdit,
  onProbe,
  onUpdate,
  onHistory,
  onToggle,
  onDelete,
}: CardProps) {
  const { t, i18n } = useTranslation();
  const { isDark } = useTheme();
  const scope = accentScope(node.accent ?? '', node.country ?? '', isDark);
  const st = node.enable === false ? 'off' : nodeStatus(node);
  const offline = st === 'bad';
  const word = t(
    st === 'ok'
      ? 'shell.status.online'
      : st === 'warn'
        ? 'shell.status.highLoadShort'
        : st === 'bad'
          ? 'shell.status.offline'
          : 'shell.status.paused',
  );
  const missing = new Set(node.unavailableMetrics ?? []);
  const updatable =
    !!latest &&
    !!node.panelVersion &&
    formatPanelVersion(node.panelVersion) !== formatPanelVersion(latest);
  const sub = [
    node.country ? countryName(node.country, i18n.language) : '',
    offline
      ? t('dashboard.nodes.lastSeen', { ago: since(t, node.lastHeartbeat ?? 0) })
      : node.latencyMs
        ? t('shell.ms', { n: fmtInt(node.latencyMs) })
        : '',
  ]
    .filter(Boolean)
    .join(' · ');
  return (
    <section
      className={`tile edge s4 ${scope.className}${offline ? ' st-bad' : st === 'warn' ? ' st-warn' : ''}`}
      style={scope.style}
      aria-label={node.name}
    >
      <div className="row" style={{ marginBottom: 14 }}>
        <FlagDot country={node.country} size={36} />
        <div className="f1">
          <div className="h3">{node.name}</div>
          <div className="mut" style={{ fontSize: 12.5 }}>
            {sub}
          </div>
        </div>
        <Pill status={st}>{word}</Pill>
      </div>
      <div className="col" style={{ gap: 10 }}>
        {!offline && st !== 'off' && (
          <>
            <Meter
              label={t('pages.index.cpu')}
              value={(node.cpuPct ?? 0) / 100}
              highLabel={t('dashboard.high')}
              shown={missing.has('cpu') ? '–' : undefined}
            />
            <Meter
              label={t('dashboard.health.memory')}
              value={(node.memPct ?? 0) / 100}
              highLabel={t('dashboard.high')}
            />
          </>
        )}
        {node.xrayError && (
          <div
            className="row"
            style={{ gap: 8, color: 'var(--bad-t)', fontSize: 12.5, fontWeight: 650 }}
          >
            <Icon name="alertc" size={16} />
            <span className="f1">{node.xrayError}</span>
          </div>
        )}
        <dl className="kv">
          <dt>{t('dashboard.nodes.link')}</dt>
          <dd>{link ?? t('dashboard.nodes.panelLink')}</dd>
          <dt>{t('dashboard.usage.clients')}</dt>
          <dd className="num">{fmtInt(node.clientCount ?? 0)}</dd>
          <dt>{t('dashboard.nodes.traffic')}</dt>
          <dd className="num">
            <bdi>{SizeFormatter.sizeFormat(traffic)}</bdi>
          </dd>
          <dt>{t('dashboard.nodes.version')}</dt>
          <dd className="num">
            {node.panelVersion ? formatPanelVersion(node.panelVersion) : '–'}
            {updatable && (
              <>
                {' '}
                <Badge acc>{t('pages.nodes.updateAvailable')}</Badge>
              </>
            )}
          </dd>
          <dt>Xray</dt>
          <dd className="num">{node.xrayVersion || '–'}</dd>
          <dt>{t('pages.nodes.uptime')}</dt>
          <dd className="num">
            {missing.has('uptime') || !node.uptimeSecs
              ? '–'
              : t('dashboard.ago.day', { n: fmtInt(Math.floor(node.uptimeSecs / 86400)) })}
          </dd>
          {behind.length > 0 && (
            <>
              <dt>{t('nodes.behind')}</dt>
              <dd>{behind.map((b) => b.name).join(', ')}</dd>
            </>
          )}
        </dl>
        <div className="row" style={{ gap: 8, marginTop: 4 }}>
          <Btn kind="pri" size="sm" onClick={onOpen} disabled={node.enable === false}>
            {t('nodes.open')}
          </Btn>
          <Btn size="sm" icon="edit" onClick={onEdit}>
            {t('edit')}
          </Btn>
          <span className="f1" />
          <MoreMenu
            label={t('pages.nodes.actions')}
            actions={[
              { key: 'probe', icon: 'refresh', label: t('pages.nodes.probe'), onSelect: onProbe },
              {
                key: 'update',
                icon: 'down',
                label: t('pages.nodes.updatePanel'),
                disabled: !(node.enable && node.status === 'online'),
                onSelect: onUpdate,
              },
              { key: 'history', icon: 'sparkline', label: t('nodes.history'), onSelect: onHistory },
              {
                key: 'toggle',
                icon: 'power',
                label: t(node.enable === false ? 'nodes.turnOn' : 'nodes.turnOff'),
                onSelect: onToggle,
              },
              {
                key: 'delete',
                icon: 'trash',
                label: t('delete'),
                danger: true,
                onSelect: onDelete,
              },
            ]}
          />
        </div>
      </div>
    </section>
  );
}

export default function NodesPage() {
  const { t } = useTranslation();
  const toast = useToast();
  const navigate = useNavigate();
  const { select } = useActiveNode();
  const { nodes, fetched, fetchError, refetch } = useNodesQuery();
  const {
    create,
    update,
    remove,
    setEnable,
    testConnection,
    fetchFingerprint,
    fetchInbounds,
    probe,
    updatePanels,
  } = useNodeMutations();
  const { tunnels } = useTunnelsQuery();
  const traffic = useNodeTraffic();
  const { data: latest = '' } = useQuery({
    queryKey: ['server', 'panelUpdateInfo'],
    queryFn: async () =>
      (await HttpUtil.get<PanelUpdateInfo>('/panel/api/server/getPanelUpdateInfo'))?.obj
        ?.latestVersion || '',
    staleTime: 5 * 60 * 1000,
  });
  const [form, setForm] = useState<{ mode: 'add' | 'edit'; node: NodeRecord | null } | null>(null);
  const [enrol, setEnrol] = useState(false);
  const [mtls, setMtls] = useState(false);
  const [trustCa, setTrustCa] = useState('');
  const [history, setHistory] = useState<NodeRecord | null>(null);
  const [deleting, setDeleting] = useState<NodeRecord | null>(null);
  const [updating, setUpdating] = useState<number[] | null>(null);
  const [dev, setDev] = useState(false);
  const direct = nodes.filter((n) => !n.transitive);
  const eligible = direct.filter((n) => n.enable && n.status === 'online').map((n) => n.id);

  const linkFor = (n: NodeRecord) => {
    const tu = tunnels.find((x) => x.nodeId === n.id);
    return tu
      ? `${tu.engine === 'rakhsh' ? 'Rakhsh' : 'BackPack'} · ${t(tu.mode === 'direct' ? 'dashboard.tunnels.direct' : 'dashboard.tunnels.reverse')}`
      : undefined;
  };
  const runProbe = async (n: NodeRecord) => {
    const msg = await probe(n.id);
    if (msg?.success && msg.obj?.status === 'online')
      toast(t('pages.nodes.connectionOk', { ms: fmtInt(msg.obj.latencyMs ?? 0) }));
    else toast(msg?.obj?.error || t('pages.nodes.toasts.probeFailed'), 'bad');
    refetch();
  };
  const runUpdate = async () => {
    if (!updating) return;
    const ids = updating;
    setUpdating(null);
    const msg = await updatePanels(ids, dev);
    if (!msg?.success) return toast(msg?.msg || t('somethingWentWrong'), 'bad');
    const results = msg.obj ?? [];
    const failed = results.filter((r) => !r.ok);
    if (failed.length === 0) toast(t('pages.nodes.toasts.updateStarted'));
    else
      toast(
        t('pages.nodes.toasts.updateResult', {
          ok: results.length - failed.length,
          failed: failed.length,
        }),
        'warn',
      );
  };
  const copyCa = async () => {
    const msg = await HttpUtil.post<{ caCert: string }>('/panel/api/nodes/mtls/ca');
    if (msg?.success && msg.obj?.caCert) {
      await navigator.clipboard.writeText(msg.obj.caCert);
      toast(t('pages.nodes.mtls.caCopied'));
    } else toast(msg?.msg || t('pages.nodes.mtls.caFailed'), 'bad');
  };
  const saveCa = async () => {
    const msg = await HttpUtil.post('/panel/api/nodes/mtls/trustCA', { caCert: trustCa });
    if (msg?.success) {
      toast(t('pages.nodes.mtls.saved'));
      setMtls(false);
    } else toast(msg?.msg || t('somethingWentWrong'), 'bad');
  };

  return (
    <div className="col stag" style={{ gap: 18 }}>
      <div className="pagehead">
        <div>
          <h2 className="h2" style={{ fontSize: 28 }}>
            {t('menu.nodes')}
          </h2>
          <p>{t('nodes.sub')}</p>
        </div>
        <div className="row wrap">
          <Btn icon="lock" onClick={() => setMtls(true)}>
            {t('pages.nodes.mtls.title')}
          </Btn>
          <Btn
            icon="down"
            disabled={eligible.length === 0}
            onClick={() => {
              setDev(false);
              setUpdating(eligible);
            }}
          >
            {t('nodes.updateAll')}
          </Btn>
          <Btn kind="pri" icon="plus" onClick={() => setForm({ mode: 'add', node: null })}>
            {t('pages.nodes.addNode')}
          </Btn>
        </div>
      </div>
      <ReenrolNotices nodes={nodes} />
      {!fetched ? (
        <div className="grid">
          {[0, 1, 2].map((i) => (
            <section key={i} className="tile s4 col" style={{ gap: 12 }}>
              <div className="row">
                <Sk w={36} h={36} r={18} />
                <Sk w="50%" h={14} />
              </div>
              <Sk h={8} r={99} />
              <Sk h={8} r={99} />
              <Sk h={60} />
            </section>
          ))}
        </div>
      ) : fetchError ? (
        <section className="tile edge st-bad">
          <Empty
            art={<IlloOffline />}
            title={t('somethingWentWrong')}
            actions={
              <Btn kind="pri" icon="refresh" onClick={() => refetch()}>
                {t('refresh')}
              </Btn>
            }
          >
            {fetchError}
          </Empty>
        </section>
      ) : (
        <div className="grid">
          {direct.map((n) => (
            <NodeCard
              key={n.id}
              node={n}
              behind={nodes.filter((b) => b.transitive && b.parentGuid && b.parentGuid === n.guid)}
              traffic={traffic.get(n.id) ?? 0}
              link={linkFor(n)}
              latest={latest}
              onOpen={() => {
                select(n.id);
                navigate(`/?node=${n.id}`);
              }}
              onEdit={() => setForm({ mode: 'edit', node: { ...n } })}
              onProbe={() => runProbe(n)}
              onUpdate={() => {
                setDev(false);
                setUpdating([n.id]);
              }}
              onHistory={() => setHistory(n)}
              onToggle={() => setEnable(n.id, n.enable === false)}
              onDelete={() => setDeleting(n)}
            />
          ))}
          <section
            className="tile s4"
            style={{
              borderStyle: 'dashed',
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
              gap: 10,
              textAlign: 'center',
              minHeight: 240,
            }}
          >
            <span className="ico acc">
              <Icon name="plus" />
            </span>
            <div className="h3">{t('nodes.addTitle')}</div>
            <p className="mut" style={{ margin: 0, maxWidth: 240, fontSize: 13 }}>
              {t('nodes.addBody')}
            </p>
            <Btn size="sm" icon="copy" onClick={() => setEnrol(true)}>
              {t('pages.enrolments.title')}
            </Btn>
          </section>
        </div>
      )}

      <NodeFormModal
        open={form !== null}
        mode={form?.mode ?? 'add'}
        node={form?.node ?? null}
        testConnection={testConnection}
        fetchFingerprint={fetchFingerprint}
        fetchInbounds={fetchInbounds}
        save={(payload) =>
          form?.mode === 'edit' && form.node?.id ? update(form.node.id, payload) : create(payload)
        }
        onOpenChange={(open) => !open && setForm(null)}
        onRemove={form?.mode === 'edit' && form.node ? () => setDeleting(form.node) : undefined}
      />
      <EnrolNodeDialog open={enrol} onClose={() => setEnrol(false)} />

      <Modal
        open={deleting !== null}
        onClose={() => setDeleting(null)}
        label={t('pages.nodes.deleteConfirmTitle', { name: deleting?.name ?? '' })}
        width={460}
      >
        <div className="row" style={{ gap: 12 }}>
          <span className="ico bad">
            <Icon name="trash" />
          </span>
          <div>
            <h2 className="h3">
              {t('pages.nodes.deleteConfirmTitle', { name: deleting?.name ?? '' })}
            </h2>
            <div className="mut" style={{ fontSize: 13 }}>
              {t('pages.nodes.deleteConfirmContent')}
            </div>
          </div>
        </div>
        <div className="mfoot">
          <Btn kind="ghost" onClick={() => setDeleting(null)}>
            {t('cancel')}
          </Btn>
          <Btn
            kind="dng"
            icon="trash"
            onClick={async () => {
              const n = deleting;
              setDeleting(null);
              setForm(null);
              if (n && (await remove(n.id))?.success) toast(t('pages.nodes.toasts.deleted'));
            }}
          >
            {t('delete')}
          </Btn>
        </div>
      </Modal>

      <Modal
        open={updating !== null}
        onClose={() => setUpdating(null)}
        label={t('pages.nodes.updateConfirmTitle', { count: updating?.length ?? 0 })}
        width={480}
      >
        <h2 className="h2">
          {t('pages.nodes.updateConfirmTitle', { count: updating?.length ?? 0 })}
        </h2>
        <p className="mut" style={{ margin: 0 }}>
          {t('pages.nodes.updateConfirmContent')}
        </p>
        <label className="row" style={{ gap: 10, fontWeight: 650 }}>
          <input
            type="checkbox"
            checked={dev}
            onChange={(e) => setDev(e.target.checked)}
            style={{ width: 18, height: 18, accentColor: 'var(--acc)' }}
          />
          {t('pages.nodes.updateDevChannel')}
        </label>
        {dev && (
          <div
            className="row"
            style={{ gap: 8, color: 'var(--warn-t)', fontSize: 13, fontWeight: 650 }}
          >
            <Icon name="alert" size={16} />
            {t('pages.index.devChannelWarning')}
          </div>
        )}
        <div className="mfoot">
          <Btn kind="ghost" onClick={() => setUpdating(null)}>
            {t('cancel')}
          </Btn>
          <Btn kind="pri" icon="down" onClick={runUpdate}>
            {t('update')}
          </Btn>
        </div>
      </Modal>

      <Modal
        open={history !== null}
        onClose={() => setHistory(null)}
        label={t('nodes.history')}
        width={760}
      >
        <div className="row sp">
          <h2 className="h2">
            {t('nodes.history')} · {history?.name}
          </h2>
          <button
            type="button"
            className="ib sm"
            aria-label={t('close')}
            onClick={() => setHistory(null)}
          >
            <Icon name="x" />
          </button>
        </div>
        {history && <NodeHistoryPanel node={{ id: history.id }} />}
      </Modal>

      <Modal
        open={mtls}
        onClose={() => setMtls(false)}
        label={t('pages.nodes.mtls.title')}
        width={560}
      >
        <div className="row sp">
          <h2 className="h2">{t('pages.nodes.mtls.title')}</h2>
          <button
            type="button"
            className="ib sm"
            aria-label={t('close')}
            onClick={() => setMtls(false)}
          >
            <Icon name="x" />
          </button>
        </div>
        <p className="mut" style={{ margin: 0 }}>
          {t('pages.nodes.mtls.intro')}
        </p>
        <div>
          <Btn icon="copy" onClick={copyCa}>
            {t('pages.nodes.mtls.copyCa')}
          </Btn>
          <p className="mut" style={{ margin: '6px 0 0', fontSize: 12.5 }}>
            {t('pages.nodes.mtls.copyCaHint')}
          </p>
        </div>
        <label className="field">
          <span>{t('pages.nodes.mtls.trustLabel')}</span>
          <textarea
            className="inp mono"
            rows={5}
            value={trustCa}
            onChange={(e) => setTrustCa(e.target.value)}
            placeholder={t('pages.nodes.mtls.trustPlaceholder')}
            style={{ height: 'auto', padding: 12, lineHeight: 1.5 }}
          />
          <small>{t('pages.nodes.mtls.trustHint')}</small>
        </label>
        <div className="mfoot">
          <Btn kind="pri" icon="check" onClick={saveCa}>
            {t('pages.nodes.mtls.save')}
          </Btn>
        </div>
      </Modal>
    </div>
  );
}
