import { fireEvent, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import SetupPage from '@/pages/setup/SetupPage';
import { HttpUtil, Msg } from '@/utils';
import { renderWithProviders } from './test-utils';

function postCalls(url: string) {
  return vi.mocked(HttpUtil.post).mock.calls.filter((c) => c[0] === url);
}

describe('first-run wizard', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    vi.spyOn(HttpUtil, 'get').mockImplementation(async (url) =>
      url === '/panel/api/server/getNewX25519Cert'
        ? new Msg(true, '', { privateKey: 'priv', publicKey: 'pub' })
        : new Msg(true, '', {}),
    );
    vi.spyOn(HttpUtil, 'post').mockImplementation(async (url) => {
      if (url === '/panel/api/setting/all')
        return new Msg(true, '', { webPort: 2053, subEnable: true });
      if (url === '/panel/api/setting/defaultSettings')
        return new Msg(true, '', { subEnable: true, subURI: 'https://sub.example/s/' });
      return new Msg(true, '', {});
    });
  });

  it('walks the three steps and creates a Reality inbound, then shows the link', async () => {
    renderWithProviders(
      <MemoryRouter>
        <SetupPage />
      </MemoryRouter>,
    );
    fireEvent.change(screen.getByLabelText('Server name'), { target: { value: 'Frankfurt 1' } });
    fireEvent.change(screen.getByLabelText('Country'), { target: { value: 'DE' } });
    fireEvent.click(screen.getByRole('button', { name: 'Continue' }));
    expect(
      screen.getByRole('radio', { name: /Best for most people/ }).getAttribute('aria-checked'),
    ).toBe('true');
    expect((screen.getByRole('radio', { name: /Low ping/ }) as HTMLButtonElement).disabled).toBe(
      true,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Continue' }));
    fireEvent.change(screen.getByLabelText('Name'), { target: { value: 'sara.k' } });
    await waitFor(() => expect(postCalls('/panel/api/setting/all').length).toBeGreaterThan(0));
    fireEvent.click(screen.getByRole('button', { name: 'Create and finish' }));

    expect(await screen.findByRole('heading', { name: 'You’re live' })).toBeTruthy();
    const [add] = postCalls('/panel/api/inbounds/add');
    const body = add[1] as {
      protocol: string;
      port: number;
      settings: string;
      streamSettings: string;
    };
    expect(body.protocol).toBe('vless');
    expect(body.port).toBe(443);
    expect(JSON.parse(body.settings).clients[0].email).toBe('sara.k');
    expect(JSON.parse(body.streamSettings).realitySettings.privateKey).toBe('priv');
    const [save] = postCalls('/panel/api/setting/update');
    expect(save[1]).toMatchObject({
      serverName: 'Frankfurt 1',
      serverCountry: 'DE',
      setupDone: true,
    });
    const subId = JSON.parse(body.settings).clients[0].subId;
    expect(screen.getByText(`https://sub.example/s/${subId}`)).toBeTruthy();
  });

  it('keeps the user on the last step and shows why when the inbound is refused', async () => {
    vi.mocked(HttpUtil.post).mockImplementation(async (url) =>
      url === '/panel/api/inbounds/add'
        ? new Msg(false, 'Port 443 is already in use', null)
        : new Msg(true, '', {}),
    );
    renderWithProviders(
      <MemoryRouter>
        <SetupPage />
      </MemoryRouter>,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Continue' }));
    fireEvent.click(screen.getByRole('button', { name: 'Continue' }));
    fireEvent.click(screen.getByRole('button', { name: 'Create and finish' }));
    expect((await screen.findByRole('alert')).textContent).toContain('Port 443 is already in use');
    expect(postCalls('/panel/api/setting/update')).toHaveLength(0);
  });
});
