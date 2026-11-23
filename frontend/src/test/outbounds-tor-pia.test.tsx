import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import TorPiaTiles from '@/pages/xray/outbounds/TorPiaTiles';
import { HttpUtil, Msg } from '@/utils';
import { renderWithProviders } from './test-utils';

const status = (state: string) => ({
  state,
  progress: 100,
  latencyMs: 400,
  exitIp: '',
  exitLoc: '',
  error: '',
});
const exits = [
  { id: 1, country: 'de', port: 62701, enable: true, tag: 'tor-de', status: status('connected') },
  { id: 2, country: 'nl', port: 62702, enable: true, tag: 'tor-nl', status: status('failed') },
];

describe('Tor and PIA tiles on the Outbounds page', () => {
  it('counts the exits that are on and the ones connected now, and opens PIA setup', async () => {
    vi.spyOn(HttpUtil, 'get').mockImplementation(
      async (url) =>
        new Msg(
          true,
          '',
          url === '/panel/api/tor/list'
            ? exits
            : url.includes('engine')
              ? { installed: true, version: '16.0.1', pinned: '16.0.1' }
              : '',
        ),
    );
    const onOpenPia = vi.fn();
    renderWithProviders(<TorPiaTiles onOpenPia={onOpenPia} />);
    const tor = await screen.findByRole('region', { name: 'Tor' });
    await within(tor).findByText('exit countries on');
    await waitFor(() =>
      expect(within(tor).getByText('exit countries on').previousElementSibling?.textContent).toBe(
        '2',
      ),
    );
    expect(within(tor).getByText('connected now').previousElementSibling?.textContent).toBe('1');

    fireEvent.click(
      within(screen.getByRole('region', { name: 'PIA' })).getByRole('button', { name: 'PIA' }),
    );
    expect(onOpenPia).toHaveBeenCalledTimes(1);
  });
});
