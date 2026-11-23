import { screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { afterEach, describe, expect, it, vi } from 'vitest';

import NodeTabs from '@/components/dl/shell/NodeTabs';
import { HttpUtil, Msg } from '@/utils';
import { renderWithProviders } from './test-utils';

describe('this server tab', () => {
  afterEach(() => vi.restoreAllMocks());

  it('shows the name chosen in the first-run wizard instead of the generic label', async () => {
    vi.spyOn(HttpUtil, 'post').mockImplementation(async (url) =>
      url === '/panel/api/setting/all'
        ? new Msg(true, '', { serverName: 'Frankfurt 1', serverCountry: 'DE' })
        : new Msg(true, '', {}),
    );
    vi.spyOn(HttpUtil, 'get').mockResolvedValue(new Msg(true, '', []));
    renderWithProviders(
      <MemoryRouter>
        <NodeTabs />
      </MemoryRouter>,
    );
    expect(await screen.findByRole('button', { name: /Frankfurt 1/ })).toBeTruthy();
    expect(screen.queryByText('This server')).toBeNull();
  });
});
