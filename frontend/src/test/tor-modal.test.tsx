import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import i18next from 'i18next';

import TorModal from '@/pages/xray/overrides/TorModal';
import { HttpUtil, Msg } from '@/utils';

import { renderWithProviders } from './test-utils';

interface FakeExit {
  id: number;
  country: string;
  port: number;
  enable: boolean;
  tag: string;
  status: {
    state: string;
    progress: number;
    latencyMs: number;
    exitIp: string;
    exitLoc: string;
    error: string;
  };
}

function exit(id: number, country: string, state = 'connected'): FakeExit {
  return {
    id,
    country,
    port: 62700 + id,
    enable: true,
    tag: `tor-${country}`,
    status: {
      state,
      progress: 100,
      latencyMs: 900,
      exitIp: '',
      exitLoc: country.toUpperCase(),
      error: '',
    },
  };
}

interface Server {
  exits: FakeExit[];
  installed: boolean;
  bridges: string;
  addError: string;
  bridgesError: string;
  tests: Array<{ country: string; resolve: (m: Msg) => void }>;
  posts: Array<{ url: string; body: unknown }>;
}

function fakeServer(over: Partial<Server> = {}): Server {
  const server: Server = {
    exits: [],
    installed: true,
    bridges: '',
    addError: '',
    bridgesError: '',
    tests: [],
    posts: [],
    ...over,
  };
  vi.spyOn(HttpUtil, 'get').mockImplementation(async (url: string) => {
    if (url === '/panel/api/tor/list') return new Msg(true, '', server.exits as never);
    if (url === '/panel/api/tor/engine') {
      return new Msg(true, '', {
        engine: 'tor',
        installed: server.installed,
        version: server.installed ? '16.0.1' : '',
        pinned: '16.0.1',
      } as never);
    }
    if (url === '/panel/api/tor/bridges')
      return new Msg(true, '', { bridges: server.bridges } as never);
    return new Msg(false, `Unexpected GET ${url}`, null);
  });
  vi.spyOn(HttpUtil, 'post').mockImplementation(async (url: string, body?: unknown) => {
    server.posts.push({ url, body });
    if (url === '/panel/api/tor/add') {
      if (server.addError) return new Msg(false, server.addError, null);
      const country = (body as { country: string }).country;
      server.exits.push(exit(server.exits.length + 1, country));
      return new Msg(true, '', null);
    }
    if (url.startsWith('/panel/api/tor/del/')) {
      const id = Number(url.split('/').pop());
      server.exits = server.exits.filter((e) => e.id !== id);
      return new Msg(true, '', null);
    }
    if (url === '/panel/api/tor/test') {
      return new Promise<Msg>((resolve) => {
        server.tests.push({ country: (body as { country: string }).country, resolve });
      });
    }
    if (url === '/panel/api/tor/engine/install') {
      server.installed = true;
      return new Msg(true, '', null);
    }
    if (url === '/panel/api/tor/bridges') {
      if (server.bridgesError) return new Msg(false, server.bridgesError, null);
      server.bridges = (body as { bridges: string }).bridges;
      return new Msg(true, '', null);
    }
    return new Msg(false, `Unexpected POST ${url}`, null);
  });
  return server;
}

function row(country: string): HTMLElement {
  const el = document.querySelector<HTMLElement>(`[data-country="${country}"]`);
  if (!el) throw new Error(`no row for ${country}`);
  return el;
}

async function openModal() {
  renderWithProviders(<TorModal open onClose={vi.fn()} onChanged={vi.fn()} />);
  await waitFor(() => expect(document.querySelector('[data-country="de"]')).not.toBeNull());
}

beforeEach(() => {
  vi.restoreAllMocks();
});
afterEach(async () => {
  await i18next.changeLanguage('en-US');
});

