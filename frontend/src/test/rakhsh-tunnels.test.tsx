import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import TunnelsPage from '@/pages/tunnels/TunnelsPage';
import { HttpUtil, Msg } from '@/utils';
import { renderWithProviders } from './test-utils';

vi.mock('@/layouts/AppSidebar', () => ({ default: () => null }));

const JSON_HEADERS = { headers: { 'Content-Type': 'application/json' } };

const option = (key: string, extra: Record<string, unknown>) => ({
  key,
  kind: 'string',
  role: 'both',
  mode: 'both',
  group: 'tunnel',
  choices: null,
  min: 0,
  max: 0,
  hintEn: `about ${key}`,
  hintFa: '',
  ...extra,
});
const rakhshCatalog = [
  option('lanes', { kind: 'int', min: 1, max: 64 }),
  option('sni', { group: 'carrier' }),
  option('ws_path', { group: 'carrier' }),
  option('links', { kind: 'links', group: 'links' }),
  option('peer_ip', { group: 'raw' }),
  option('l3_addr', { group: 'layer3' }),
  option('limit', { group: 'limits', role: 'portal' }),
];
const backpackCatalog = [
  option('sni', { group: 'carrier' }),
  option('keepalive_period', { kind: 'int', mode: 'reverse', group: 'performance', min: 1 }),
];

const rakhshPortal = {
  id: 5,
  name: 'rk1',
  role: 'portal',
  engine: 'rakhsh',
  enable: true,
  address: '203.0.113.10',
  port: 7000,
  uuid: '8d6f3a0e-5c1b-4f0e-9a51-2f6d1c7b9e10',
  security: 'none',
  serverName: '',
  realityPublicKey: '',
  rakhshPeerPublic: '',
  shortId: '',
  routeInbounds: null,
  exitOutbound: '',
  transport: 'wss',
  fallbackTransports: null,
  forwardPorts: ['443=127.0.0.1:2096'],
  acceptUdp: false,
  mode: 'reverse',
  options: {
    sni: 'www.example.com',
    links: [
      { carrier: 'wss', addr: '203.0.113.10:7000' },
      { carrier: 'quic', addr: '203.0.113.10:7443', sni: 'cdn.example.com' },
    ],
  },
};
const rakhshBridge = {
  ...rakhshPortal,
  id: 6,
  name: 'rk2',
  role: 'bridge',
  options: { sni: 'www.example.com' },
};

let listed: Record<string, unknown>[];
let rakhshInstalled: boolean;
let rakhshStatus: Record<string, unknown>;

function openTunnels() {
  return renderWithProviders(
    <MemoryRouter>
      <TunnelsPage />
    </MemoryRouter>,
  );
}

async function startRakhshTunnel(name: string) {
  fireEvent.click(await screen.findByRole('button', { name: 'New tunnel' }));
  fireEvent.change(await screen.findByLabelText('Name'), { target: { value: name } });
  fireEvent.change(screen.getByLabelText('Address'), { target: { value: '198.51.100.7' } });
  fireEvent.click(screen.getByRole('radio', { name: 'Rakhsh' }));
  await screen.findByLabelText('Forwarded ports');
}

beforeEach(() => {
  listed = [];
  rakhshInstalled = true;
  rakhshStatus = {
    connected: true,
    latencyMs: 49,
    exitIp: '',
    exitLoc: '',
    error: '',
    lanesUp: 3,
    targetLanes: 4,
    lossPct: 0.4,
  };
  vi.spyOn(HttpUtil, 'get').mockImplementation(async (url, params) => {
    const engine = (params as { engine?: string } | undefined)?.engine ?? 'backpack';
    if (url === '/panel/api/tunnels/list') return new Msg(true, '', listed);
    if (url === '/panel/api/tunnels/engine')
      return new Msg(true, '', {
        engine,
        installed: engine === 'rakhsh' ? rakhshInstalled : true,
        version: engine === 'rakhsh' ? (rakhshInstalled ? 'v0.1.0' : '') : 'v1.9.0',
        pinned: engine === 'rakhsh' ? 'v0.1.0' : 'v1.9.0',
      });
    if (url === '/panel/api/tunnels/options')
      return new Msg(true, '', engine === 'rakhsh' ? rakhshCatalog : backpackCatalog);
    if (url === '/panel/api/tunnels/status/5') return new Msg(true, '', rakhshStatus);
    return new Msg(true, '', []);
  });
  vi.spyOn(HttpUtil, 'post').mockImplementation(async () => new Msg(true, '', {}));
});

afterEach(() => {
  vi.restoreAllMocks();
});

