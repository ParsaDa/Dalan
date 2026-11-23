import { memo, type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

import { MoreMenu } from '@/components/dl/MoreMenu';
import { Badge, Bar, HIGH_USAGE, IconBtn, Pill } from '@/components/dl/primitives';
import type { InboundOption } from '@/hooks/useClients';
import type { ClientRecord } from '@/schemas/client';
import { IntlUtil, SizeFormatter } from '@/utils';

export type Bucket = 'active' | 'deactive' | 'depleted' | 'expiring';

export interface ClientTableProps {
  rows: ClientRecord[];
  narrow: boolean;
  selected: string[];
  inboundsById: Record<number, InboundOption>;
  clientSpeed: Record<string, { up: number; down: number }>;
  togglingEmail: string | null;
  bucketOf: (row: ClientRecord) => Bucket | null;
  isOnline: (email: string) => boolean;
  onSelect: (email: string, on: boolean) => void;
  onSelectAll: (on: boolean) => void;
  onEdit: (email: string) => void;
  onShowQr: (email: string) => void;
  onShowInfo: (email: string) => void;
  onResetTraffic: (email: string) => void;
  onDelete: (email: string) => void;
  onToggle: (row: ClientRecord, on: boolean) => void;
  onGroup: (group: string) => void;
}

const STATUS: Record<Bucket, 'ok' | 'warn' | 'bad' | 'off'> = {
  active: 'ok',
  expiring: 'warn',
  depleted: 'bad',
  deactive: 'off',
};

function initials(email: string) {
  const letters = email.replace(/[^\p{L}\p{N}]/gu, '');
  return (letters.slice(0, 2) || '?').toUpperCase();
}

// The protocols of the inbounds a client is attached to, each once.
export function clientProtocols(
  ids: number[],
  inboundsById: Record<number, InboundOption>,
): string[] {
  const out: string[] = [];
  for (const id of ids) {
    const p = inboundsById[id]?.protocol;
    if (p && !out.includes(p)) out.push(p);
  }
  return out;
}

function useCells(p: ClientTableProps, row: ClientRecord) {
  const { t } = useTranslation();
  const bucket = p.bucketOf(row);
  const used = (row.traffic?.up || 0) + (row.traffic?.down || 0);
  const total = row.totalGB || 0;
  const ratio = total > 0 ? used / total : 0;
  const speed = p.clientSpeed[row.email];
  const online = row.enable && bucket !== 'depleted' && p.isOnline(row.email);
  const label: Record<Bucket, string> = {
    active: t('clientsPage.status.active'),
    expiring: t('clientsPage.status.expiring'),
    depleted: t('clientsPage.status.depleted'),
    deactive: t('clientsPage.status.paused'),
  };
  const select = (
    <input
      type="checkbox"
      aria-label={t('clientsPage.select', { name: row.email })}
      checked={p.selected.includes(row.email)}
      onChange={(e) => p.onSelect(row.email, e.target.checked)}
      style={{ width: 18, height: 18, accentColor: 'var(--acc)' }}
    />
  );
  const name = (
    <div className="who">
      <i>{initials(row.email)}</i>
      <div style={{ minWidth: 0 }}>
        <div style={{ fontWeight: 750, overflowWrap: 'anywhere' }}>{row.email}</div>
        <div className="mut" style={{ fontSize: 12 }}>
          {row.group ? (
            <button
              type="button"
              className="mut"
              style={{ all: 'unset', cursor: 'pointer', textDecoration: 'underline dotted' }}
              onClick={() => p.onGroup(row.group!)}
            >
              {row.group}
            </button>
          ) : (
            t('clientsPage.noGroup')
          )}
          {row.comment ? ` · ${row.comment}` : ''}
        </div>
      </div>
    </div>
  );
  const status = bucket ? <Pill status={STATUS[bucket]}>{label[bucket]}</Pill> : null;
  const protocols = clientProtocols(row.inboundIds || [], p.inboundsById);
  const protocol = (
    <div
      className="row wrap"
      style={{ gap: 4 }}
      title={(row.inboundIds || []).map((id) => p.inboundsById[id]?.remark || `#${id}`).join(', ')}
    >
      {protocols.length ? (
        protocols.map((x) => <Badge key={x}>{x}</Badge>)
      ) : (
        <span className="mut">–</span>
      )}
    </div>
  );
  const usage = (
    <div className="quota">
      <div className="row sp" style={{ fontSize: 12, fontWeight: 700, marginBottom: 5 }}>
        <bdi className="num">{SizeFormatter.sizeFormat(used)}</bdi>
        <bdi className="mut num">
          {total > 0 ? SizeFormatter.sizeFormat(total) : t('clientsPage.unlimited')}
        </bdi>
      </div>
      <Bar
        value={total > 0 ? ratio : 0.06}
        tone={ratio >= 1 ? 'bad' : ratio >= HIGH_USAGE ? 'warn' : undefined}
      />
      {speed && (speed.up > 0 || speed.down > 0) && (
        <div className="mut num" style={{ fontSize: 11.5, marginTop: 4 }}>
          <bdi>{`↑ ${SizeFormatter.speedFormat(speed.up)} · ↓ ${SizeFormatter.speedFormat(speed.down)}`}</bdi>
        </div>
      )}
    </div>
  );
  const exp = row.expiryTime ?? 0;
  const expires = (
    <span
      className="num"
      style={{ fontWeight: 650 }}
      title={exp > 0 ? new Date(exp).toLocaleString() : undefined}
    >
      {exp === 0
        ? t('clientsPage.never')
        : exp < 0
          ? t('clientsPage.afterFirstUse', { n: Math.round(exp / -86_400_000) })
          : IntlUtil.formatRelativeTime(exp)}
    </span>
  );
  const onlineCell = (
    <span className="row" style={{ gap: 6, fontWeight: 650 }}>
      <span className={`dot sd ${online ? 'ok' : 'off'}`} />
      {online ? t('pages.clients.online') : t('pages.clients.offline')}
    </span>
  );
  const actions = (
    <div className="row" style={{ gap: 4, justifyContent: 'flex-end' }}>
      <IconBtn
        small
        icon="link"
        label={`${t('pages.clients.clientInfo')} ${row.email}`}
        onClick={() => p.onShowInfo(row.email)}
      />
      <IconBtn
        small
        icon="qr"
        label={`${t('pages.clients.qrCode')} ${row.email}`}
        onClick={() => p.onShowQr(row.email)}
      />
      <MoreMenu
        label={`${t('pages.clients.actions')} ${row.email}`}
        actions={[
          { key: 'edit', label: t('edit'), icon: 'edit', onSelect: () => p.onEdit(row.email) },
          {
            key: 'toggle',
            label: row.enable ? t('clientsPage.pause') : t('clientsPage.resume'),
            icon: 'power',
            disabled: p.togglingEmail === row.email,
            onSelect: () => p.onToggle(row, !row.enable),
          },
          {
            key: 'reset',
            label: t('pages.inbounds.resetTraffic'),
            icon: 'refresh',
            onSelect: () => p.onResetTraffic(row.email),
          },
          {
            key: 'del',
            label: t('delete'),
            icon: 'trash',
            danger: true,
            onSelect: () => p.onDelete(row.email),
          },
        ]}
      />
    </div>
  );
  return { select, name, status, protocol, usage, expires, onlineCell, actions };
}

function Row(p: ClientTableProps & { row: ClientRecord }) {
  const c = useCells(p, p.row);
  return (
    <tr data-row-key={p.row.email}>
      <td>{c.select}</td>
      <td>{c.name}</td>
      <td>{c.status}</td>
      <td>{c.protocol}</td>
      <td style={{ minWidth: 170 }}>{c.usage}</td>
      <td>{c.expires}</td>
      <td>{c.onlineCell}</td>
      <td>{c.actions}</td>
    </tr>
  );
}

function Card(p: ClientTableProps & { row: ClientRecord }) {
  const c = useCells(p, p.row);
  return (
    <div
      className="li client-card"
      style={{ flexDirection: 'column', alignItems: 'stretch', gap: 10 }}
    >
      <div className="row" style={{ gap: 10 }}>
        {c.select}
        <div className="f1" style={{ minWidth: 0 }}>
          {c.name}
        </div>
        {c.status}
      </div>
      {c.usage}
      <div className="row sp wrap" style={{ gap: 8 }}>
        {c.protocol}
        {c.expires}
        {c.onlineCell}
      </div>
      {c.actions}
    </div>
  );
}

// The design's client table; a stacked card list on phones.
function ClientTable(p: ClientTableProps): ReactNode {
  const { t } = useTranslation();
  if (p.narrow) {
    return (
      <div className="list">
        {p.rows.map((row) => (
          <Card key={row.email} {...p} row={row} />
        ))}
      </div>
    );
  }
  return (
    <div className="tblbox">
      <table className="tbl" style={{ minWidth: 920 }}>
        <thead>
          <tr>
            <th>
              <input
                type="checkbox"
                aria-label={t('pages.clients.selectAll')}
                checked={p.rows.length > 0 && p.selected.length === p.rows.length}
                onChange={(e) => p.onSelectAll(e.target.checked)}
                style={{ width: 18, height: 18, accentColor: 'var(--acc)' }}
              />
            </th>
            <th>{t('outbounds.colName')}</th>
            <th>{t('clientsPage.colStatus')}</th>
            <th>{t('pages.inbounds.protocol')}</th>
            <th>{t('clientsPage.colUsage')}</th>
            <th>{t('clientsPage.colExpires')}</th>
            <th>{t('pages.clients.online')}</th>
            <th>
              <span style={{ position: 'absolute', width: 1, height: 1, overflow: 'hidden' }}>
                {t('pages.clients.actions')}
              </span>
            </th>
          </tr>
        </thead>
        <tbody>
          {p.rows.map((row) => (
            <Row key={row.email} {...p} row={row} />
          ))}
        </tbody>
      </table>
    </div>
  );
}

export default memo(ClientTable);
