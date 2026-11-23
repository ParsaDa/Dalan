import { useCallback, useMemo, useState, type ReactNode } from 'react';
import { useLocation, useSearchParams } from 'react-router';
import { useTranslation } from 'react-i18next';

import Icon from '@/components/dl/Icon';
import { DropChip } from '@/components/dl/DropChip';
import { IlloEmpty } from '@/components/dl/Illustrations';
import { MoreMenu, type MenuAction } from '@/components/dl/MoreMenu';
import {
  Badge,
  Btn,
  Chip,
  Empty as DlEmpty,
  IconBtn,
  Pill,
  Toggle,
} from '@/components/dl/primitives';
import { fmtInt, useNum } from '@/components/dl/numbers';
import { coerceInboundJsonField } from '@/models/dbinbound';
import { HttpUtil, IntlUtil, SizeFormatter } from '@/utils';
import { buildRowActionsMenu } from './RowActions';
import {
  buildHostRemarksByInboundId,
  formatHostRemarksLabel,
  mixedNetworkLabel,
  networkL4,
  networkLabel,
  readStreamHints,
  shadowsocksNetworkLabel,
  tunnelNetworkLabel,
} from './helpers';
import type {
  ClientCountEntry,
  DBInboundRecord,
  GeneralAction,
  InboundListProps,
  RowAction,
} from './types';

type State = 'ok' | 'warn' | 'bad' | 'off';
type Filter = 'all' | 'on' | 'off' | 'attention';

// The transport badges shown after the protocol, from the inbound's stream settings.
export function protocolBadges(r: DBInboundRecord): string[] {
  const out = [r.protocol];
  if (r.isWireguard || r.isAmneziawg || r.isHysteria || r.isTuic) return [...out, 'UDP'];
  if (r.isSS) {
    const s = readStreamHints(r.streamSettings);
    return [...out, shadowsocksNetworkLabel(r.settings), ...(s.isTls ? ['TLS'] : [])];
  }
  if (r.isTunnel) return [...out, tunnelNetworkLabel(r.settings)];
  if (r.isMixed) return [...out, mixedNetworkLabel(r.settings)];
  if (r.isVMess || r.isVLess || r.isTrojan) {
    const s = readStreamHints(r.streamSettings);
    const l4 = networkL4(s.network);
    return [
      ...out,
      networkLabel(s.network),
      ...(l4 ? [l4] : []),
      ...(s.isTls ? ['TLS'] : []),
      ...(s.isReality ? ['Reality'] : []),
    ];
  }
  return out;
}

function fallbackClientCount(record: DBInboundRecord): ClientCountEntry | null {
  const settings = coerceInboundJsonField(record.settings) as {
    clients?: { email?: string; enable?: boolean }[];
  };
  const clients = Array.isArray(settings.clients) ? settings.clients : [];
  if (clients.length === 0) return null;
  const emails = (on: boolean) =>
    clients.filter((c) => c.email && (c.enable !== false) === on).map((c) => c.email!);
  return {
    clients: clients.length,
    active: emails(true),
    deactive: emails(false),
    depleted: [],
    expiring: [],
    online: [],
  };
}

// An enabled inbound needs attention when it ran out of traffic, expired, or expires soon.
export function inboundState(r: DBInboundRecord, expireDiff: number, now: number): State {
  if (!r.enable) return 'off';
  if (r.total > 0 && r.up + r.down >= r.total) return 'bad';
  if (r.expiryTime > 0 && r.expiryTime <= now) return 'bad';
  if (r.expiryTime > 0 && r.expiryTime - now <= expireDiff) return 'warn';
  return 'ok';
}

interface RowProps {
  record: DBInboundRecord;
  state: State;
  now: number;
  selected: boolean;
  hostRemarks: string[];
  node: string;
  cc: ClientCountEntry | null;
  actions: MenuAction[];
  speed?: { up: number; down: number };
  onSelect: (on: boolean) => void;
  onToggle: (on: boolean) => void;
  onRow: (key: RowAction) => void;
}