test('choosing Rakhsh loads Rakhsh options and drops the options of the engine it replaces', async () => {
  openTunnels();
  fireEvent.click(await screen.findByRole('button', { name: 'New tunnel' }));
  fireEvent.change(await screen.findByLabelText('Name'), { target: { value: 'rk3' } });
  fireEvent.change(screen.getByLabelText('Address'), { target: { value: '198.51.100.7' } });

  fireEvent.click(screen.getByRole('radio', { name: 'BackPack' }));
  fireEvent.click(await screen.findByText('Advanced options'));
  fireEvent.click(await screen.findByText('Performance'));
  fireEvent.change(await screen.findByLabelText('keepalive_period'), { target: { value: '30' } });
  fireEvent.click(await screen.findByText('Carrier'));
  fireEvent.change(await screen.findByLabelText('sni'), {
    target: { value: 'typed-for-backpack' },
  });

  fireEvent.click(screen.getByRole('radio', { name: 'Rakhsh' }));
  expect(await screen.findByText('l3_addr')).toBeTruthy();
  expect(HttpUtil.get).toHaveBeenCalledWith(
    '/panel/api/tunnels/options',
    { engine: 'rakhsh' },
    expect.anything(),
  );
  expect(screen.queryByLabelText('keepalive_period')).toBeNull();

  fireEvent.click(screen.getByRole('button', { name: 'OK' }));
  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/add',
      expect.objectContaining({ name: 'rk3', engine: 'rakhsh', options: {} }),
      JSON_HEADERS,
    ),
  );
});

test('creates a Rakhsh tunnel with its carrier, forwarded ports and the options that were set', async () => {
  openTunnels();
  await startRakhshTunnel('rk3');

  fireEvent.change(screen.getByLabelText('Forwarded ports'), {
    target: { value: '443=127.0.0.1:2096' },
  });
  fireEvent.keyDown(screen.getByLabelText('Forwarded ports'), { key: 'Enter', keyCode: 13 });
  fireEvent.click(await screen.findByText('Advanced options'));
  fireEvent.click(await screen.findByText('Carrier'));
  fireEvent.change(await screen.findByLabelText('sni'), { target: { value: 'www.example.com' } });
  fireEvent.click(screen.getByRole('button', { name: 'OK' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/add',
      {
        name: 'rk3',
        address: '198.51.100.7',
        port: 8443,
        engine: 'rakhsh',
        transport: 'tcp',
        forwardPorts: ['443=127.0.0.1:2096'],
        routeInbounds: [],
        mode: 'reverse',
        options: { sni: 'www.example.com' },
      },
      JSON_HEADERS,
    ),
  );
});

test('a raw carrier warns that it needs root and refuses to be created without peer_ip', async () => {
  openTunnels();
  await startRakhshTunnel('rk4');

  expect(screen.queryByText(/forges packets/)).toBeNull();
  fireEvent.mouseDown(screen.getByLabelText('Transport'));
  fireEvent.click((await screen.findAllByText('faketcp')).at(-1) as HTMLElement);

  expect(await screen.findByText(/forges packets/)).toBeTruthy();
  fireEvent.click(screen.getByRole('button', { name: 'OK' }));
  expect(await screen.findByText('This setting is required.')).toBeTruthy();
  expect(HttpUtil.post).not.toHaveBeenCalled();

  fireEvent.change(screen.getByLabelText('peer_ip'), { target: { value: '203.0.113.9' } });
  fireEvent.click(screen.getByRole('button', { name: 'OK' }));
  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/add',
      expect.objectContaining({
        engine: 'rakhsh',
        transport: 'faketcp',
        options: { peer_ip: '203.0.113.9' },
      }),
      JSON_HEADERS,
    ),
  );
});

