import { fireEvent, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import HostsPage from '@/pages/hosts/HostsPage';
import { ClipboardManager, HttpUtil, Msg } from '@/utils';
import { renderWithProviders } from './test-utils';

vi.mock('@/pages/hosts/HostFormModal', () => ({ default: () => null }));

const hosts = [
  {
    groupId: 'a',
    remark: 'cdn',
    sortOrder: 0,
    inboundIds: [],
    hosts: ['cdn.example.ir'],
    port: 443,
  },
  { groupId: 'b', remark: 'old', sortOrder: 1, inboundIds: [], hosts: [], isDisabled: true },
];

describe('Hosts page', () => {
  beforeEach(() => {
    vi.spyOn(HttpUtil, 'get').mockImplementation(
      async (url) => new Msg(true, '', url.includes('/hosts/list') ? hosts : []),
    );
  });

  it('lists only the hosts in the chosen state', async () => {
    renderWithProviders(
      <MemoryRouter>
        <HostsPage />
      </MemoryRouter>,
    );
    await screen.findByText('cdn');

    fireEvent.click(screen.getByRole('button', { name: /^Off/ }));
    expect(screen.queryByText('cdn')).toBeNull();
    expect(screen.getByText('old')).toBeTruthy();
    expect(screen.getByText('Inherits')).toBeTruthy();
  });

  it('copies the first address of a host', async () => {
    const copy = vi.spyOn(ClipboardManager, 'copyText').mockResolvedValue(true);
    renderWithProviders(
      <MemoryRouter>
        <HostsPage />
      </MemoryRouter>,
    );
    await screen.findByText('cdn');

    fireEvent.click(screen.getAllByRole('button', { name: 'Copy address' })[0]);
    expect(copy).toHaveBeenCalledWith('cdn.example.ir');
    expect(
      (screen.getAllByRole('button', { name: 'Copy address' })[1] as HTMLButtonElement).disabled,
    ).toBe(true);
  });
});
