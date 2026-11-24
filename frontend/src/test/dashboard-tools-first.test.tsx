import { screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { MemoryRouter } from 'react-router';

import { keys } from '@/api/queryKeys';
import { Status } from '@/models/status';
import DashboardPage from '@/pages/index/dashboard/DashboardPage';
import { HttpUtil, Msg } from '@/utils';
import { makeTestQueryClient, renderWithProviders } from './test-utils';

describe('dashboard System Tools card', () => {
  it('is the first card of the local view, ahead of server health', async () => {
    vi.spyOn(HttpUtil, 'get').mockResolvedValue(new Msg(true, '', []));
    vi.spyOn(HttpUtil, 'post').mockResolvedValue(new Msg(true, '', []));
    const queryClient = makeTestQueryClient();
    queryClient.setQueryDefaults(keys.server.status(), {
      staleTime: Infinity,
      refetchInterval: false,
    });
    queryClient.setQueryData(keys.server.status(), new Status({}));
    renderWithProviders(
      <MemoryRouter>
        <DashboardPage tools={<section aria-label="System tools probe" className="tile s12" />} />
      </MemoryRouter>,
      { queryClient },
    );

    const tools = await screen.findByLabelText('System tools probe');
    await waitFor(() => expect(screen.getByLabelText('Server health')).toBeTruthy());
    const health = screen.getByLabelText('Server health');
    expect(tools.compareDocumentPosition(health) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });
});
