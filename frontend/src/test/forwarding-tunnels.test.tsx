import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import TunnelsPage from '@/pages/tunnels/TunnelsPage';
import { HttpUtil, Msg } from '@/utils';
import { renderWithProviders } from './test-utils';

const JSON_HEADERS = { headers: { 'Content-Type': 'application/json' } };

const transport = (id: string, label: string, reverse: boolean, direct: boolean) => ({
  id,
  label,
  reverse,
  direct,
});
const catalog = [
  {
    id: 'gost',
    label: 'GOST',
    pinned: 'v3.3.3',
    transports: [transport('tls', 'TLS', true, true), transport('grpc', 'gRPC', true, true)],
    options: [],
  },
  {
    id: 'rathole',
    label: 'Rathole',
    pinned: 'v0.5.0',
    transports: [transport('tcp', 'TCP', true, false), transport('noise', 'Noise', true, false)],
    options: [],
  },
  {
    id: 'singbox',
    label: 'sing-box',
    pinned: 'v1.14.4',
    transports: [transport('wss', 'WebSocket over TLS', false, true)],
    options: [],
  },
  {
    id: 'kariz',
    label: 'Kariz',
    pinned: 'v2.8.1',
    transports: [transport('tcp', 'TCP', true, true), transport('kcp', 'KCP', true, false)],
    options: [
      { key: 'profile', label: 'Profile', choices: ['balanced', 'gaming'], default: 'balanced' },
    ],
  },
];

const carrierEngine = {
  id: 'carrier',
  label: 'Kernel carriers',
  pinned: 'built in',
  transports: [transport('gre', 'GRE', true, true), transport('vxlan', 'VXLAN', true, true)],
  options: [
    { key: 'dialerIp', label: 'Public IPv4 address', choices: [], default: '', kind: 'ip' },
  ],
};

const gostPortal = {
  id: 8,
  name: 'gs1',
  role: 'portal',
  engine: 'gost',
  enable: true,
  address: '203.0.113.10',
  port: 7000,
  uuid: '',
  routeInbounds: ['inbound-443'],
  exitOutbound: '',
  transport: 'tls',
  fallbackTransports: null,
  forwardPorts: ['443=127.0.0.1:2096'],
  acceptUdp: false,
  mode: 'reverse',
  options: null,
};
const gostBridge = { ...gostPortal, id: 9, name: 'gs2', role: 'bridge', routeInbounds: null };

let listed: Record<string, unknown>[];

function openTunnels() {
  return renderWithProviders(
    <MemoryRouter>
      <TunnelsPage />
    </MemoryRouter>,
  );
}

// The engine list arrives asynchronously, so the form is ready once the first transport shows.
async function startTunnel(name: string, engine: string, firstTransport: string) {
  // The empty state offers a second button once the list has loaded.
  fireEvent.click((await screen.findAllByRole('button', { name: 'New tunnel' }))[0]);
  fireEvent.change(await screen.findByLabelText('Name'), { target: { value: name } });
  fireEvent.change(screen.getByLabelText('Address'), { target: { value: '198.51.100.7' } });
  fireEvent.click(await screen.findByRole('radio', { name: engine }));
  await waitFor(() =>
    expect(screen.getByLabelText('Transport').closest('.ant-select')?.textContent).toBe(
      firstTransport,
    ),
  );
}

beforeEach(() => {
  listed = [];
  vi.spyOn(HttpUtil, 'get').mockImplementation(async (url) => {
    if (url === '/panel/api/tunnels/list') return new Msg(true, '', listed);
    if (url === '/panel/api/tunnels/engines') return new Msg(true, '', [...catalog, carrierEngine]);
    if (url === '/panel/api/tunnels/engine')
      return new Msg(true, '', { installed: true, version: 'v1.0.0', pinned: 'v1.0.0' });
    if (url === '/panel/api/tunnels/status/8')
      return new Msg(true, '', {
        connected: true,
        latencyMs: 31,
        exitIp: '',
        exitLoc: '',
        error: '',
      });
    return new Msg(true, '', []);
  });
  vi.spyOn(HttpUtil, 'post').mockImplementation(async () => new Msg(true, '', {}));
});