describe('TorModal', () => {
  it('lists countries with a flag and the English name only, even in a Persian panel', async () => {
    await i18next.changeLanguage('fa-IR');
    fakeServer();
    await openModal();

    const de = row('de');
    expect(de.textContent).toContain('🇩🇪');
    expect(de.textContent).toContain('Germany');
    expect(row('gb').textContent).toContain('United Kingdom');
    expect(document.querySelector('.tor-country-list')?.textContent).not.toMatch(/[؀-ۿ]/);
    expect(document.querySelectorAll('[data-country]').length).toBeGreaterThanOrEqual(200);
  });

  it('filters the list as the user types a name or a code', async () => {
    fakeServer();
    await openModal();

    fireEvent.change(screen.getByPlaceholderText('Search country'), { target: { value: 'germ' } });
    await waitFor(() => expect(document.querySelectorAll('[data-country]').length).toBe(1));
    expect(row('de').textContent).toContain('Germany');

    fireEvent.change(screen.getByPlaceholderText('Search country'), { target: { value: 'nl' } });
    await waitFor(() => expect(document.querySelector('[data-country="nl"]')).not.toBeNull());
    expect(document.querySelector('[data-country="de"]')).toBeNull();
  });

  it('tests one country at a time: spinner on that row, others locked, then a green or red answer', async () => {
    const server = fakeServer();
    await openModal();

    fireEvent.click(within(row('de')).getByRole('button', { name: 'Test' }));
    await waitFor(() => expect(server.tests.length).toBe(1));
    expect(server.tests[0].country).toBe('de');
    expect(within(row('de')).getByRole('button', { name: /Test/ }).className).toContain(
      'ant-btn-loading',
    );
    expect(
      (within(row('fr')).getByRole('button', { name: 'Test' }) as HTMLButtonElement).disabled,
    ).toBe(true);
    expect(within(row('fr')).getByRole('button', { name: 'Test' }).className).not.toContain(
      'ant-btn-loading',
    );

    await act(async () => {
      server.tests[0].resolve(
        new Msg(true, '', {
          ok: true,
          latencyMs: 1400,
          exitIp: '203.0.113.9',
          exitLoc: 'DE',
          reason: '',
          error: '',
        } as never),
      );
    });
    await waitFor(() => expect(row('de').textContent).toContain('1400 ms'));
    expect(row('de').textContent).toContain('Works');
    expect(
      (within(row('fr')).getByRole('button', { name: 'Test' }) as HTMLButtonElement).disabled,
    ).toBe(false);

    fireEvent.click(within(row('aq')).getByRole('button', { name: 'Test' }));
    await waitFor(() => expect(server.tests.length).toBe(2));
    await act(async () => {
      server.tests[1].resolve(
        new Msg(true, '', {
          ok: false,
          latencyMs: 0,
          exitIp: '',
          exitLoc: '',
          reason: 'no exit relays reachable in this country',
          error: '',
        } as never),
      );
    });
    await waitFor(() =>
      expect(row('aq').textContent).toContain('no exit relays reachable in this country'),
    );
    expect(row('aq').textContent).not.toContain('Works');
  });

  it('shows the server error when a test request itself is refused', async () => {
    const server = fakeServer();
    await openModal();

    fireEvent.click(within(row('de')).getByRole('button', { name: 'Test' }));
    await waitFor(() => expect(server.tests.length).toBe(1));
    await act(async () => {
      server.tests[0].resolve(
        new Msg(false, 'Test a Tor country: another country is being tested', null),
      );
    });
    await waitFor(() => expect(row('de').textContent).toContain('another country is being tested'));
  });

  it('adds a country and then offers to remove it', async () => {
    const server = fakeServer();
    await openModal();

    await act(async () => {
      fireEvent.click(within(row('nl')).getByRole('button', { name: 'Add' }));
    });
    expect(server.posts.find((p) => p.url === '/panel/api/tor/add')?.body).toEqual({
      country: 'nl',
    });
    await waitFor(() =>
      expect(within(row('nl')).getByRole('button', { name: 'Remove' })).toBeTruthy(),
    );

    await act(async () => {
      fireEvent.click(within(row('nl')).getByRole('button', { name: 'Remove' }));
    });
    expect(server.posts.some((p) => p.url === '/panel/api/tor/del/1')).toBe(true);
    await waitFor(() =>
      expect(within(row('nl')).getByRole('button', { name: 'Add' })).toBeTruthy(),
    );
  });

  it('shows the live status of an added country', async () => {
    fakeServer({
      exits: [
        {
          ...exit(1, 'de', 'failed'),
          status: {
            ...exit(1, 'de').status,
            state: 'failed',
            error: 'socks connect: general failure',
          },
        },
      ],
    });
    await openModal();

    await waitFor(() => expect(row('de').textContent).toContain('Not connected'));
    expect(row('de').textContent).toContain('socks connect: general failure');
  });

  it('shows the cap the server enforces when another country is added', async () => {
    fakeServer({ addError: 'Add a Tor country: at most 10 exits can be enabled' });
    await openModal();

    await act(async () => {
      fireEvent.click(within(row('fr')).getByRole('button', { name: 'Add' }));
    });
    await waitFor(() => expect(screen.getByText(/at most 10 exits can be enabled/)).toBeTruthy());
    expect(within(row('fr')).queryByRole('button', { name: 'Remove' })).toBeNull();
  });

  it('offers to install the engine when it is missing, and installs it', async () => {
    const server = fakeServer({ installed: false });
    await openModal();

    await waitFor(() => expect(screen.getByText('Not installed')).toBeTruthy());
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: /Install Tor 16\.0\.1/ }));
    });
    expect(server.posts.some((p) => p.url === '/panel/api/tor/engine/install')).toBe(true);
    await waitFor(() => expect(screen.getByText('Installed')).toBeTruthy());
  });

  it('keeps bridges under Advanced, saves them and shows the server refusal', async () => {
    const server = fakeServer({ bridges: 'obfs4 192.0.2.4:443 AAAA cert=x iat-mode=0' });
    await openModal();

    expect(document.querySelector('textarea')).toBeNull();
    fireEvent.click(screen.getByText('Advanced'));
    const box = (await waitFor(() => {
      const el = document.querySelector('textarea');
      if (!el) throw new Error('bridges box not shown');
      return el;
    })) as HTMLTextAreaElement;
    await waitFor(() => expect(box.value).toContain('obfs4 192.0.2.4:443'));

    fireEvent.change(box, { target: { value: 'snowflake 192.0.2.3:80 BBBB' } });
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Save bridges' }));
    });
    expect(server.posts.find((p) => p.url === '/panel/api/tor/bridges')?.body).toEqual({
      bridges: 'snowflake 192.0.2.3:80 BBBB',
    });

    server.bridgesError = 'Tor bridges: bridge line 2 is not a bridge';
    fireEvent.change(box, {
      target: { value: 'snowflake 192.0.2.3:80 BBBB\nExitPolicy accept *:*' },
    });
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Save bridges' }));
    });
    await waitFor(() => expect(screen.getByText(/bridge line 2/)).toBeTruthy());
  });

  it('warns that Tor carries TCP only and that exits can read unencrypted traffic', async () => {
    fakeServer();
    await openModal();

    expect(screen.getByText(/TCP traffic only/)).toBeTruthy();
    expect(screen.getByText(/not encrypted/)).toBeTruthy();
  });
});
