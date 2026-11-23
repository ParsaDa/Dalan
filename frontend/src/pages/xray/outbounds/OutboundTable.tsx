import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router';

import Icon from '@/components/dl/Icon';
import { MoreMenu } from '@/components/dl/MoreMenu';
import { Badge, FlagDot, IconBtn, Pill } from '@/components/dl/primitives';
import type { OutboundTestState, OutboundTrafficRow } from '@/hooks/useXraySetting';
import { SizeFormatter } from '@/utils';
import type { OutboundRow } from './outbounds-tab-types';
import TestResultPopover from './TestResultPopover';
import type { TorStatusInfo } from './TorStatusChip';
import {
  countryName,
  isTesting,
  isUntestable,
  outboundAddresses,
  testResult,
  trafficFor,
} from './outbounds-tab-helpers';

export interface OutboundTableProps {
  rows: OutboundRow[];
  storedCount: number;
  narrow: boolean;
  outboundsTraffic: OutboundTrafficRow[];
  outboundTestStates: Record<number, OutboundTestState>;
  torStatus?: Record<string, TorStatusInfo>;
  openEdit: (idx: number) => void;
  setFirst: (idx: number) => void;
  moveUp: (idx: number) => void;
  moveDown: (idx: number) => void;
  confirmDelete: (idx: number) => void;
  onResetTraffic: (tag: string) => void;
  onTest: (index: number) => void;
  onShowTor: () => void;
}

interface Cells {
  name: ReactNode;
  type: ReactNode;
  state: ReactNode;
  latency: ReactNode;
  actions: ReactNode;
}

// One outbound's cells, shared by the table row and the phone card.
function useCells(p: OutboundTableProps, row: OutboundRow, index: number): Cells {
  const { t, i18n } = useTranslation();
  const tr = trafficFor(p.outboundsTraffic, row);
  const result =
    row.torCountry || row.tunnelEngine ? null : testResult(p.outboundTestStates, row.key);
  const testing = isTesting(p.outboundTestStates, row.key);
  const tor = row.torCountry ? p.torStatus?.[row.tag || ''] : undefined;
  const egress = result?.egress;
  const country = row.torCountry || egress?.country;
  const sub = row.tunnelEngine
    ? `${row.tunnelEngine === 'rakhsh' ? t('pages.tunnels.engineRakhsh') : row.tunnelEngine === 'backpack' ? t('pages.tunnels.engineBackpack') : row.tunnelEngine} · ${t('pages.tunnels.modeReverse')}`
    : row.torCountry
      ? `${t('outbounds.torExit')} · ${countryName(row.torCountry, i18n.language)}`
      : [row.protocol, outboundAddresses(row)[0]].filter(Boolean).join(' · ');
  const security = row.streamSettings?.security;
  const network = row.streamSettings?.network;

  const name = (
    <div className="row" style={{ gap: 8 }}>
      {row.tunnelEngine ? (
        <span className="ico acc" style={{ width: 22, height: 22, borderRadius: '50%' }}>
          <Icon name="tunnel" size={14} />
        </span>
      ) : row.torCountry ? (
        <FlagDot country={row.torCountry} size={22} />
      ) : null}
      <div style={{ minWidth: 0 }}>
        <div className="tag-name" style={{ fontWeight: 750, overflowWrap: 'anywhere' }}>
          {row.tag}
        </div>
        <div className="mut" style={{ fontSize: 12 }}>
          <bdi>{sub}</bdi>
        </div>
        {(network || (security && security !== 'none')) && (
          <div className="row wrap" style={{ gap: 4, marginTop: 4 }}>
            {network && <span className="badge">{network}</span>}
            {security && security !== 'none' && <span className="badge">{security}</span>}
          </div>
        )}
      </div>
    </div>
  );

  const type = (
    <Badge acc={Boolean(row.tunnelEngine)}>{row.tunnelEngine ? 'tunnel' : row.protocol}</Badge>
  );

  let state: ReactNode;
  if (tor) {
    state =
      tor.state === 'connected' ? (
        <Pill status="ok">{t('pages.tor.status.connected')}</Pill>
      ) : tor.state === 'starting' ? (
        <Pill status="warn">{`${t('pages.tor.status.starting')} ${tor.progress ?? 0}%`}</Pill>
      ) : (
        <>
          <Pill status="bad">{t('pages.tor.status.notConnected')}</Pill>
          <div className="mut" style={{ fontSize: 12, marginTop: 4 }}>
            {tor.state === 'not_installed'
              ? t('pages.tor.reason.notInstalled')
              : tor.state === 'stopped'
                ? t('pages.tor.reason.stopped')
                : tor.error || t('pages.tor.reason.failed')}
          </div>
        </>
      );
  } else if (testing) {
    state = <Pill tone="neutral">{t('outbounds.testing')}</Pill>;
  } else if (result) {
    state = result.success ? (
      <Pill status="ok">{t('outbounds.ready')}</Pill>
    ) : (
      <span title={result.error}>
        <Pill status="bad">{t('outbounds.failed')}</Pill>
      </span>
    );
  } else if (row.tunnelEngine || row.torCountry) {
    state = <span className="mut">–</span>;
  } else if (isUntestable(row)) {
    state = <Pill status="ok">{t('outbounds.ready')}</Pill>;
  } else {
    state = <Pill tone="neutral">{t('outbounds.notTested')}</Pill>;
  }

  const latencyValue =
    tor?.state === 'connected' ? tor.latencyMs : result?.success ? result.delay : 0;
  const latency = (
    <>
      {latencyValue ? (
        result ? (
          <TestResultPopover result={result}>
            <button
              type="button"
              className="num"
              style={{ all: 'unset', cursor: 'help', fontWeight: 700 }}
            >
              {`${latencyValue} ms`}
            </button>
          </TestResultPopover>
        ) : (
          <span className="num" style={{ fontWeight: 700 }}>{`${latencyValue} ms`}</span>
        )
      ) : (
        <span className="num">–</span>
      )}
      {country && !row.torCountry && (
        <div
          className="mut"
          style={{ fontSize: 12 }}
          title={[egress?.ipv4, egress?.ipv6].filter(Boolean).join(' · ')}
        >
          {countryName(country, i18n.language)}
        </div>
      )}
      {(tr.up > 0 || tr.down > 0) && (
        <div className="mut num" style={{ fontSize: 12 }}>
          <bdi>{`↑ ${SizeFormatter.sizeFormat(tr.up)} · ↓ ${SizeFormatter.sizeFormat(tr.down)}`}</bdi>
        </div>
      )}
    </>
  );

  const stored = index < p.storedCount;
  const actions = (
    <div className="row" style={{ gap: 4, justifyContent: 'flex-end' }}>
      <IconBtn
        small
        icon="refresh"
        label={t('check')}
        disabled={isUntestable(row) || testing}
        onClick={() => p.onTest(row.key)}
      />
      {row.torCountry ? (
        <IconBtn small icon="tor" label={t('pages.tor.manage')} onClick={p.onShowTor} />
      ) : row.tunnelEngine ? (
        <Link
          to="/tunnels"
          className="ib sm"
          aria-label={t('pages.xray.outbound.manageTunnel')}
          title={t('pages.xray.outbound.manageTunnel')}
        >
          <Icon name="tunnel" />
        </Link>
      ) : (
        <IconBtn small icon="edit" label={t('edit')} onClick={() => p.openEdit(index)} />
      )}
      {stored && (
        <MoreMenu
          label={`${t('pages.nodes.actions')} ${row.tag}`}
          actions={[
            ...(index > 0
              ? [
                  {
                    key: 'top',
                    label: t('pages.xray.outbound.moveToTop'),
                    onSelect: () => p.setFirst(index),
                  },
                ]
              : []),
            {
              key: 'up',
              label: t('pages.inbounds.form.moveUp'),
              disabled: index === 0,
              onSelect: () => p.moveUp(index),
            },
            {
              key: 'down',
              label: t('pages.inbounds.form.moveDown'),
              disabled: index === p.storedCount - 1,
              onSelect: () => p.moveDown(index),
            },
            {
              key: 'reset',
              label: t('pages.inbounds.resetTraffic'),
              icon: 'refresh' as const,
              onSelect: () => p.onResetTraffic(row.tag || ''),
            },
            {
              key: 'del',
              label: t('delete'),
              icon: 'trash' as const,
              danger: true,
              onSelect: () => p.confirmDelete(index),
            },
          ]}
        />
      )}
    </div>
  );
  return { name, type, state, latency, actions };
}

