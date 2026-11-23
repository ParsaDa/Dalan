import { screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { afterEach, describe, expect, it, vi } from 'vitest';

import MorePage from '@/pages/more/MorePage';
import { HttpUtil, Msg } from '@/utils';
import { renderWithProviders } from './test-utils';

describe('phone More page', () => {
  afterEach(() => vi.restoreAllMocks());

  it('reaches every page the tab bar has no room for, with live counts and states', async () => {
    vi.spyOn(HttpUtil, 'post').mockResolvedValue(new Msg(true, '', { twoFactorEnable: true }));
    vi.spyOn(HttpUtil, 'get').mockResolvedValue(
      new Msg(true, '', [{ id: 1 }, { id: 2 }, { id: 3 }]),
    );
    renderWithProviders(
      <MemoryRouter>
        <MorePage />
      </MemoryRouter>,
    );
    const link = (name: RegExp) => screen.getByRole('link', { name }).getAttribute('href');
    expect(await screen.findByText('2-step on')).toBeTruthy();
    expect(link(/Inbounds/)).toBe('/inbounds');
    expect(screen.getByRole('link', { name: /Inbounds/ }).textContent).toContain('3');
    expect(screen.queryByRole('link', { name: /Tor and PIA/ })).toBeNull();
    expect(link(/^Logs/)).toBe('/logs');
    expect(link(/Backups/)).toBe('/settings#backups');
    expect(link(/Security/)).toBe('/settings#security');
  });
});
