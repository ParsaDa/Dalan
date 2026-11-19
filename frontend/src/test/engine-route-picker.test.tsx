import { useState } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, screen, waitFor } from '@testing-library/react';

import EngineRoutePicker from '@/components/engine/EngineRoutePicker';
import { engineRouteReady, type EngineRoute } from '@/schemas/engineRoute';
import { HttpUtil, Msg } from '@/utils';

import { chooseSelectOption, listSelectOptions, renderWithProviders } from './test-utils';

const choices = [
  { via: 'panel', value: '', kind: 'panel' },
  { via: 'direct', value: '', kind: 'direct' },
  { via: 'outbound', value: 'tunnel-de1', kind: 'tunnel' },
  { via: 'outbound', value: 'tor-de', kind: 'tor' },
  { via: 'outbound', value: 'warp', kind: 'outbound' },
];

let seen: EngineRoute[];

function Harness({ start }: { start?: EngineRoute }) {
  const [route, setRoute] = useState<EngineRoute>(start ?? { via: 'panel', value: '' });
  return (
    <EngineRoutePicker
      id="route"
      value={route}
      onChange={(next) => {
        seen.push(next);
        setRoute(next);
      }}
    />
  );
}

beforeEach(() => {
  seen = [];
  vi.spyOn(HttpUtil, 'get').mockImplementation(async (url: string) =>
    url === '/panel/api/tunnels/engine/routes'
      ? new Msg(true, '', choices as never)
      : new Msg(false, `Unexpected GET ${url}`, null),
  );
});
afterEach(() => vi.restoreAllMocks());

describe('EngineRoutePicker', () => {
  it('offers the panel setting, direct, every outbound the server lists and a link of your own', async () => {
    renderWithProviders(<Harness />);
    await waitFor(() => expect(listSelectOptions('route')).toContain('tor-de'));
    expect(listSelectOptions('route')).toEqual([
      'As the panel is set',
      'Direct from this server',
      'tunnel-de1',
      'tor-de',
      'warp',
      'A proxy address or share link',
    ]);
  });

  it('hands back an outbound as its tag and shows no address box for it', async () => {
    renderWithProviders(<Harness />);
    await waitFor(() => expect(listSelectOptions('route')).toContain('tor-de'));
    chooseSelectOption('route', 'tor-de');
    expect(seen.at(-1)).toEqual({ via: 'outbound', value: 'tor-de' });
    expect(screen.queryByRole('textbox', { name: 'Proxy address or share link' })).toBeNull();
  });

  it('asks for the address when a link is chosen and passes what is typed', async () => {
    renderWithProviders(<Harness />);
    chooseSelectOption('route', 'A proxy address or share link');
    expect(seen.at(-1)).toEqual({ via: 'link', value: '' });
    fireEvent.change(
      await screen.findByRole('textbox', {
        name: 'Proxy address or share link',
      }),
      {
        target: { value: 'vless://id@203.0.113.7:443' },
      },
    );
    expect(seen.at(-1)).toEqual({
      via: 'link',
      value: 'vless://id@203.0.113.7:443',
    });
  });

  it('still offers panel, direct and a link when the list cannot be read', async () => {
    vi.spyOn(HttpUtil, 'get').mockResolvedValue(new Msg(false, 'down', null));
    renderWithProviders(<Harness />);
    await waitFor(() => expect(HttpUtil.get).toHaveBeenCalled());
    expect(listSelectOptions('route')).toEqual([
      'As the panel is set',
      'Direct from this server',
      'A proxy address or share link',
    ]);
  });
});

describe('engineRouteReady', () => {
  it('needs an address only for a link and a tag only for an outbound', () => {
    expect(engineRouteReady({ via: 'panel', value: '' })).toBe(true);
    expect(engineRouteReady({ via: 'direct', value: '' })).toBe(true);
    expect(engineRouteReady({ via: 'outbound', value: 'tor-de' })).toBe(true);
    expect(engineRouteReady({ via: 'outbound', value: '' })).toBe(false);
    expect(engineRouteReady({ via: 'link', value: '  ' })).toBe(false);
    expect(engineRouteReady({ via: 'link', value: 'socks5://127.0.0.1:1080' })).toBe(true);
  });
});