test('adding two links submits them in order inside options.links', async () => {
  openTunnels();
  await startRakhshTunnel('rk5');

  fireEvent.click(await screen.findByText('Advanced options'));
  fireEvent.click(await screen.findByText('Links'));
  fireEvent.click(screen.getByRole('button', { name: 'Add link' }));
  fireEvent.click(screen.getByRole('button', { name: 'Add link' }));
  fireEvent.mouseDown(await screen.findByLabelText('Link 1 carrier'));
  fireEvent.click((await screen.findAllByText('wss')).at(-1) as HTMLElement);
  fireEvent.change(screen.getByLabelText('Link 1 address'), {
    target: { value: '198.51.100.7:7000' },
  });
  fireEvent.mouseDown(screen.getByLabelText('Link 2 carrier'));
  fireEvent.click((await screen.findAllByText('quic')).at(-1) as HTMLElement);
  fireEvent.change(screen.getByLabelText('Link 2 address'), {
    target: { value: '198.51.100.7:7443' },
  });
  fireEvent.click(screen.getByRole('button', { name: 'OK' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/add',
      expect.objectContaining({
        options: {
          links: [
            { carrier: 'wss', addr: '198.51.100.7:7000' },
            { carrier: 'quic', addr: '198.51.100.7:7443' },
          ],
        },
      }),
      JSON_HEADERS,
    ),
  );
});

test('removing a link leaves the others in place', async () => {
  listed = [rakhshPortal];
  openTunnels();

  fireEvent.click(await screen.findByRole('button', { name: 'Edit rk1' }));
  fireEvent.click(await screen.findByText('Advanced options'));
  fireEvent.click(await screen.findByText('Links'));
  fireEvent.click(await screen.findByRole('button', { name: 'Remove link 1' }));
  fireEvent.click(screen.getByRole('button', { name: 'OK' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/update/5',
      expect.objectContaining({
        options: {
          sni: 'www.example.com',
          links: [{ carrier: 'quic', addr: '203.0.113.10:7443', sni: 'cdn.example.com' }],
        },
      }),
      JSON_HEADERS,
    ),
  );
});

test('the edit dialog shows the stored links and submits a change to one of them', async () => {
  listed = [rakhshPortal];
  openTunnels();

  fireEvent.click(await screen.findByRole('button', { name: 'Edit rk1' }));
  const second = (await screen.findByLabelText('Link 2 address')) as HTMLInputElement;
  expect(second.value).toBe('203.0.113.10:7443');
  expect((screen.getByLabelText('Link 1 address') as HTMLInputElement).value).toBe(
    '203.0.113.10:7000',
  );
  expect((screen.getByLabelText('Link 2 sni') as HTMLInputElement).value).toBe('cdn.example.com');
  fireEvent.change(second, { target: { value: '203.0.113.10:7444' } });
  fireEvent.click(screen.getByRole('button', { name: 'OK' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/update/5',
      {
        address: '203.0.113.10',
        port: 7000,
        routeInbounds: [],
        transport: 'wss',
        forwardPorts: ['443=127.0.0.1:2096'],
        options: {
          sni: 'www.example.com',
          links: [
            { carrier: 'wss', addr: '203.0.113.10:7000' },
            { carrier: 'quic', addr: '203.0.113.10:7444', sni: 'cdn.example.com' },
          ],
        },
      },
      JSON_HEADERS,
    ),
  );
});

test('the exit end of a Rakhsh tunnel edits only its exit outbound and its own options', async () => {
  listed = [rakhshBridge];
  openTunnels();

  fireEvent.click(await screen.findByRole('button', { name: 'Edit rk2' }));
  expect(screen.queryByLabelText('Address')).toBeNull();
  fireEvent.click(await screen.findByText('Advanced options'));
  expect(screen.queryByText('limit')).toBeNull();
  expect(screen.getByText('sni')).toBeTruthy();
  fireEvent.change(await screen.findByLabelText('Exit outbound'), { target: { value: 'warp' } });
  fireEvent.click(screen.getByRole('button', { name: 'OK' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/update/6',
      { exitOutbound: 'warp', options: { sni: 'www.example.com' } },
      JSON_HEADERS,
    ),
  );
});

test('a Rakhsh row shows its engine, mode, lanes and an Edit button', async () => {
  listed = [rakhshPortal, { ...rakhshPortal, id: 7, name: 'rk9', mode: 'direct', enable: false }];
  openTunnels();

  expect(await screen.findByText('rk1')).toBeTruthy();
  expect(screen.getAllByText('Rakhsh').length).toBeGreaterThan(0);
  expect(screen.getByText('Reverse')).toBeTruthy();
  expect(screen.getByText('Direct')).toBeTruthy();
  expect(screen.getByRole('button', { name: 'Edit rk1' })).toBeTruthy();
  expect(await screen.findByText(/3 of 4 lanes/)).toBeTruthy();
  expect(screen.getByText(/49 ms/)).toBeTruthy();
});

test('a Rakhsh tunnel with no lane up shows the engine reason', async () => {
  listed = [rakhshPortal];
  rakhshStatus = {
    ...rakhshStatus,
    connected: false,
    lanesUp: 0,
    error: 'no lane is up (0 of 4)',
  };
  openTunnels();

  const tag = await screen.findByText('Down');
  expect(tag.closest('[title]')?.getAttribute('title')).toBe('no lane is up (0 of 4)');
});

test('offers to install the Rakhsh engine, not BackPack, when only Rakhsh tunnels exist', async () => {
  listed = [rakhshPortal];
  rakhshInstalled = false;
  openTunnels();

  expect(screen.queryByRole('button', { name: 'Install BackPack' })).toBeNull();
  fireEvent.click(await screen.findByRole('button', { name: 'Install Rakhsh' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/engine/install',
      { engine: 'rakhsh' },
      JSON_HEADERS,
    ),
  );
});

test('updates the installed Rakhsh engine and says which release it is', async () => {
  listed = [rakhshPortal];
  openTunnels();

  const version = await screen.findByText('Rakhsh engine v0.1.0');
  const bar = version.closest('.ant-space') as HTMLElement;
  fireEvent.click(within(bar).getByRole('button', { name: 'Update engine' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/engine/update',
      { nodeId: 0, engine: 'rakhsh', version: '' },
      JSON_HEADERS,
    ),
  );
});