function Row(props: OutboundTableProps & { row: OutboundRow; index: number }) {
  const c = useCells(props, props.row, props.index);
  return (
    <tr className="ob-row">
      <td>{c.name}</td>
      <td>{c.type}</td>
      <td>{c.state}</td>
      <td>{c.latency}</td>
      <td>{c.actions}</td>
    </tr>
  );
}

function Card(props: OutboundTableProps & { row: OutboundRow; index: number }) {
  const c = useCells(props, props.row, props.index);
  return (
    <div
      className="li outbound-card"
      style={{ flexDirection: 'column', alignItems: 'stretch', gap: 8 }}
    >
      <div className="row sp" style={{ alignItems: 'flex-start' }}>
        {c.name}
        {c.type}
      </div>
      <div className="row sp">
        <div>{c.state}</div>
        <div style={{ textAlign: 'end' }}>{c.latency}</div>
      </div>
      {c.actions}
    </div>
  );
}

// The design's outbound table; tunnel and Tor rows are listed but managed on their own pages.
export default function OutboundTable(props: OutboundTableProps) {
  const { t } = useTranslation();
  if (props.narrow) {
    return (
      <div className="list">
        {props.rows.map((row, i) => (
          <Card key={row.key} {...props} row={row} index={i} />
        ))}
      </div>
    );
  }
  return (
    <div className="tblbox">
      <table className="tbl" style={{ minWidth: 560 }}>
        <thead>
          <tr>
            <th>{t('outbounds.colName')}</th>
            <th>{t('outbounds.colType')}</th>
            <th>{t('tunnels.colState')}</th>
            <th>{t('tunnels.colLatency')}</th>
            <th>
              <span style={{ position: 'absolute', width: 1, height: 1, overflow: 'hidden' }}>
                {t('pages.nodes.actions')}
              </span>
            </th>
          </tr>
        </thead>
        <tbody>
          {props.rows.map((row, i) => (
            <Row key={row.key} {...props} row={row} index={i} />
          ))}
        </tbody>
      </table>
    </div>
  );
}
