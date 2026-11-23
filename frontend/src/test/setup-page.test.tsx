import { fireEvent, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { useAllSettings } from '@/api/queries/useAllSettings';
import SetupPage from '@/pages/setup/SetupPage';
import { HttpUtil, Msg } from '@/utils';
import { renderWithProviders } from './test-utils';

const PROFILE = '/panel/api/setting/profile';
const ALL = '/panel/api/setting/all';

function postCalls(url: string) {
  return vi.mocked(HttpUtil.post).mock.calls.filter((c) => c[0] === url);
}

// Stands in for the dashboard redirect: it holds the settings query and records what it saw first.
const seenOnArrival: boolean[] = [];

function Holder() {
  useAllSettings();
  return null;
}

function Dashboard() {
  const { allSetting } = useAllSettings();
  if (seenOnArrival.length === 0) seenOnArrival.push(allSetting.setupDone);
  return <h1>Dashboard</h1>;
}

function renderWizard() {
  seenOnArrival.length = 0;
  return renderWithProviders(
    <MemoryRouter initialEntries={['/setup']}>
      <Holder />
      <Routes>
        <Route path="/setup" element={<SetupPage />} />
        <Route path="/" element={<Dashboard />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe('first-run wizard', () => {
  let setupDone: boolean;

  beforeEach(() => {
    vi.restoreAllMocks();
    setupDone = false;
    vi.spyOn(HttpUtil, 'get').mockResolvedValue(new Msg(true, '', []));
    vi.spyOn(HttpUtil, 'post').mockImplementation(async (url) => {
      if (url === ALL) {
        const snapshot = setupDone;
        await new Promise((r) => setTimeout(r, 25));
        return new Msg(true, '', { webPort: 2053, setupDone: snapshot });
      }
      if (url === PROFILE) {
        setupDone = true;
        return new Msg(true, '', null);
      }
      return new Msg(true, '', {});
    });
  });

  it('saves name, country and address on their own, then opens the dashboard', async () => {
    renderWizard();
    fireEvent.change(screen.getByLabelText('Server name'), { target: { value: 'Frankfurt 1' } });
    fireEvent.change(screen.getByLabelText('Country'), { target: { value: 'DE' } });
    fireEvent.change(screen.getByLabelText('Public address'), {
      target: { value: '203.0.113.7' },
    });
    await waitFor(() => expect(postCalls(ALL).length).toBeGreaterThan(0));
    fireEvent.click(screen.getByRole('button', { name: 'Save' }));

    expect(await screen.findByRole('heading', { name: 'Dashboard' })).toBeTruthy();
    expect(postCalls(PROFILE)[0][1]).toEqual({
      serverName: 'Frankfurt 1',
      serverCountry: 'DE',
      serverAddress: '203.0.113.7',
      setupDone: true,
    });
    expect(postCalls('/panel/api/inbounds/add')).toHaveLength(0);
    expect(postCalls('/panel/api/setting/update')).toHaveLength(0);
    // The dashboard must see setupDone=true on arrival, or it sends the person straight back here.
    expect(seenOnArrival).toEqual([true]);
  });

  it('skips from the one step by marking setup done and leaving the name alone', async () => {
    renderWizard();
    fireEvent.click(screen.getByRole('button', { name: 'Set up later' }));

    expect(await screen.findByRole('heading', { name: 'Dashboard' })).toBeTruthy();
    expect(postCalls(PROFILE)[0][1]).toEqual({ setupDone: true });
  });

  it('stays on the page and shows the server message when the save is refused', async () => {
    vi.mocked(HttpUtil.post).mockImplementation(async (url) =>
      url === PROFILE
        ? new Msg(false, 'server address must be an IP address or a domain', null)
        : new Msg(true, '', { webPort: 2053, setupDone: false }),
    );
    renderWizard();
    fireEvent.change(screen.getByLabelText('Public address'), {
      target: { value: 'https://x.example/a' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Save' }));

    expect((await screen.findByRole('alert')).textContent).toContain(
      'server address must be an IP address or a domain',
    );
    expect(screen.queryByRole('heading', { name: 'Dashboard' })).toBeNull();
  });
});