function useCells(p: RowProps) {
  const { t } = useTranslation();
  const num = useNum();
  const r = p.record;
  const name = r.remark || `#${r.id}`;
  const hosts = p.hostRemarks.length ? formatHostRemarksLabel(p.hostRemarks) : null;
  const stateText: Record<State, string> = {
    ok: t('inbounds.state.ok'),
    warn: t('inbounds.state.warn'),
    bad: r.total > 0 && r.up + r.down >= r.total ? t('depleted') : t('inbounds.state.expired'),
    off: t('inbounds.state.off'),
  };
  const select = (
    <input
      type="checkbox"
      aria-label={t('inbounds.select', { name })}
      checked={p.selected}
      onChange={(e) => p.onSelect(e.target.checked)}
      style={{ width: 18, height: 18, accentColor: 'var(--acc)' }}
    />
  );
  const inbound = (
    <div className="row" style={{ minWidth: 0 }}>
      <Toggle on={r.enable} label={`${t('pages.inbounds.enable')} ${name}`} onChange={p.onToggle} />
      <div style={{ minWidth: 0 }}>
        <div style={{ fontWeight: 750, overflowWrap: 'anywhere' }}>{name}</div>
        <div className="mut" style={{ fontSize: 12 }} title={hosts?.full}>
          {[p.node, hosts?.display].filter(Boolean).join(' · ')}
        </div>
      </div>
    </div>
  );
  const protocol = (
    <div className="row wrap" style={{ gap: 4 }}>
      {protocolBadges(r).map((b, i) => (
        <Badge key={`${b}-${i}`} acc={b === 'Reality'}>
          {b}
        </Badge>
      ))}
    </div>
  );
  const port = (
    <span className="num" style={{ fontWeight: 700 }}>
      {r.port}
    </span>
  );
  const clients = p.cc ? (
    <div>
      <span className="num" style={{ fontWeight: 700 }}>
        {num.int(p.cc.clients)}
      </span>
      {p.cc.online.length > 0 && (
        <div className="row" style={{ gap: 6, fontSize: 12, fontWeight: 650 }}>
          <span className="dot sd ok" />
          {t('inbounds.online', { n: num.int(p.cc.online.length) })}
        </div>
      )}
      {(p.cc.depleted.length > 0 || p.cc.expiring.length > 0) && (
        <div className="mut" style={{ fontSize: 12 }}>
          {[
            p.cc.depleted.length ? `${t('depleted')} ${num.int(p.cc.depleted.length)}` : '',
            p.cc.expiring.length
              ? `${t('inbounds.expiring')} ${num.int(p.cc.expiring.length)}`
              : '',
          ]
            .filter(Boolean)
            .join(' · ')}
        </div>
      )}
    </div>
  ) : (
    <span className="mut">–</span>
  );
  const traffic = (
    <div>
      <bdi className="num" style={{ fontWeight: 700 }}>
        {`${SizeFormatter.sizeFormat(r.up + r.down)} / ${r.total > 0 ? SizeFormatter.sizeFormat(r.total) : '∞'}`}
      </bdi>
      <div className="mut num" style={{ fontSize: 12 }}>
        <bdi>
          {p.speed && (p.speed.up > 0 || p.speed.down > 0)
            ? `↑ ${SizeFormatter.speedFormat(p.speed.up)} · ↓ ${SizeFormatter.speedFormat(p.speed.down)}`
            : `↑ ${SizeFormatter.sizeFormat(r.up)} · ↓ ${SizeFormatter.sizeFormat(r.down)}`}
        </bdi>
      </div>
    </div>
  );
  const state = (
    <div>
      <Pill status={p.state}>{stateText[p.state]}</Pill>
      {r.expiryTime > 0 && (
        <div
          className="mut"
          style={{ fontSize: 12, marginTop: 4 }}
          title={new Date(r.expiryTime).toLocaleString()}
        >
          {IntlUtil.formatRelativeTime(r.expiryTime)}
        </div>
      )}
    </div>
  );
  const actions = (
    <div className="row" style={{ gap: 4, justifyContent: 'flex-end' }}>
      <IconBtn small icon="edit" label={`${t('edit')} ${name}`} onClick={() => p.onRow('edit')} />
      <IconBtn
        small
        icon="qr"
        label={`${t('pages.inbounds.inboundInfo')} ${name}`}
        onClick={() => p.onRow('showInfo')}
      />
      <MoreMenu label={`${t('pages.nodes.actions')} ${name}`} actions={p.actions} />
    </div>
  );
  return { select, inbound, protocol, port, clients, traffic, state, actions };
}

