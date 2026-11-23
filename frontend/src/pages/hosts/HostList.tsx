import { useMemo, useRef, type PointerEvent } from 'react';
import { useTranslation } from 'react-i18next';

import Icon from '@/components/dl/Icon';
import { MoreMenu } from '@/components/dl/MoreMenu';
import { Badge, Btn, IconBtn, Pill, Toggle } from '@/components/dl/primitives';
import { ClipboardManager } from '@/utils';

import type { HostRecord } from '@/api/queries/useHostsQuery';
import type { InboundOption } from '@/schemas/client';
import { usePointerRowReorder, type PointerRowDrop } from '@/hooks/usePointerRowReorder';

interface HostListProps {
  hosts: HostRecord[];
  inboundOptions: InboundOption[];
  loading?: boolean;
  isMobile?: boolean;
  selectedGroupIds: string[];
  onSelectionChange: (groupIds: string[]) => void;
  onAdd: () => void;
  onEdit: (host: HostRecord) => void;
  onDelete: (host: HostRecord) => void;
  onToggleEnable: (host: HostRecord, next: boolean) => void;
  onMove: (host: HostRecord, dir: 'up' | 'down') => void;
  onReorder: (groupIds: string[]) => void;
  reordering?: boolean;
  onBulkEnable: (enable: boolean) => void;
  onBulkDelete: () => void;
}

export function sortHosts(hosts: HostRecord[]): HostRecord[] {
  return [...hosts].sort((a, b) => {
    const sa = a.sortOrder ?? 0;
    const sb = b.sortOrder ?? 0;
    if (sa !== sb) return sa - sb;
    const ar = a.remark || '';
    const br = b.remark || '';
    if (ar !== br) return ar < br ? -1 : 1;
    return a.groupId < b.groupId ? -1 : a.groupId > b.groupId ? 1 : 0;
  });
}

function hostDropDestination(keys: string[], { from, to, after }: PointerRowDrop<string>) {
  const source = keys.indexOf(from);
  const slot = keys.indexOf(to) + (after ? 1 : 0);
  return slot - (source < slot ? 1 : 0);
}

interface RowProps {
  host: HostRecord;
  index: number;
  count: number;
  inbound: (id: number) => string;
  selected: boolean;
  dragState: '' | 'dragging' | 'before' | 'after';
  dragDisabled: boolean;
  onSelect: (on: boolean) => void;
  onHandle: (ev: PointerEvent) => void;
  props: HostListProps;
}

function useHostCells({ host: h, index, count, inbound, onHandle, dragDisabled, props }: RowProps) {
  const { t } = useTranslation();
  const name = h.remark || h.groupId;
  const addrs = h.hosts?.filter((a) => a.trim() !== '') || [];
  const sni = h.sni || h.snis?.[0] || '';
  return {
    name,
    address: (
      <div>
        <div style={{ fontWeight: 750, overflowWrap: 'anywhere' }}>{name}</div>
        {h.serverDescription && (
          <div className="mut" style={{ fontSize: 12 }}>
            {h.serverDescription}
          </div>
        )}
        {h.tags && h.tags.length > 0 && (
          <div className="row wrap" style={{ gap: 4, marginTop: 4 }}>
            {h.tags.map((tag) => (
              <span key={tag} className="badge mono">
                {tag}
              </span>
            ))}
          </div>
        )}
      </div>
    ),
    usedBy: (
      <div className="row wrap" style={{ gap: 4 }}>
        {(h.inboundIds || []).length === 0 ? (
          <span className="mut">–</span>
        ) : (
          h.inboundIds.map((id) => <Badge key={id}>{inbound(id)}</Badge>)
        )}
      </div>
    ),
    endpoint:
      addrs.length === 0 ? (
        <Badge acc>{t('pages.hosts.fields.inheritAddress')}</Badge>
      ) : (
        <div className="col" style={{ gap: 2 }}>
          {addrs.map((a) => (
            <bdi key={a} className="num mono" style={{ fontSize: 12.5 }}>
              {h.port ? `${a}:${h.port}` : a}
            </bdi>
          ))}
        </div>
      ),
    sni: (
      <div className="row wrap" style={{ gap: 4 }}>
        {sni ? <Badge acc>{`SNI: ${sni}`}</Badge> : <span className="mut">–</span>}
        <span className="badge">{h.security || 'same'}</span>
      </div>
    ),
    state: h.isDisabled ? (
      <Pill status="off">{t('hosts.off')}</Pill>
    ) : (
      <Pill status="ok">{t('hosts.on')}</Pill>
    ),
    actions: (
      <div className="row" style={{ gap: 4, justifyContent: 'flex-end' }}>
        <button
          type="button"
          className="ib sm host-drag-handle"
          aria-label={t('pages.hosts.dragToReorder', { name })}
          disabled={dragDisabled}
          style={{ cursor: 'grab', touchAction: 'none' }}
          onPointerDown={onHandle}
        >
          <Icon name="menu" />
        </button>
        <IconBtn small icon="edit" label={t('edit')} onClick={() => props.onEdit(h)} />
        <IconBtn
          small
          icon="copy"
          label={t('hosts.copyAddress')}
          disabled={addrs.length === 0}
          onClick={() => ClipboardManager.copyText(addrs[0])}
        />
        <MoreMenu
          label={`${t('pages.hosts.fields.actions')} ${name}`}
          actions={[
            {
              key: 'up',
              label: t('pages.hosts.moveUp'),
              disabled: index === 0,
              onSelect: () => props.onMove(h, 'up'),
            },
            {
              key: 'down',
              label: t('pages.hosts.moveDown'),
              disabled: index === count - 1,
              onSelect: () => props.onMove(h, 'down'),
            },
            {
              key: 'del',
              label: t('delete'),
              icon: 'trash',
              danger: true,
              onSelect: () => props.onDelete(h),
            },
          ]}
        />
      </div>
    ),
  };
}

