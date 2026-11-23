import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import GroupsPage from '@/pages/groups/GroupsPage';
import { HttpUtil, Msg } from '@/utils';
import { renderWithProviders } from './test-utils';

const groups = [
  {
    name: 'Family',
    clientCount: 3,
    trafficUsed: 3 * 1024 ** 3,
    up: 1024 ** 3,
    down: 2 * 1024 ** 3,
  },
  { name: 'Gaming', clientCount: 1, trafficUsed: 1024 ** 3, up: 0, down: 1024 ** 3 },
];

function openGroups() {
  renderWithProviders(
    <MemoryRouter>
      <GroupsPage />
    </MemoryRouter>,
  );
}

describe('Groups page', () => {
  beforeEach(() => {
    vi.spyOn(HttpUtil, 'get').mockImplementation(
      async (url) => new Msg(true, '', url === '/panel/api/clients/groups' ? groups : []),
    );
    vi.spyOn(HttpUtil, 'post').mockResolvedValue(new Msg(true, '', { affected: 0 }));
  });

  it('shows each group with its members and its share of all group traffic', async () => {
    openGroups();
    const family = await screen.findByRole('region', { name: 'Family' });
    expect(within(family).getByText('3')).toBeTruthy();
    expect(within(family).getByText('75%')).toBeTruthy();
    expect(within(family).getByText(/^3\.00 GB · ↑ 1\.00 GB/)).toBeTruthy();
    expect(within(screen.getByRole('region', { name: 'Gaming' })).getByText('25%')).toBeTruthy();
  });

  it('deletes a group only after the dialog is confirmed', async () => {
    openGroups();
    await screen.findByRole('region', { name: 'Gaming' });
    fireEvent.click(screen.getByRole('button', { name: 'Actions Gaming' }));
    fireEvent.click(await screen.findByRole('menuitem', { name: 'Delete group (keep clients)' }));
    expect(HttpUtil.post).not.toHaveBeenCalledWith(
      '/panel/api/clients/groups/delete',
      expect.anything(),
      expect.anything(),
    );

    fireEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Delete' }));
    await waitFor(() =>
      expect(HttpUtil.post).toHaveBeenCalledWith(
        '/panel/api/clients/groups/delete',
        { name: 'Gaming' },
        { headers: { 'Content-Type': 'application/json' } },
      ),
    );
  });
});