function Row(p: RowProps) {
  const c = useCells(p);
  return (
    <tr data-row-key={p.record.id}>
      <td>{c.select}</td>
      <td>{c.inbound}</td>
      <td>{c.protocol}</td>
      <td>{c.port}</td>
      <td>{c.clients}</td>
      <td>{c.traffic}</td>
      <td>{c.state}</td>
      <td>{c.actions}</td>
    </tr>
  );
}

function Card(p: RowProps) {
  const c = useCells(p);
  return (
    <div
      className="li inbound-card"
      style={{ flexDirection: 'column', alignItems: 'stretch', gap: 10 }}
    >
      <div className="row" style={{ gap: 10 }}>
        {c.select}
        <div className="f1" style={{ minWidth: 0 }}>
          {c.inbound}
        </div>
        {c.port}
      </div>
      {c.protocol}
      <div className="row sp" style={{ alignItems: 'flex-start' }}>
        {c.clients}
        {c.traffic}
        {c.state}
      </div>
      {c.actions}
    </div>
  );
}

function Pager({
  page,
  pages,
  onPage,
}: {
  page: number;
  pages: number;
  onPage: (p: number) => void;
}): ReactNode {
  const { t } = useTranslation();
  const num = useNum();
  if (pages <= 1) return null;
  return (
    <div className="row" style={{ gap: 6, justifyContent: 'flex-end' }}>
      <IconBtn
        small
        icon="back"
        className="flipx"
        label={t('inbounds.prev')}
        disabled={page === 1}
        onClick={() => onPage(page - 1)}
      />
      <span className="badge acc num">{`${num.int(page)} / ${num.int(pages)}`}</span>
      <IconBtn
        small
        icon="chev"
        className="flipx"
        label={t('inbounds.next')}
        disabled={page === pages}
        onClick={() => onPage(page + 1)}
      />
    </div>
  );
}

