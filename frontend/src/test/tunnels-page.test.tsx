import { fireEvent, within, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import TunnelsPage from '@/pages/tunnels/TunnelsPage';
import { ClipboardManager, HttpUtil, Msg } from '@/utils';
import { renderWithProviders } from './test-utils';

const portal = {
  id: 1,
  name: 'de1',
  role: 'portal',
  engine: 'backpack',
  enable: true,
  address: '203.0.113.10',
  port: 8443,
  uuid: '8d6f3a0e-5c1b-4f0e-9a51-2f6d1c7b9e10',
  rakhshPeerPublic: '',
  routeInbounds: ['inbound-443'],
  exitOutbound: '',
  transport: 'wssmux',
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
  expect(screen.getByText(/Entry \(listens\)/)).toBeTruthy();
  expect(screen.getByText(/Exit \(dials\)/)).toBeTruthy();

  fireEvent.click(screen.getByRole('button', { name: 'Actions ir1' }));
  expect(screen.queryByRole('menuitem', { name: 'Copy setup link' })).toBeNull();
  fireEvent.click(screen.getByRole('button', { name: 'Actions de1' }));
  fireEvent.click(await screen.findByRole('menuitem', { name: 'Copy setup link' }));

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
  fireEvent.click(screen.getByRole('button', { name: 'Import' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/import',
      { link: 'dalan://tunnel/xyz', exitOutbound: '' },
      { headers: { 'Content-Type': 'application/json' } },
    ),
  );
});

test('offers only the BackPack and Rakhsh engines for a new tunnel', async () => {
  openTunnels();
  await screen.findByText('de1');

  fireEvent.click(screen.getByRole('button', { name: 'New tunnel' }));

  expect(await screen.findByLabelText('BackPack')).toBeTruthy();
  expect(screen.getByLabelText('Rakhsh')).toBeTruthy();
  expect(screen.queryByLabelText('Xray (built in)')).toBeNull();
  expect(screen.queryByLabelText('Server name (SNI)')).toBeNull();
});

test('shows a working tunnel as connected with its latency and exit', async () => {
  openTunnels();

  const table = (await screen.findAllByRole('table'))[0];
  expect(await within(table).findByText('Connected')).toBeTruthy();
  expect(within(table).getByText('182 ms')).toBeTruthy();
  expect(screen.getByText('Exits from DE · 203.0.113.77')).toBeTruthy();
});

test('shows a tunnel that does not answer as down', async () => {
  statusOfPortal = { connected: false, latencyMs: 0, exitIp: '', exitLoc: '', error: 'timeout' };
  openTunnels();

  const row = (await screen.findByText('de1')).closest('tr') as HTMLElement;
  expect(await within(row).findByText('Down')).toBeTruthy();
  expect(within(row).queryByText(/Connected/)).toBeNull();
});

test('shows a BackPack tunnel with its transport, forwarded ports and engine status', async () => {
  listed = [backpackPortal];
  openTunnels();

  expect(await screen.findByText('bp1')).toBeTruthy();
  expect(within(screen.getAllByRole('table')[0]).getByText('BackPack')).toBeTruthy();
  expect(screen.getByText('wssmux')).toBeTruthy();
  expect(screen.getByText('443=127.0.0.1:2096')).toBeTruthy();
  expect(await screen.findByText('Connected')).toBeTruthy();
  expect(screen.getByText('Exits from 198.51.100.7')).toBeTruthy();
});

test('offers to install the BackPack engine when a BackPack tunnel has none', async () => {
  listed = [backpackPortal];
  engineInstalled = false;
  openTunnels();

  fireEvent.click(await screen.findByRole('button', { name: 'BackPack options' }));
  fireEvent.click(await screen.findByRole('button', { name: 'Install BackPack' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/engine/install',
      { engine: 'backpack', route: { via: 'panel', value: '' } },
      { headers: { 'Content-Type': 'application/json' } },
    ),
  );
});

test('installs the BackPack engine through the outbound chosen as the download route', async () => {
  listed = [backpackPortal];
  engineInstalled = false;
  const base = vi.mocked(HttpUtil.get).getMockImplementation();
  vi.spyOn(HttpUtil, 'get').mockImplementation(async (url, ...rest) =>
    url === '/panel/api/tunnels/engine/routes'
      ? new Msg(true, '', [
          { via: 'panel', value: '', kind: 'panel' },
          { via: 'direct', value: '', kind: 'direct' },
          { via: 'outbound', value: 'tunnel-de1', kind: 'tunnel' },
        ])
      : base!(url, ...rest),
  );
  openTunnels();

  fireEvent.click(await screen.findByRole('button', { name: 'BackPack options' }));
  const picker = await screen.findByLabelText('Download route');
  await waitFor(() =>
    expect(HttpUtil.get).toHaveBeenCalledWith('/panel/api/tunnels/engine/routes', undefined, {
      silent: true,
    }),
  );
  fireEvent.mouseDown(picker);
  fireEvent.click(await screen.findByText('tunnel-de1'));
  fireEvent.click(screen.getByRole('button', { name: 'Install BackPack' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/engine/install',
      { engine: 'backpack', route: { via: 'outbound', value: 'tunnel-de1' } },
      { headers: { 'Content-Type': 'application/json' } },
    ),
  );
});

test('will not install through a link until an address is typed', async () => {
  listed = [backpackPortal];
  engineInstalled = false;
  openTunnels();

  fireEvent.click(await screen.findByRole('button', { name: 'BackPack options' }));
  fireEvent.mouseDown(await screen.findByLabelText('Download route'));
  fireEvent.click(await screen.findByText('A proxy address or share link'));
  const install = screen.getByRole('button', { name: 'Install BackPack' });
  expect(install.hasAttribute('disabled')).toBe(true);

  fireEvent.change(screen.getByRole('textbox', { name: 'Proxy address or share link' }), {
    target: { value: 'socks5://127.0.0.1:1080' },
  });
  expect(install.hasAttribute('disabled')).toBe(false);
  fireEvent.click(install);

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/engine/install',
      { engine: 'backpack', route: { via: 'link', value: 'socks5://127.0.0.1:1080' } },
      { headers: { 'Content-Type': 'application/json' } },
    ),
  );
});

test('shows the installed BackPack engine release and updates it from this server', async () => {
  listed = [backpackPortal];
  openTunnels();
  expect(await screen.findByText('BackPack engine v1.8.5')).toBeTruthy();

  fireEvent.click(screen.getByRole('button', { name: 'BackPack options' }));
  fireEvent.click(await screen.findByRole('button', { name: 'Update engine' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/engine/update',
      { nodeId: 0, engine: 'backpack', version: '', route: { via: 'panel', value: '' } },
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

  fireEvent.click(await screen.findByRole('button', { name: 'BackPack options' }));
  fireEvent.click(await screen.findByRole('button', { name: /Download on/ }));
  fireEvent.click(await screen.findByRole('option', { name: 'nl1' }));
  expect(screen.queryByLabelText('Download route')).toBeNull();
  fireEvent.click(screen.getByRole('button', { name: 'Install BackPack' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/engine/update',
      { nodeId: 9, engine: 'backpack', version: '', route: { via: 'panel', value: '' } },
      { headers: { 'Content-Type': 'application/json' } },
    ),
  );
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
  fireEvent.click(screen.getByRole('button', { name: 'Create tunnel' }));

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
  fireEvent.click(screen.getByRole('button', { name: 'Create tunnel' }));

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
  fireEvent.mouseDown(await screen.findByLabelText('Create the other end on a node'));
  fireEvent.click(await screen.findByText('nl1'));
  fireEvent.click(screen.getByRole('button', { name: 'Create tunnel' }));

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

  fireEvent.click(screen.getByRole('button', { name: 'Actions tr5' }));
  fireEvent.click(await screen.findByRole('menuitem', { name: 'Delete tr5' }));
  fireEvent.click(await screen.findByRole('button', { name: 'Delete' }));
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

test('says on every tunnel whether it is reverse or direct', async () => {
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
  fireEvent.click(screen.getByRole('button', { name: 'Create tunnel' }));

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
  fireEvent.click(screen.getByRole('button', { name: 'Create tunnel' }));

  expect(await screen.findByText('This setting is required.')).toBeTruthy();
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
  fireEvent.click(screen.getByRole('button', { name: 'Create tunnel' }));

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
  fireEvent.click(screen.getByRole('button', { name: 'Save' }));

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

test('edits the exit outbound of an exit end without asking for an address', async () => {
  openTunnels();

  fireEvent.click(await screen.findByRole('button', { name: 'Edit ir1' }));
  expect(screen.queryByLabelText('Address')).toBeNull();
  fireEvent.change(await screen.findByLabelText('Exit outbound'), { target: { value: 'warp' } });
  fireEvent.click(screen.getByRole('button', { name: 'Save' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/update/2',
      { exitOutbound: 'warp', options: {} },
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
  fireEvent.click(screen.getByRole('button', { name: 'Save' }));

  expect(await screen.findByText('the node did not take the change')).toBeTruthy();
  expect(screen.getByLabelText('Address')).toBeTruthy();
});

test('the state chips list only the tunnels in that state', async () => {
  listed = [portal, { ...portal, id: 4, name: 'nl2', enable: false }];
  openTunnels();
  const row = (await screen.findByText('de1')).closest('tr') as HTMLElement;
  await within(row).findByText('Connected');

  fireEvent.click(screen.getByRole('button', { name: /^Paused/ }));
  expect(screen.queryByText('de1')).toBeNull();
  expect(screen.getByText('nl2')).toBeTruthy();
  fireEvent.click(screen.getByRole('button', { name: /^Up/ }));
  expect(screen.getByText('de1')).toBeTruthy();
  expect(screen.queryByText('nl2')).toBeNull();
});

test('test all probes every tunnel again', async () => {
  openTunnels();
  const row = (await screen.findByText('de1')).closest('tr') as HTMLElement;
  await within(row).findByText('Connected');
  const probes = () =>
    vi.mocked(HttpUtil.get).mock.calls.filter(([url]) => url === '/panel/api/tunnels/status/1')
      .length;
  const before = probes();

  fireEvent.click(screen.getByRole('button', { name: 'Test all' }));
  await waitFor(() => expect(probes()).toBe(before + 1));
});

test('draws the last hour of probe latency for a tunnel', async () => {
  statusOfPortal = { ...statusOfPortal, history: [180, 0, 182] };
  openTunnels();
  const row = (await screen.findByText('de1')).closest('tr') as HTMLElement;
  await within(row).findByText('Connected');
  expect(row.querySelector('svg.spark')).not.toBeNull();
});
