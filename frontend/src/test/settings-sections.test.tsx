import { screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { afterEach, describe, expect, it, vi } from 'vitest';

import SettingsPage from '@/pages/settings/SettingsPage';
import { HttpUtil, Msg } from '@/utils';
import { renderWithProviders } from './test-utils';

afterEach(() => vi.restoreAllMocks());

describe('settings sections', () => {
  it('lists the real sections and no placeholder for extra admin accounts', async () => {
    vi.spyOn(HttpUtil, 'get').mockResolvedValue(new Msg(true, '', []));
    vi.spyOn(HttpUtil, 'post').mockResolvedValue(new Msg(true, '', { webPort: 2053 }));
    renderWithProviders(
      <MemoryRouter initialEntries={['/settings']}>
        <SettingsPage />
      </MemoryRouter>,
    );

    expect(await screen.findByRole('link', { name: /Appearance/ })).toBeTruthy();
    expect(screen.queryByRole('link', { name: /Users and access/ })).toBeNull();
    expect(screen.queryByText('Soon')).toBeNull();
  });
});