export default function InboundList({
  dbInbounds,
  clientCount,
  inboundSpeed,
  expireDiff,
  pageSize,
  isMobile,
  subEnable,
  nodesById,
  hosts,
  onAddInbound,
  onGeneralAction,
  onRowAction,
  onBulkDelete,
}: InboundListProps) {
  const { t } = useTranslation();
  const [selected, setSelected] = useState<number[]>([]);
  const [nodeFilter, setNodeFilter] = useState('all');
  const [filter, setFilter] = useState<Filter>('all');
  const [page, setPage] = useState(1);
  const location = useLocation();
  const [searchParams] = useSearchParams();
  const searchParam = searchParams.get('search');
  const [searchKey, setSearchKey] = useState(() => searchParam || '');
  const [prevLocationKey, setPrevLocationKey] = useState(location.key);
  if (location.key !== prevLocationKey) {
    setPrevLocationKey(location.key);
    if (searchParam !== null) setSearchKey(searchParam);
  }
  const [now] = useState(() => Date.now());

  const showNodeFilter = nodesById.size > 0 || dbInbounds.some((ib) => ib.nodeId != null);
  const hostRemarksByInboundId = useMemo(() => buildHostRemarksByInboundId(hosts), [hosts]);
  const stateOf = useCallback(
    (r: DBInboundRecord) => inboundState(r, expireDiff, now),
    [expireDiff, now],
  );

  const searched = useMemo(() => {
    let list = dbInbounds;
    if (nodeFilter === '0') list = list.filter((ib) => ib.nodeId == null);
    else if (nodeFilter !== 'all') list = list.filter((ib) => ib.nodeId === Number(nodeFilter));
    const q = searchKey.trim().toLowerCase();
    if (!q) return list;
    return list.filter(
      (ib) =>
        (ib.remark || '').toLowerCase().includes(q) ||
        String(ib.port).includes(q) ||
        (ib.protocol || '').toLowerCase().includes(q) ||
        (hostRemarksByInboundId.get(ib.id) ?? []).some((h) => h.toLowerCase().includes(q)),
    );
  }, [dbInbounds, nodeFilter, searchKey, hostRemarksByInboundId]);
  const matches = (r: DBInboundRecord, f: Filter) => {
    const s = stateOf(r);
    return (
      f === 'all' ||
      (f === 'on' ? s !== 'off' : f === 'off' ? s === 'off' : s === 'warn' || s === 'bad')
    );
  };
  const visible = searched.filter((r) => matches(r, filter));
  const size = pageSize > 0 ? pageSize : visible.length || 1;
  const pages = Math.max(1, Math.ceil(visible.length / size));
  const current = Math.min(page, pages);
  const shown = visible.slice((current - 1) * size, current * size);

  const onToggle = useCallback(async (r: DBInboundRecord, next: boolean) => {
    const previous = r.enable;
    r.enable = next;
    try {
      const form = new FormData();
      form.append('enable', String(next));
      const msg = await HttpUtil.post(`/panel/api/inbounds/setEnable/${r.id}`, form);
      if (!msg?.success) r.enable = previous;
    } catch {
      r.enable = previous;
    }
  }, []);

  const rowActions = (r: DBInboundRecord): MenuAction[] =>
    (
      buildRowActionsMenu({
        record: r,
        subEnable,
        t,
        hasClients: ((clientCount[r.id] || fallbackClientCount(r))?.clients || 0) > 0,
      }) ?? []
    )
      .filter(
        (i): i is { key: string; label: string; danger?: boolean } =>
          !!i && 'key' in i && 'label' in i,
      )
      .map((i) => ({
        key: String(i.key),
        label: String(i.label),
        danger: i.danger,
        icon: i.key === 'delete' || i.key === 'delAllClients' ? ('trash' as const) : undefined,
        onSelect: () => onRowAction({ key: i.key as RowAction, dbInbound: r }),
      }));

  const rowProps = (r: DBInboundRecord): RowProps => ({
    record: r,
    state: stateOf(r),
    now,
    selected: selected.includes(r.id),
    hostRemarks: hostRemarksByInboundId.get(r.id) ?? [],
    node: r.nodeId == null ? '' : nodesById.get(r.nodeId)?.name || `node #${r.nodeId}`,
    cc: clientCount[r.id] || fallbackClientCount(r),
    actions: rowActions(r),
    speed: inboundSpeed[r.id],
    onSelect: (on) =>
      setSelected((prev) => (on ? [...prev, r.id] : prev.filter((id) => id !== r.id))),
    onToggle: (on) => void onToggle(r, on),
    onRow: (key) => onRowAction({ key, dbInbound: r }),
  });

  const general: MenuAction[] = [
    {
      key: 'import',
      label: t('pages.inbounds.importInbound'),
      icon: 'down',
      onSelect: () => onGeneralAction('import'),
    },
    { key: 'export', label: t('pages.inbounds.export'), onSelect: () => onGeneralAction('export') },
    ...(subEnable
      ? [
          {
            key: 'subs',
            label: `${t('pages.inbounds.export')} — ${t('pages.settings.subSettings')}`,
            onSelect: () => onGeneralAction('subs' as GeneralAction),
          },
        ]
      : []),
    {
      key: 'resetInbounds',
      label: t('pages.inbounds.resetAllTraffic'),
      icon: 'refresh',
      danger: true,
      onSelect: () => onGeneralAction('resetInbounds'),
    },
  ];

  return (
    <div className="col" style={{ gap: 14 }}>
      <div className="pagehead">
        <div>
          <h2 className="h2" style={{ fontSize: 28 }}>
            {t('menu.inbounds')}
          </h2>
          <p>{t('inbounds.subtitle')}</p>
        </div>
        <div className="row wrap">
          <MoreMenu label={t('pages.inbounds.generalActions')} actions={general} />
          <Btn icon="down" onClick={() => onGeneralAction('import')}>
            {t('inbounds.import')}
          </Btn>
          <Btn kind="pri" icon="plus" onClick={onAddInbound}>
            {t('pages.inbounds.addInbound')}
          </Btn>
        </div>
      </div>
      <div className="row wrap sp">
        <div className="chips" role="group" aria-label={t('inbounds.filters')}>
          {(['all', 'on', 'off', 'attention'] as const).map((f) => (
            <Chip
              key={f}
              on={filter === f}
              count={fmtInt(searched.filter((r) => matches(r, f)).length)}
              onClick={() => {
                setFilter(f);
                setPage(1);
              }}
            >
              {t(`inbounds.filter.${f}`)}
            </Chip>
          ))}
        </div>
        <div className="row wrap">
          {showNodeFilter && (
            <DropChip
              icon="nodes"
              label={t('pages.clients.filters.nodes')}
              value={nodeFilter}
              onChange={(v) => {
                setNodeFilter(v);
                setPage(1);
              }}
              options={[
                { value: 'all', label: t('inbounds.anyServer') },
                { value: '0', label: t('pages.clients.filters.localPanel') },
                ...Array.from(nodesById.values()).map((n) => ({
                  value: String(n.id),
                  label: n.name || `#${n.id}`,
                })),
              ]}
            />
          )}
          <label className="search" style={{ minWidth: 220, cursor: 'text' }}>
            <Icon name="search" />
            <input
              aria-label={t('search')}
              placeholder={t('inbounds.searchHint')}
              value={searchKey}
              onChange={(e) => {
                setSearchKey(e.target.value);
                setPage(1);
              }}
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
        </div>
      </div>
      {selected.length > 0 && (
        <div className="tile row wrap" style={{ padding: '10px 14px', gap: 12 }}>
          <span className="badge acc">
            {t('pages.inbounds.selectedCount', { count: selected.length })}
          </span>
          <span className="f1" />
          <Btn
            size="sm"
            kind="dng"
            icon="trash"
            onClick={async () => {
              if (await onBulkDelete(selected)) setSelected([]);
            }}
          >
            {t('delete')}
          </Btn>
          <IconBtn small icon="x" label={t('cancel')} onClick={() => setSelected([])} />
        </div>
      )}
      {dbInbounds.length === 0 ? (
        <section className="tile">
          <DlEmpty
            art={<IlloEmpty size={130} />}
            title={t('inbounds.empty')}
            actions={
              <Btn kind="pri" icon="plus" onClick={onAddInbound}>
                {t('pages.inbounds.addInbound')}
              </Btn>
            }
          >
            {t('inbounds.emptyHint')}
          </DlEmpty>
        </section>
      ) : (
        <section className="tile dense" style={{ padding: 14 }}>
          {isMobile ? (
            <div className="list">
              {shown.map((r) => (
                <Card key={r.id} {...rowProps(r)} />
              ))}
            </div>
          ) : (
            <div className="tblbox">
              <table className="tbl" style={{ minWidth: 980 }}>
                <thead>
                  <tr>
                    <th>
                      <input
                        type="checkbox"
                        aria-label={t('pages.inbounds.selectAll')}
                        checked={visible.length > 0 && selected.length === visible.length}
                        onChange={(e) =>
                          setSelected(e.target.checked ? visible.map((r) => r.id) : [])
                        }
                        style={{ width: 18, height: 18, accentColor: 'var(--acc)' }}
                      />
                    </th>
                    <th>{t('inbounds.colInbound')}</th>
                    <th>{t('pages.inbounds.protocol')}</th>
                    <th>{t('pages.inbounds.port')}</th>
                    <th>{t('clients')}</th>
                    <th>{t('pages.inbounds.traffic')}</th>
                    <th>{t('tunnels.colState')}</th>
                    <th>
                      <span
                        style={{ position: 'absolute', width: 1, height: 1, overflow: 'hidden' }}
                      >
                        {t('pages.nodes.actions')}
                      </span>
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {shown.map((r) => (
                    <Row key={r.id} {...rowProps(r)} />
                  ))}
                </tbody>
              </table>
            </div>
          )}
          {visible.length === 0 && (
            <div className="mut" style={{ padding: 14, textAlign: 'center' }}>
              {t('noData')}
            </div>
          )}
        </section>
      )}
      <Pager page={current} pages={pages} onPage={setPage} />
    </div>
  );
}
