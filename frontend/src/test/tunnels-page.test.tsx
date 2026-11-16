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
