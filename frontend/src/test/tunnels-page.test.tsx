import { fireEvent, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import TunnelsPage from '@/pages/tunnels/TunnelsPage';
import { ClipboardManager, HttpUtil, Msg } from '@/utils';
import { renderWithProviders } from './test-utils';

vi.mock('@/layouts/AppSidebar', () => ({ default: () => null }));

const portal = {
  id: 1,
  name: 'de1',
  role: 'portal',
  engine: 'xray',
  enable: true,
  address: '203.0.113.10',
  port: 8443,
  uuid: '8d6f3a0e-5c1b-4f0e-9a51-2f6d1c7b9e10',
  security: 'reality',
  serverName: 'www.example.com',
  realityPublicKey: 'pub',
  shortId: 'a1b2c3d4',
  routeInbounds: ['inbound-443'],
  exitOutbound: '',
  transport: '',
  fallbackTransports: null,
  forwardPorts: null,
  acceptUdp: false,
  mode: 'reverse',
  options: null,
};
const backpackPortal = {
  ...portal,
  id: 3,
  name: 'bp1',
  engine: 'backpack',
  security: 'none',
  transport: 'wssmux',
  fallbackTransports: ['quic'],
  forwardPorts: ['443=127.0.0.1:2096'],
  routeInbounds: null,
};
const option = (key: string, extra: Record<string, unknown>) => ({
  key,
  kind: 'string',
  role: 'both',
  mode: 'reverse',
  group: 'performance',
  choices: null,
  min: 0,
  max: 0,
  hintEn: `about ${key}`,
  hintFa: '',
  ...extra,
});
const optionCatalog = [
  option('nodelay', { kind: 'bool' }),
  option('keepalive_period', { kind: 'int', min: 1, max: 600 }),
  option('carrier', { kind: 'choice', mode: 'direct', group: 'carrier', choices: ['udp', 'pck'] }),
  option('local_ip', { mode: 'direct', group: 'carrier' }),
  option('mtu', { kind: 'int', mode: 'direct', group: 'quality', min: 576, max: 9000 }),
];
let listed: Record<string, unknown>[];
let engineInstalled: boolean;
const bridge = { ...portal, id: 2, name: 'ir1', role: 'bridge', routeInbounds: null };
let statusOfPortal: Record<string, unknown>;

function openTunnels() {
  return renderWithProviders(
    <MemoryRouter>
      <TunnelsPage />
    </MemoryRouter>,
  );
}

beforeEach(() => {
  listed = [portal, bridge];
  engineInstalled = true;
  statusOfPortal = {
    connected: true,
    latencyMs: 182,
    exitIp: '203.0.113.77',
    exitLoc: 'DE',
    error: '',
  };
  vi.spyOn(HttpUtil, 'get').mockImplementation(async (url) => {
    if (url === '/panel/api/tunnels/list') return new Msg(true, '', listed);
    if (url === '/panel/api/tunnels/engine')
      return new Msg(true, '', {
        installed: engineInstalled,
        version: engineInstalled ? 'v1.8.5' : '',
        pinned: 'v1.8.5',
      });
    if (url === '/panel/api/tunnels/status/3')
      return new Msg(true, '', {
        connected: true,
        latencyMs: 0,
        exitIp: '198.51.100.7',
        exitLoc: '',
        error: '',
      });
    if (url === '/panel/api/tunnels/options') return new Msg(true, '', optionCatalog);
    if (url === '/panel/api/tunnels/link/1') return new Msg(true, '', 'dalan://tunnel/abc');
    if (url === '/panel/api/tunnels/status/1') return new Msg(true, '', statusOfPortal);
    return new Msg(true, '', []);
  });
  vi.spyOn(HttpUtil, 'post').mockImplementation(async () => new Msg(true, '', {}));
});

afterEach(() => {
  vi.restoreAllMocks();
});

test('lists both ends with their role and offers the setup link only for the entry end', async () => {
  vi.spyOn(ClipboardManager, 'copyText').mockResolvedValue(true);
  openTunnels();

  expect(await screen.findByText('de1')).toBeTruthy();
  expect(screen.getByText('ir1')).toBeTruthy();
  expect(screen.getByText('Entry (listens)')).toBeTruthy();
  expect(screen.getByText('Exit (dials)')).toBeTruthy();

  const copyButtons = screen.getAllByRole('button', { name: 'Copy setup link' });
  expect(copyButtons).toHaveLength(1);
  fireEvent.click(copyButtons[0]);

  await waitFor(() => expect(ClipboardManager.copyText).toHaveBeenCalledWith('dalan://tunnel/abc'));
});

test('imports a pasted setup link', async () => {
  openTunnels();
  await screen.findByText('de1');

  fireEvent.click(screen.getByRole('button', { name: 'Import link' }));
  fireEvent.change(
    await screen.findByPlaceholderText('Paste the setup link from the entry server'),
    { target: { value: '  dalan://tunnel/xyz ' } },
  );
  fireEvent.click(screen.getByRole('button', { name: 'OK' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/import',
      { link: 'dalan://tunnel/xyz', exitOutbound: '' },
      { headers: { 'Content-Type': 'application/json' } },
    ),
  );
});

test('creates an entry tunnel from the form', async () => {
  openTunnels();
  await screen.findByText('de1');

  fireEvent.click(screen.getByRole('button', { name: 'New tunnel' }));
  fireEvent.change(await screen.findByLabelText('Name'), { target: { value: 'tr1' } });
  fireEvent.change(screen.getByLabelText('Address'), { target: { value: '198.51.100.7' } });
  fireEvent.change(screen.getByLabelText('Port'), { target: { value: '9443' } });
  fireEvent.change(screen.getByLabelText('Server name (SNI)'), {
    target: { value: 'www.example.org' },
  });
  fireEvent.click(screen.getByRole('button', { name: 'OK' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/add',
      {
        name: 'tr1',
        address: '198.51.100.7',
        port: 9443,
        engine: 'xray',
        security: 'reality',
        serverName: 'www.example.org',
        routeInbounds: [],
      },
      { headers: { 'Content-Type': 'application/json' } },
    ),
  );
});

test('shows a working entry tunnel as connected with its latency and exit', async () => {
  openTunnels();

  expect(await screen.findByText('Connected · 182 ms')).toBeTruthy();
  expect(screen.getByText('Exits from DE · 203.0.113.77')).toBeTruthy();
  expect(screen.getByText('Tested from the entry server')).toBeTruthy();
  expect(HttpUtil.get).not.toHaveBeenCalledWith(
    '/panel/api/tunnels/status/2',
    expect.anything(),
    expect.anything(),
  );
});

test('shows a tunnel that does not answer as down', async () => {
  statusOfPortal = { connected: false, latencyMs: 0, exitIp: '', exitLoc: '', error: 'timeout' };
  openTunnels();

  expect(await screen.findByText('Down')).toBeTruthy();
  expect(screen.queryByText(/Connected/)).toBeNull();
});

test('shows a BackPack tunnel with its transport, forwarded ports and engine status', async () => {
  listed = [backpackPortal];
  openTunnels();

  expect(await screen.findByText('bp1')).toBeTruthy();
  expect(screen.getByText('BackPack')).toBeTruthy();
  expect(screen.getByText('wssmux')).toBeTruthy();
  expect(screen.getByText('443=127.0.0.1:2096')).toBeTruthy();
  expect(await screen.findByText('Connected')).toBeTruthy();
  expect(screen.getByText('Exits from 198.51.100.7')).toBeTruthy();
});

test('offers to install the BackPack engine when a BackPack tunnel has none', async () => {
  listed = [backpackPortal];
  engineInstalled = false;
  openTunnels();

  fireEvent.click(await screen.findByRole('button', { name: 'Install BackPack' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith('/panel/api/tunnels/engine/install'),
  );
});

test('shows the installed BackPack engine release and updates it from this server', async () => {
  listed = [backpackPortal];
  openTunnels();
  expect(await screen.findByText('BackPack engine v1.8.5')).toBeTruthy();

  fireEvent.click(screen.getByRole('button', { name: 'Update engine' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/engine/update',
      { nodeId: 0, version: '' },
      { headers: { 'Content-Type': 'application/json' } },
    ),
  );
});

test('installs the BackPack engine through a node when this server cannot reach GitHub', async () => {
  listed = [backpackPortal];
  engineInstalled = false;
  vi.spyOn(HttpUtil, 'get').mockImplementation(async (url) => {
    if (url === '/panel/api/nodes/list')
      return new Msg(true, '', [{ id: 9, name: 'nl1', enable: true }]);
    if (url === '/panel/api/tunnels/list') return new Msg(true, '', listed);
    if (url === '/panel/api/tunnels/engine')
      return new Msg(true, '', { installed: false, version: '', pinned: 'v1.8.5' });
    return new Msg(true, '', []);
  });
  openTunnels();

  fireEvent.mouseDown(await screen.findByLabelText('Download through node'));
  fireEvent.click(await screen.findByText('nl1'));
  fireEvent.click(screen.getByRole('button', { name: 'Install BackPack' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/engine/update',
      { nodeId: 9, version: '' },
      { headers: { 'Content-Type': 'application/json' } },
    ),
  );
});

test('does not mention the BackPack engine while only Xray tunnels exist', async () => {
  engineInstalled = false;
  openTunnels();
  await screen.findByText('de1');

  expect(screen.queryByRole('button', { name: 'Install BackPack' })).toBeNull();
});

test('creates a BackPack entry tunnel with its transport and forwarded ports', async () => {
  openTunnels();
  await screen.findByText('de1');

  fireEvent.click(screen.getByRole('button', { name: 'New tunnel' }));
  fireEvent.change(await screen.findByLabelText('Name'), { target: { value: 'bp2' } });
  fireEvent.change(screen.getByLabelText('Address'), { target: { value: '198.51.100.7' } });
  fireEvent.click(screen.getByRole('radio', { name: 'BackPack' }));
  const ports = await screen.findByLabelText('Forwarded ports');
  fireEvent.change(ports, { target: { value: '443=127.0.0.1:2096' } });
  fireEvent.keyDown(ports, { key: 'Enter', keyCode: 13 });
  fireEvent.click(screen.getByRole('button', { name: 'OK' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/add',
      {
        name: 'bp2',
        address: '198.51.100.7',
        port: 8443,
        engine: 'backpack',
        transport: 'tcpmux',
        fallbackTransports: [],
        forwardPorts: ['443=127.0.0.1:2096'],
        acceptUdp: false,
        routeInbounds: [],
        mode: 'reverse',
        options: {},
      },
      { headers: { 'Content-Type': 'application/json' } },
    ),
  );
});

test('creates a BackPack entry tunnel without any forwarded port', async () => {
  openTunnels();
  await screen.findByText('de1');

  fireEvent.click(screen.getByRole('button', { name: 'New tunnel' }));
  fireEvent.change(await screen.findByLabelText('Name'), { target: { value: 'bp3' } });
  fireEvent.change(screen.getByLabelText('Address'), { target: { value: '198.51.100.7' } });
  fireEvent.click(screen.getByRole('radio', { name: 'BackPack' }));
  await screen.findByLabelText('Forwarded ports');
  fireEvent.click(screen.getByRole('button', { name: 'OK' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/add',
      expect.objectContaining({ name: 'bp3', engine: 'backpack', forwardPorts: [] }),
      { headers: { 'Content-Type': 'application/json' } },
    ),
  );
});

test('makes the other end on a chosen node from the same form', async () => {
  vi.spyOn(HttpUtil, 'get').mockImplementation(async (url) => {
    if (url === '/panel/api/nodes/list')
      return new Msg(true, '', [{ id: 9, name: 'nl1', enable: true }]);
    if (url === '/panel/api/tunnels/list') return new Msg(true, '', listed);
    return new Msg(true, '', []);
  });
  openTunnels();
  await screen.findByText('de1');

  fireEvent.click(screen.getByRole('button', { name: 'New tunnel' }));
  fireEvent.change(await screen.findByLabelText('Name'), { target: { value: 'tr2' } });
  fireEvent.change(screen.getByLabelText('Address'), { target: { value: '198.51.100.7' } });
  fireEvent.change(screen.getByLabelText('Server name (SNI)'), {
    target: { value: 'www.example.org' },
  });
  fireEvent.mouseDown(await screen.findByLabelText('Create the other end on a node'));
  fireEvent.click(await screen.findByText('nl1'));
  fireEvent.click(screen.getByRole('button', { name: 'OK' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/add',
      expect.objectContaining({ name: 'tr2', nodeId: 9 }),
      { headers: { 'Content-Type': 'application/json' } },
    ),
  );
});

test('offers to remove only this end when the node holding the other end cannot be reached', async () => {
  listed = [{ ...portal, id: 5, name: 'tr5', nodeId: 9 }];
  vi.spyOn(HttpUtil, 'post').mockImplementation(async (url) =>
    url === '/panel/api/tunnels/del/5'
      ? new Msg(false, 'node nl1 is unreachable', null)
      : new Msg(true, '', {}),
  );
  openTunnels();
  await screen.findByText('tr5');

  fireEvent.click(screen.getByRole('button', { name: 'Delete tr5' }));
  fireEvent.click(await screen.findByRole('button', { name: 'OK' }));
  fireEvent.click(await screen.findByRole('button', { name: 'Remove this end only' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith('/panel/api/tunnels/del/5?here=true'),
  );
});

test('tests which ports the chosen node can reach and fills in the one that is clicked', async () => {
  vi.spyOn(HttpUtil, 'get').mockImplementation(async (url) => {
    if (url === '/panel/api/nodes/list')
      return new Msg(true, '', [{ id: 9, name: 'nl1', enable: true }]);
    if (url === '/panel/api/tunnels/list') return new Msg(true, '', listed);
    return new Msg(true, '', []);
  });
  vi.spyOn(HttpUtil, 'post').mockImplementation(async (url) =>
    url === '/panel/api/tunnels/routetest'
      ? new Msg(true, '', [
          { port: 443, reachable: false, latencyMs: 0, error: 'timeout' },
          { port: 8443, reachable: true, latencyMs: 96, error: '' },
        ])
      : new Msg(true, '', {}),
  );
  openTunnels();
  await screen.findByText('de1');

  fireEvent.click(screen.getByRole('button', { name: 'New tunnel' }));
  fireEvent.change(await screen.findByLabelText('Name'), { target: { value: 'tr3' } });
  fireEvent.change(screen.getByLabelText('Address'), { target: { value: '198.51.100.7' } });
  fireEvent.mouseDown(await screen.findByLabelText('Create the other end on a node'));
  fireEvent.click(await screen.findByText('nl1'));
  fireEvent.click(await screen.findByRole('button', { name: 'Test ports' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/routetest',
      expect.objectContaining({ nodeId: 9, address: '198.51.100.7', ports: expect.any(Array) }),
      { headers: { 'Content-Type': 'application/json' } },
    ),
  );
  expect(await screen.findByText('443: timeout')).toBeTruthy();
  fireEvent.click(await screen.findByRole('button', { name: '8443 · 96 ms' }));
  expect((screen.getByLabelText('Port') as HTMLInputElement).value).toBe('8443');
});

test('shows how much has passed through an entry tunnel', async () => {
  listed = [{ ...portal, up: 1536, down: 5 * 1024 * 1024 }, bridge];
  openTunnels();

  expect(await screen.findByText('↑ 1.50 KB · ↓ 5.00 MB')).toBeTruthy();
});

const directPortal = {
  ...backpackPortal,
  id: 5,
  name: 'gm1',
  mode: 'direct',
  transport: '',
  fallbackTransports: null,
  forwardPorts: ['27015'],
  options: { carrier: 'udp', local_ip: '10.10.7.1/30', peer_ip: '10.10.7.2', mtu: 1400 },
};

test('says on every tunnel whether it is reverse or direct, Xray ones included', async () => {
  listed = [portal, directPortal];
  openTunnels();

  expect(await screen.findByText('gm1')).toBeTruthy();
  expect(screen.getByText('Reverse')).toBeTruthy();
  expect(screen.getByText('Direct')).toBeTruthy();
});

test('shows no Xray traffic for a direct tunnel, which is not an Xray outbound', async () => {
  listed = [{ ...directPortal, up: 2048, down: 4096 }];
  openTunnels();

  await screen.findByText('gm1');
  expect(screen.queryByText(/↑/)).toBeNull();
});

test('creates a direct tunnel with a carrier, no transport and no routed inbounds', async () => {
  openTunnels();
  await screen.findByText('de1');

  fireEvent.click(screen.getByRole('button', { name: 'New tunnel' }));
  fireEvent.change(await screen.findByLabelText('Name'), { target: { value: 'gm2' } });
  fireEvent.change(screen.getByLabelText('Address'), { target: { value: '198.51.100.7' } });
  fireEvent.click(screen.getByRole('radio', { name: 'BackPack' }));
  fireEvent.click(await screen.findByRole('radio', { name: 'Direct' }));
  await waitFor(() => expect(screen.queryByLabelText('Transport')).toBeNull());
  expect(screen.queryByLabelText('Route inbounds')).toBeNull();

  const ports = await screen.findByLabelText('Forwarded ports');
  fireEvent.change(ports, { target: { value: '27015' } });
  fireEvent.keyDown(ports, { key: 'Enter', keyCode: 13 });
  fireEvent.mouseDown(await screen.findByLabelText('carrier'));
  fireEvent.click((await screen.findAllByText('pck')).at(-1) as HTMLElement);
  fireEvent.click(screen.getByRole('button', { name: 'OK' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/add',
      {
        name: 'gm2',
        address: '198.51.100.7',
        port: 8443,
        engine: 'backpack',
        transport: '',
        fallbackTransports: [],
        forwardPorts: ['27015'],
        acceptUdp: false,
        routeInbounds: [],
        mode: 'direct',
        options: { carrier: 'pck' },
      },
      { headers: { 'Content-Type': 'application/json' } },
    ),
  );
});

test('refuses to create a direct tunnel before a carrier is chosen', async () => {
  openTunnels();
  await screen.findByText('de1');

  fireEvent.click(screen.getByRole('button', { name: 'New tunnel' }));
  fireEvent.change(await screen.findByLabelText('Name'), { target: { value: 'gm3' } });
  fireEvent.change(screen.getByLabelText('Address'), { target: { value: '198.51.100.7' } });
  fireEvent.click(screen.getByRole('radio', { name: 'BackPack' }));
  fireEvent.click(await screen.findByRole('radio', { name: 'Direct' }));
  await screen.findByLabelText('carrier');
  fireEvent.click(screen.getByRole('button', { name: 'OK' }));

  await new Promise((resolve) => setTimeout(resolve, 100));
  expect(HttpUtil.post).not.toHaveBeenCalled();
});

test('sends the advanced options the operator set and leaves the rest to the engine', async () => {
  openTunnels();
  await screen.findByText('de1');

  fireEvent.click(screen.getByRole('button', { name: 'New tunnel' }));
  fireEvent.change(await screen.findByLabelText('Name'), { target: { value: 'bp4' } });
  fireEvent.change(screen.getByLabelText('Address'), { target: { value: '198.51.100.7' } });
  fireEvent.click(screen.getByRole('radio', { name: 'BackPack' }));
  fireEvent.click(await screen.findByText('Advanced options'));
  fireEvent.click(await screen.findByText('Performance'));
  fireEvent.mouseDown(await screen.findByLabelText('nodelay'));
  fireEvent.click(await screen.findByText('Off'));
  fireEvent.click(screen.getByRole('button', { name: 'OK' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/add',
      expect.objectContaining({ name: 'bp4', mode: 'reverse', options: { nodelay: false } }),
      { headers: { 'Content-Type': 'application/json' } },
    ),
  );
});

test('edits an entry tunnel in place from its stored values', async () => {
  listed = [{ ...backpackPortal, options: { nodelay: true } }];
  openTunnels();

  fireEvent.click(await screen.findByRole('button', { name: 'Edit bp1' }));
  const address = (await screen.findByLabelText('Address')) as HTMLInputElement;
  expect(address.value).toBe('203.0.113.10');
  fireEvent.change(address, { target: { value: '203.0.113.99' } });
  fireEvent.click(screen.getByRole('button', { name: 'OK' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/update/3',
      {
        address: '203.0.113.99',
        port: 8443,
        routeInbounds: [],
        transport: 'wssmux',
        fallbackTransports: ['quic'],
        forwardPorts: ['443=127.0.0.1:2096'],
        acceptUdp: false,
        options: { nodelay: true },
      },
      { headers: { 'Content-Type': 'application/json' } },
    ),
  );
});

test('edits only the exit outbound of an exit end of an Xray tunnel', async () => {
  openTunnels();

  fireEvent.click(await screen.findByRole('button', { name: 'Edit ir1' }));
  expect(screen.queryByLabelText('Address')).toBeNull();
  fireEvent.change(await screen.findByLabelText('Exit outbound'), { target: { value: 'warp' } });
  fireEvent.click(screen.getByRole('button', { name: 'OK' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/update/2',
      { exitOutbound: 'warp' },
      { headers: { 'Content-Type': 'application/json' } },
    ),
  );
});

test('keeps the edit dialog open and shows the node refusal when the update fails', async () => {
  listed = [{ ...backpackPortal, nodeId: 9 }];
  vi.spyOn(HttpUtil, 'post').mockImplementation(
    async () => new Msg(false, 'the node did not take the change', null),
  );
  openTunnels();

  fireEvent.click(await screen.findByRole('button', { name: 'Edit bp1' }));
  expect(await screen.findByText(/also sent to the node/)).toBeTruthy();
  fireEvent.click(screen.getByRole('button', { name: 'OK' }));

  expect(await screen.findByText('the node did not take the change')).toBeTruthy();
  expect(screen.getByLabelText('Address')).toBeTruthy();
});