function dragStyle(state: RowProps['dragState']) {
  if (state === 'dragging') return { opacity: 0.55 };
  if (state === 'before') return { boxShadow: 'inset 0 2px 0 var(--acc-line)' };
  if (state === 'after') return { boxShadow: 'inset 0 -2px 0 var(--acc-line)' };
  return undefined;
}

function HostRow(p: RowProps) {
  const { t } = useTranslation();
  const c = useHostCells(p);
  return (
    <tr data-row-key={p.host.groupId} style={dragStyle(p.dragState)}>
      <td>
        <input
          type="checkbox"
          aria-label={t('hosts.select', { name: c.name })}
          checked={p.selected}
          onChange={(e) => p.onSelect(e.target.checked)}
          style={{ width: 18, height: 18, accentColor: 'var(--acc)' }}
        />
      </td>
      <td>
        <div className="row">
          <Toggle
            on={!p.host.isDisabled}
            label={`${t('pages.hosts.fields.enable')} ${c.name}`}
            onChange={(on) => p.props.onToggleEnable(p.host, on)}
          />
          {c.address}
        </div>
      </td>
      <td>{c.usedBy}</td>
      <td>{c.endpoint}</td>
      <td>{c.sni}</td>
      <td>{c.state}</td>
      <td>{c.actions}</td>
    </tr>
  );
}

function HostCard(p: RowProps) {
  const { t } = useTranslation();
  const c = useHostCells(p);
  return (
    <div
      className="li"
      data-row-key={p.host.groupId}
      style={{ flexDirection: 'column', alignItems: 'stretch', gap: 8, ...dragStyle(p.dragState) }}
    >
      <div className="row" style={{ gap: 10, alignItems: 'flex-start' }}>
        <input
          type="checkbox"
          aria-label={t('hosts.select', { name: c.name })}
          checked={p.selected}
          onChange={(e) => p.onSelect(e.target.checked)}
          style={{ width: 18, height: 18, accentColor: 'var(--acc)', marginTop: 4 }}
        />
        <div className="f1" style={{ minWidth: 0 }}>
          {c.address}
        </div>
        <Toggle
          on={!p.host.isDisabled}
          label={`${t('pages.hosts.fields.enable')} ${c.name}`}
          onChange={(on) => p.props.onToggleEnable(p.host, on)}
        />
      </div>
      {c.usedBy}
      {c.endpoint}
      {c.sni}
      <div className="row sp">
        {c.state}
        {c.actions}
      </div>
    </div>
  );
}

