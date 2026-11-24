import { fireEvent, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { SpeedtestTile } from '@/pages/index/dashboard/SpeedtestTile';
import { HttpUtil, Msg } from '@/utils';
import { renderWithProviders } from './test-utils';

const finished = {
  server: {
    id: '1',
    name: 'Tehran',
    sponsor: 'Example ISP',
    country: 'Iran',
    host: 'st.example.ir:8080',
    distance: 12.5,
  },
  pingMs: 14.2,
  jitterMs: 1.3,
  downMbps: 412.6,
  upMbps: 188.9,
  at: 1795000000000,
};

const idle = {
  running: false,
  phase: '',
  progress: 0,
  mbps: 0,
  pingMs: 0,
  result: null,
  error: '',
};

function serve(status: object, history: object[] = []) {
  vi.spyOn(HttpUtil, 'get').mockImplementation(async (url: string) => {
    if (url.endsWith('/speedtest/status')) return new Msg(true, '', status);
    if (url.endsWith('/speedtest/history')) return new Msg(true, '', history);
    return new Msg(true, '', null);
  });
  return vi.spyOn(HttpUtil, 'post').mockResolvedValue(new Msg(true, '', null));
}

afterEach(() => vi.restoreAllMocks());

describe('dashboard speed test card', () => {
  it('shows the last finished test and starts a new one from the button', async () => {
    const post = serve(idle, [finished]);
    renderWithProviders(<SpeedtestTile />);

    expect(await screen.findByText('412.6')).toBeTruthy();
    expect(screen.getByText('188.9')).toBeTruthy();
    expect(screen.getByText('14.2')).toBeTruthy();
    expect(screen.getByText(/Example ISP/)).toBeTruthy();

    fireEvent.click(screen.getByRole('button', { name: 'Run speed test' }));
    await waitFor(() => expect(post).toHaveBeenCalledWith('/panel/api/server/speedtest/start'));
  });

  it('shows the live rate and phase while a test runs and blocks a second start', async () => {
    serve({ ...idle, running: true, phase: 'download', progress: 0.4, mbps: 233.4, pingMs: 15 });
    renderWithProviders(<SpeedtestTile />);

    expect(await screen.findByText('233.4')).toBeTruthy();
    expect(screen.getByText('Testing download…')).toBeTruthy();
    const button = screen.getByRole('button', { name: 'Testing…' }) as HTMLButtonElement;
    expect(button.disabled).toBe(true);
  });

  it('names a tunnel test as the link between the two servers, not a speedtest.net host', async () => {
    const link = {
      ...finished,
      server: { ...finished.server, id: 'link', sponsor: '', name: 'tunnel link' },
    };
    serve(idle, [link]);
    renderWithProviders(<SpeedtestTile />);

    expect(await screen.findByText('412.6')).toBeTruthy();
    expect(screen.getByText('Speed between this server and the exit server')).toBeTruthy();
    expect(screen.queryByText(/tunnel link/)).toBeNull();
  });

  it('tells the person when speedtest.net could not be reached', async () => {
    serve({ ...idle, error: 'could not get the server list from speedtest.net: status 503' });
    renderWithProviders(<SpeedtestTile />);

    expect(await screen.findByText(/could not get the server list/)).toBeTruthy();
  });
});
