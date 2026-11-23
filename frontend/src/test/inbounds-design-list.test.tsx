import { fireEvent, screen, within } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { describe, expect, it, vi } from 'vitest';

import { DBInbound } from '@/models/dbinbound';
import InboundList, { inboundState, protocolBadges } from '@/pages/inbounds/list/InboundList';
import type { DBInboundRecord } from '@/pages/inbounds/list/types';
import { HttpUtil, Msg } from '@/utils';
import { renderWithProviders } from './test-utils';

const DAY = 86_400_000;
const NOW = Date.now();

function inbound(over: Record<string, unknown>): DBInboundRecord {
  return new DBInbound({
    id: 1,
    remark: 'main',
    protocol: 'vless',
    port: 443,
    enable: true,
    up: 0,
    down: 0,
    total: 0,
    expiryTime: 0,
    settings: JSON.stringify({ clients: [] }),
    streamSettings: JSON.stringify({ network: 'tcp', security: 'reality' }),
    ...over,
  }) as unknown as DBInboundRecord;
}

function renderList(rows: DBInboundRecord[]) {
  renderWithProviders(
    <MemoryRouter>
      <InboundList
        dbInbounds={rows}
        clientCount={{}}
        onlineClients={[]}
        lastOnlineMap={{}}
        inboundSpeed={{}}
        expireDiff={3 * DAY}
        trafficDiff={0}
        pageSize={0}
        isMobile={false}
        subEnable={false}
        nodesById={new Map()}
        hasActiveNode={false}
        hosts={[]}
        onAddInbound={vi.fn()}
        onGeneralAction={vi.fn()}
        onRowAction={vi.fn()}
        onBulkDelete={vi.fn()}
      />
    </MemoryRouter>,
  );
}

describe('inbound list', () => {
  it('says an inbound needs attention when it is out of traffic, expired or about to expire', () => {
    expect(inboundState(inbound({}), 3 * DAY, NOW)).toBe('ok');
    expect(inboundState(inbound({ enable: false, total: 10, up: 10 }), 3 * DAY, NOW)).toBe('off');
    expect(inboundState(inbound({ total: 10, up: 6, down: 4 }), 3 * DAY, NOW)).toBe('bad');
    expect(inboundState(inbound({ expiryTime: NOW - DAY }), 3 * DAY, NOW)).toBe('bad');
    expect(inboundState(inbound({ expiryTime: NOW + DAY }), 3 * DAY, NOW)).toBe('warn');
  });

  it('labels a VLESS Reality inbound with its transport and security', () => {
    expect(protocolBadges(inbound({}))).toEqual(['vless', 'TCP', 'Reality']);
  });

  it('lists only the inbounds that need attention under that chip', () => {
    renderList([
      inbound({ id: 1, remark: 'main' }),
      inbound({ id: 2, remark: 'old', expiryTime: NOW - DAY }),
    ]);
    fireEvent.click(screen.getByRole('button', { name: /^Needs attention/ }));
    expect(screen.queryByText('main')).toBeNull();
    const row = screen.getByText('old').closest('tr') as HTMLElement;
    expect(within(row).getByText('Expired')).toBeTruthy();
  });

  it('turns an inbound off through the API', async () => {
    const post = vi.spyOn(HttpUtil, 'post').mockResolvedValue(new Msg(true, '', null));
    renderList([inbound({ id: 7, remark: 'main' })]);
    fireEvent.click(screen.getByRole('switch', { name: 'Enabled main' }));
    expect(post.mock.calls[0][0]).toBe('/panel/api/inbounds/setEnable/7');
    expect((post.mock.calls[0][1] as FormData).get('enable')).toBe('false');
  });
});