export default function HostList(props: HostListProps) {
  const { t } = useTranslation();
  const {
    hosts,
    inboundOptions,
    loading,
    isMobile,
    selectedGroupIds,
    onSelectionChange,
    onReorder,
    reordering,
  } = props;
  const inboundsMap = useMemo(() => {
    const map = new Map<number, InboundOption>();
    for (const ib of inboundOptions) map.set(ib.id, ib);
    return map;
  }, [inboundOptions]);
  const inbound = (id: number) => {
    const ib = inboundsMap.get(id);
    return ib ? ib.remark || ib.tag || `#${id}` : `#${id}`;
  };
  const sorted = useMemo(() => sortHosts(hosts), [hosts]);
  const listRef = useRef<HTMLDivElement>(null);
  const keys = sorted.map((h) => h.groupId);
  const { drag, onPointerDown } = usePointerRowReorder({
    containerRef: listRef,
    getRowKey: (row) => keys.find((key) => key === row.dataset.rowKey) ?? null,
    disabled: reordering || loading,
    isNoop: (drop) => keys.indexOf(drop.from) === hostDropDestination(keys, drop),
    onDrop: (drop) => {
      const ids = [...keys];
      const destination = hostDropDestination(keys, drop);
      ids.splice(ids.indexOf(drop.from), 1);
      ids.splice(destination, 0, drop.from);
      onReorder(ids);
    },
  });
  const selected = new Set(selectedGroupIds);
  const rowProps = (h: HostRecord, index: number): RowProps => ({
    host: h,
    index,
    count: sorted.length,
    inbound,
    selected: selected.has(h.groupId),
    dragState:
      drag?.from === h.groupId
        ? 'dragging'
        : drag?.to === h.groupId
          ? drag.after
            ? 'after'
            : 'before'
          : '',
    dragDisabled: Boolean(reordering || loading),
    onSelect: (on) =>
      onSelectionChange(
        on ? [...selectedGroupIds, h.groupId] : selectedGroupIds.filter((id) => id !== h.groupId),
      ),
    onHandle: (ev) => onPointerDown(h.groupId, ev),
    props,
  });

  return (
    <div className="col" style={{ gap: 12 }}>
      {selectedGroupIds.length > 0 && (
        <div className="tile row wrap" style={{ padding: '10px 14px', gap: 12 }}>
          <span className="badge acc">
            {t('pages.hosts.selectedCount', { count: selectedGroupIds.length })}
          </span>
          <span className="f1" />
          <Btn size="sm" icon="check" onClick={() => props.onBulkEnable(true)}>
            {t('pages.hosts.bulkEnable')}
          </Btn>
          <Btn size="sm" icon="power" onClick={() => props.onBulkEnable(false)}>
            {t('pages.hosts.bulkDisable')}
          </Btn>
          <Btn size="sm" kind="dng" icon="trash" onClick={props.onBulkDelete}>
            {t('pages.hosts.bulkDelete')}
          </Btn>
          <IconBtn small icon="x" label={t('cancel')} onClick={() => onSelectionChange([])} />
        </div>
      )}
      <section className="tile dense" style={{ padding: 14 }} ref={listRef}>
        {isMobile ? (
          <div className="list">
            {sorted.map((h, i) => (
              <HostCard key={h.groupId} {...rowProps(h, i)} />
            ))}
          </div>
        ) : (
          <div className="tblbox">
            <table className="tbl" style={{ minWidth: 900 }}>
              <thead>
                <tr>
                  <th>
                    <input
                      type="checkbox"
                      aria-label={t('hosts.selectAll')}
                      checked={sorted.length > 0 && selectedGroupIds.length === sorted.length}
                      onChange={(e) => onSelectionChange(e.target.checked ? keys : [])}
                      style={{ width: 18, height: 18, accentColor: 'var(--acc)' }}
                    />
                  </th>
                  <th>{t('pages.hosts.fields.address')}</th>
                  <th>{t('hosts.usedBy')}</th>
                  <th>{t('pages.hosts.fields.endpoint')}</th>
                  <th>{t('pages.hosts.fields.sni')}</th>
                  <th>{t('tunnels.colState')}</th>
                  <th>
                    <span style={{ position: 'absolute', width: 1, height: 1, overflow: 'hidden' }}>
                      {t('pages.hosts.fields.actions')}
                    </span>
                  </th>
                </tr>
              </thead>
              <tbody>
                {sorted.map((h, i) => (
                  <HostRow key={h.groupId} {...rowProps(h, i)} />
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  );
}
