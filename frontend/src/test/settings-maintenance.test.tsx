import { fireEvent, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import BackupsSection from '@/pages/settings/maintenance/BackupsSection';
import UpdatesSection from '@/pages/settings/maintenance/UpdatesSection';
import { HttpUtil, Msg } from '@/utils';
import { renderWithProviders } from './test-utils';

describe('Settings › Updates', () => {
  afterEach(() => vi.restoreAllMocks());

  it('shows the running and newest panel versions when an update is out', async () => {
    vi.spyOn(HttpUtil, 'get').mockImplementation(async (url) =>
      url === '/panel/api/server/getPanelUpdateInfo'
        ? new Msg(true, '', {
            currentVersion: '0.6.0',
            latestVersion: '0.6.1',
            updateAvailable: true,
          })
        : new Msg(true, '', {}),
    );
    vi.spyOn(HttpUtil, 'post').mockResolvedValue(new Msg(true, '', {}));
    renderWithProviders(<UpdatesSection />);
    expect(await screen.findByText('Update available')).toBeTruthy();
    expect(screen.getByText('v0.6.1')).toBeTruthy();
  });

  it('says the panel is up to date when nothing newer exists', async () => {
    vi.spyOn(HttpUtil, 'get').mockImplementation(async (url) =>
      url === '/panel/api/server/getPanelUpdateInfo'
        ? new Msg(true, '', {
            currentVersion: '0.6.1',
            latestVersion: '0.6.1',
            updateAvailable: false,
          })
        : new Msg(true, '', {}),
    );
    vi.spyOn(HttpUtil, 'post').mockResolvedValue(new Msg(true, '', {}));
    renderWithProviders(<UpdatesSection />);
    expect(await screen.findByText(/up to date/)).toBeTruthy();
    expect(screen.queryByText('Update available')).toBeNull();
  });
});

describe('Settings › Backups', () => {
  afterEach(() => vi.restoreAllMocks());

  it('restores a database without keeping this host’s settings once the switch is off', async () => {
    const file = new File(['db'], 'x-ui.db');
    vi.spyOn(HTMLInputElement.prototype, 'click').mockImplementation(function (
      this: HTMLInputElement,
    ) {
      Object.defineProperty(this, 'files', { value: [file] });
      this.dispatchEvent(new Event('change'));
    });
    const post = vi.spyOn(HttpUtil, 'post').mockResolvedValue(new Msg(false, 'bad file', null));
    renderWithProviders(<BackupsSection />);
    fireEvent.click(screen.getByRole('switch', { name: "Keep this machine's settings" }));
    fireEvent.click(screen.getByRole('button', { name: 'Restore' }));
    await waitFor(() => expect(post).toHaveBeenCalled());
    const form = post.mock.calls[0][1] as FormData;
    expect(post.mock.calls[0][0]).toBe('/panel/api/server/importDB');
    expect(form.get('keepHostSettings')).toBe('false');
    expect(form.get('db')).toBe(file);
  });
});
