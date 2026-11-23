import { fireEvent, screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import ClientTable, { clientProtocols, type ClientTableProps } from '@/pages/clients/ClientTable';
import type { ClientRecord } from '@/schemas/client';
import { renderWithProviders } from './test-utils';

const inboundsById = {
  1: { id: 1, protocol: 'vless', tag: 'in-1', remark: 'main' },
  2: { id: 2, protocol: 'trojan', tag: 'in-2', remark: 'cdn' },
  3: { id: 3, protocol: 'vless', tag: 'in-3', remark: 'backup' },
} as unknown as ClientTableProps['inboundsById'];

function client(over: Partial<ClientRecord>): ClientRecord {
  return {
    email: 'sara',
    enable: true,
    inboundIds: [1],
    totalGB: 0,
    expiryTime: 0,
    ...over,
  } as ClientRecord;
}

function renderTable(rows: ClientRecord[], over: Partial<ClientTableProps> = {}) {
  const props: ClientTableProps = {
    rows,
    narrow: false,
    selected: [],
    inboundsById,
    clientSpeed: {},
    togglingEmail: null,
    bucketOf: (r) => (r.enable ? 'active' : 'deactive'),
    isOnline: () => false,
    onSelect: vi.fn(),
    onSelectAll: vi.fn(),
    onEdit: vi.fn(),
    onShowQr: vi.fn(),
    onShowInfo: vi.fn(),
    onResetTraffic: vi.fn(),
    onDelete: vi.fn(),
    onToggle: vi.fn(),
    onGroup: vi.fn(),
    ...over,
  };
  renderWithProviders(<ClientTable {...props} />);
  return props;
}

describe('client table', () => {
  it('lists each protocol of the attached inbounds once', () => {
    expect(clientProtocols([1, 2, 3, 9], inboundsById)).toEqual(['vless', 'trojan']);
  });

  it('shows a paused client as paused and offers to resume it', () => {
    const props = renderTable([client({ email: 'mina', enable: false })]);
    const row = screen.getByText('mina').closest('tr') as HTMLElement;
    expect(within(row).getByText('Paused')).toBeTruthy();
    fireEvent.click(within(row).getByRole('button', { name: 'Actions mina' }));
    fireEvent.click(screen.getByRole('menuitem', { name: 'Resume' }));
    expect(props.onToggle).toHaveBeenCalledWith(expect.objectContaining({ email: 'mina' }), true);
  });

  it('shows usage against the limit, or unlimited', () => {
    renderTable([
      client({
        email: 'a',
        totalGB: 50 * 1024 ** 3,
        traffic: { up: 4 * 1024 ** 3, down: 30 * 1024 ** 3 } as ClientRecord['traffic'],
      }),
      client({ email: 'b' }),
    ]);
    const a = screen.getByText('a').closest('tr') as HTMLElement;
    expect(within(a).getByText('34.00 GB')).toBeTruthy();
    expect(within(a).getByText('50.00 GB')).toBeTruthy();
    expect(
      within(screen.getByText('b').closest('tr') as HTMLElement).getByText('Unlimited'),
    ).toBeTruthy();
  });

  it('filters by a client group when the group name is clicked', () => {
    const props = renderTable([client({ email: 'reza', group: 'Office' })]);
    fireEvent.click(screen.getByRole('button', { name: 'Office' }));
    expect(props.onGroup).toHaveBeenCalledWith('Office');
  });
});