afterEach(() => {
  vi.restoreAllMocks();
});

test('creates a tunnel on a forwarding engine with the transport, ports and inbounds that were set', async () => {
  openTunnels();
  await startTunnel('gs3', 'GOST', 'TLS');

  fireEvent.change(screen.getByLabelText('Forwarded ports'), {
    target: { value: '443=127.0.0.1:2096' },
  });
  fireEvent.keyDown(screen.getByLabelText('Forwarded ports'), { key: 'Enter', keyCode: 13 });
  fireEvent.mouseDown(screen.getByLabelText('Transport'));
  fireEvent.click(await screen.findByText('gRPC'));
  fireEvent.click(screen.getByRole('button', { name: 'Create tunnel' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/add',
      {
        name: 'gs3',
        address: '198.51.100.7',
        port: 8443,
        engine: 'gost',
        transport: 'grpc',
        forwardPorts: ['443=127.0.0.1:2096'],
        routeInbounds: [],
        mode: 'reverse',
        options: {},
      },
      JSON_HEADERS,
    ),
  );
});

test('an engine that only runs in reverse mode offers no direct mode and starts on its first transport', async () => {
  openTunnels();
  await startTunnel('rh1', 'Rathole', 'TCP');

  expect(screen.getByRole('radio', { name: 'Reverse' })).toBeTruthy();
  expect(screen.queryByRole('radio', { name: 'Direct' })).toBeNull();
  fireEvent.click(screen.getByRole('button', { name: 'Create tunnel' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/add',
      expect.objectContaining({ engine: 'rathole', transport: 'tcp', mode: 'reverse' }),
      JSON_HEADERS,
    ),
  );
});

test('choosing a direct-only engine switches to direct mode and keeps only the transports that dial', async () => {
  openTunnels();
  await startTunnel('sb1', 'sing-box', 'WebSocket over TLS');

  expect((screen.getByRole('radio', { name: 'Direct' }) as HTMLInputElement).checked).toBe(true);
  expect(screen.queryByRole('radio', { name: 'Reverse' })).toBeNull();
  fireEvent.click(screen.getByRole('button', { name: 'Create tunnel' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/add',
      expect.objectContaining({ engine: 'singbox', transport: 'wss', mode: 'direct' }),
      JSON_HEADERS,
    ),
  );
});

test('switching the mode replaces a transport the new mode cannot run', async () => {
  openTunnels();
  await startTunnel('kz1', 'Kariz', 'TCP');

  fireEvent.mouseDown(screen.getByLabelText('Transport'));
  fireEvent.click(await screen.findByText('KCP'));
  fireEvent.click(screen.getByRole('radio', { name: 'Direct' }));
  await waitFor(() =>
    expect(screen.getByLabelText('Transport').closest('.ant-select')?.textContent).toBe('TCP'),
  );
  fireEvent.click(screen.getByRole('button', { name: 'Create tunnel' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/add',
      expect.objectContaining({ engine: 'kariz', transport: 'tcp', mode: 'direct' }),
      JSON_HEADERS,
    ),
  );
});

test('an engine setting is offered as a choice and travels in options', async () => {
  openTunnels();
  await startTunnel('kz2', 'Kariz', 'TCP');

  fireEvent.click(await screen.findByText('Advanced options'));
  fireEvent.mouseDown(await screen.findByLabelText('profile'));
  fireEvent.click(await screen.findByText('gaming'));
  fireEvent.click(screen.getByRole('button', { name: 'Create tunnel' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/add',
      expect.objectContaining({ engine: 'kariz', options: { profile: 'gaming' } }),
      JSON_HEADERS,
    ),
  );
});

test('the edit dialog of a forwarding tunnel submits its transport, ports and inbounds', async () => {
  listed = [gostPortal];
  openTunnels();

  fireEvent.click(await screen.findByRole('button', { name: 'Edit gs1' }));
  fireEvent.mouseDown(await screen.findByLabelText('Transport'));
  fireEvent.click(await screen.findByText('gRPC'));
  fireEvent.click(screen.getByRole('button', { name: 'Save' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/update/8',
      {
        address: '203.0.113.10',
        port: 7000,
        routeInbounds: ['inbound-443'],
        transport: 'grpc',
        forwardPorts: ['443=127.0.0.1:2096'],
        options: {},
      },
      JSON_HEADERS,
    ),
  );
});

test('the exit end of a forwarding tunnel edits only its exit outbound', async () => {
  listed = [gostBridge];
  openTunnels();

  fireEvent.click(await screen.findByRole('button', { name: 'Edit gs2' }));
  expect(screen.queryByLabelText('Transport')).toBeNull();
  fireEvent.change(await screen.findByLabelText('Exit outbound'), { target: { value: 'warp' } });
  fireEvent.click(screen.getByRole('button', { name: 'Save' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/update/9',
      { exitOutbound: 'warp', options: {} },
      JSON_HEADERS,
    ),
  );
});

test('a forwarding row names its engine and the Other engines filter lists only those rows', async () => {
  listed = [gostPortal, { ...gostPortal, id: 10, name: 'bp9', engine: 'backpack' }];
  openTunnels();

  expect(await screen.findByText('gs1')).toBeTruthy();
  fireEvent.click(screen.getByRole('button', { name: /Other engines/ }));
  expect(screen.getByText('gs1')).toBeTruthy();
  expect(screen.queryByText('bp9')).toBeNull();
  const row = screen.getByText('gs1').closest('tr') as HTMLElement;
  expect(within(row).getByText('GOST')).toBeTruthy();
});

test('every forwarding engine has its own install card', async () => {
  openTunnels();

  for (const label of ['GOST', 'Rathole', 'Backhaul', 'Kariz', 'sing-box', 'Xray']) {
    expect(await screen.findByText(new RegExp(`${label} engine v1.0.0`))).toBeTruthy();
  }
});

test('kernel carriers show up only once the advanced engines are switched on, and have nothing to install', async () => {
  openTunnels();

  expect(await screen.findByText(/GOST engine v1.0.0/)).toBeTruthy();
  expect(screen.queryByText(/Kernel carriers engine/)).toBeNull();
  fireEvent.click(screen.getByRole('switch', { name: 'Advanced engines' }));

  expect(await screen.findByText(/Kernel carriers engine v1.0.0/)).toBeTruthy();
  expect(screen.getByRole('button', { name: 'GOST options' })).toBeTruthy();
  expect(screen.queryByRole('button', { name: 'Kernel carriers options' })).toBeNull();
});

test('a kernel carrier tunnel cannot be made without the public address of the dialing end', async () => {
  openTunnels();
  fireEvent.click(await screen.findByRole('switch', { name: 'Advanced engines' }));
  await startTunnel('kc1', 'Kernel carriers', 'GRE');

  fireEvent.click(screen.getByRole('button', { name: 'Create tunnel' }));
  expect(await screen.findByText('This setting is required.')).toBeTruthy();
  expect(HttpUtil.post).not.toHaveBeenCalled();

  fireEvent.change(screen.getByLabelText('dialerIp'), { target: { value: '198.51.100.9' } });
  fireEvent.click(screen.getByRole('button', { name: 'Create tunnel' }));
  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/add',
      expect.objectContaining({
        engine: 'carrier',
        transport: 'gre',
        options: { dialerIp: '198.51.100.9' },
      }),
      JSON_HEADERS,
    ),
  );
});

test('a tunnel already on a kernel carrier keeps the advanced engines in view', async () => {
  listed = [{ ...gostPortal, id: 21, name: 'kc9', engine: 'carrier', transport: 'gre' }];
  openTunnels();

  expect(await screen.findByText(/Kernel carriers engine v1.0.0/)).toBeTruthy();
  expect(
    screen.getByRole('switch', { name: 'Advanced engines' }).getAttribute('aria-checked'),
  ).toBe('true');
});
