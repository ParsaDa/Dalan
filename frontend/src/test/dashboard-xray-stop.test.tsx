import { fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { MemoryRouter } from 'react-router';

import { keys } from '@/api/queryKeys';
import { Status } from '@/models/status';
import { HealthTile } from '@/pages/index/dashboard/DashboardPage';
import { HttpUtil } from '@/utils';
import { makeTestQueryClient, renderWithProviders } from './test-utils';

// Stop used to fire on the first click, so a mis-click killed every proxy connection (#7103).
describe('dashboard Xray switch', () => {
  it('does not stop Xray until the dialog is confirmed', async () => {
    const queryClient = makeTestQueryClient();
    queryClient.setQueryDefaults(keys.server.status(), {
      staleTime: Infinity,
      refetchInterval: false,
    });
    queryClient.setQueryData(
      keys.server.status(),
      new Status({ xray: { state: 'running', errorMsg: '', version: '25.1' } }),
    );
    const post = vi
      .spyOn(HttpUtil, 'post')
      .mockResolvedValue({ success: true, msg: '', obj: null });
    renderWithProviders(
      <MemoryRouter>
        <HealthTile restart={() => {}} />
      </MemoryRouter>,
      { queryClient },
    );
    fireEvent.click(screen.getByRole('switch', { name: 'Xray core' }));
    expect(post).not.toHaveBeenCalled();
    fireEvent.click(await screen.findByRole('button', { name: /Stop Xray/ }));
    await waitFor(() => expect(post).toHaveBeenCalledWith('/panel/api/server/stopXrayService'));
    post.mockRestore();
  });
});
