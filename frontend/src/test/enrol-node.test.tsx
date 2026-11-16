import { fireEvent, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import EnrolNode from '@/pages/nodes/EnrolNode';
import { ClipboardManager, HttpUtil, Msg } from '@/utils';
import { renderWithProviders } from './test-utils';

const command = "dalan enrol 'dalan://enrol/abc'";
const installCommand =
  "XUI_ENROL='dalan://enrol/abc' bash <(curl -Ls https://raw.githubusercontent.com/ParsaDa/dalan/main/install.sh)";
const waiting = { id: 2, name: 'nl1', expiresAt: 1900000000, usedAt: 0, tunnelId: 5, nodeId: 0 };
const joined = {
  id: 1,
  name: 'de1',
  expiresAt: 1700000000,
  usedAt: 1700000100,
  tunnelId: 4,
  nodeId: 9,
};
let listed: Record<string, unknown>[];

beforeEach(() => {
  listed = [];
  vi.spyOn(HttpUtil, 'get').mockImplementation(async () => new Msg(true, '', listed));
  vi.spyOn(HttpUtil, 'post').mockImplementation(async (url) =>
    url === '/panel/api/enrolments/add'
      ? new Msg(true, '', {
          id: 3,
          name: 'tr1',
          expiresAt: 1900000000,
          link: 'dalan://enrol/abc',
          command,
          installCommand,
        })
      : new Msg(true, '', null),
  );
});

afterEach(() => {
  vi.restoreAllMocks();
});

async function openDialog() {
  renderWithProviders(<EnrolNode />);
  fireEvent.click(screen.getByRole('button', { name: 'Add node with a command' }));
  return screen.findByLabelText('Node name');
}

test('offers a command that installs Dalan on a fresh server and joins this panel in one go', async () => {
  vi.spyOn(ClipboardManager, 'copyText').mockResolvedValue(true);
  fireEvent.change(await openDialog(), { target: { value: 'tr1' } });
  fireEvent.change(screen.getByLabelText('Server name (SNI)'), {
    target: { value: 'www.example.org' },
  });
  fireEvent.click(screen.getByRole('button', { name: 'Create command' }));

  expect(await screen.findByText(installCommand)).toBeTruthy();
  fireEvent.click(screen.getByRole('button', { name: 'Copy install command' }));

  await waitFor(() => expect(ClipboardManager.copyText).toHaveBeenCalledWith(installCommand));
});

test('creates an enrolment for this panel and shows the one-time command to copy', async () => {
  vi.spyOn(ClipboardManager, 'copyText').mockResolvedValue(true);
  fireEvent.change(await openDialog(), { target: { value: 'tr1' } });
  fireEvent.change(screen.getByLabelText('Server name (SNI)'), {
    target: { value: 'www.example.org' },
  });
  fireEvent.click(screen.getByRole('button', { name: 'Create command' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/enrolments/add',
      {
        name: 'tr1',
        panelUrl: `${window.location.origin}/`,
        address: window.location.hostname,
        port: 8443,
        security: 'reality',
        serverName: 'www.example.org',
        withTunnel: true,
      },
      { headers: { 'Content-Type': 'application/json' } },
    ),
  );
  expect(await screen.findByText(command)).toBeTruthy();
  expect(screen.queryByRole('button', { name: 'Create command' })).toBeNull();

  fireEvent.click(screen.getByRole('button', { name: 'Copy command' }));
  await waitFor(() => expect(ClipboardManager.copyText).toHaveBeenCalledWith(command));
});

test('refuses a name the tunnel cannot carry without calling the panel', async () => {
  fireEvent.change(await openDialog(), { target: { value: 'DE 1' } });
  fireEvent.change(screen.getByLabelText('Server name (SNI)'), {
    target: { value: 'www.example.org' },
  });
  fireEvent.click(screen.getByRole('button', { name: 'Create command' }));

  expect(
    await screen.findByText('Lower-case letters, digits and dashes, at most 26 characters.'),
  ).toBeTruthy();
  expect(HttpUtil.post).not.toHaveBeenCalled();
});

test('a REALITY tunnel is not requested without a site to imitate, but an unencrypted one is', async () => {
  fireEvent.change(await openDialog(), { target: { value: 'tr1' } });
  fireEvent.click(screen.getByRole('button', { name: 'Create command' }));
  expect(await screen.findByText('This field is required.')).toBeTruthy();
  expect(HttpUtil.post).not.toHaveBeenCalled();

  fireEvent.click(screen.getByLabelText('None (not encrypted)'));
  fireEvent.click(screen.getByRole('button', { name: 'Create command' }));
  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/enrolments/add',
      expect.objectContaining({ name: 'tr1', security: 'none', serverName: '' }),
      expect.anything(),
    ),
  );
});

test('lists only the enrolments still waiting and cancels one', async () => {
  listed = [joined, waiting];
  await openDialog();

  expect(await screen.findByText('nl1')).toBeTruthy();
  expect(screen.queryByText('de1')).toBeNull();

  fireEvent.click(screen.getByRole('button', { name: 'Cancel enrolment' }));
  fireEvent.click(await screen.findByRole('button', { name: 'OK' }));
  await waitFor(() => expect(HttpUtil.post).toHaveBeenCalledWith('/panel/api/enrolments/del/2'));
});

test('carries a node on an existing tunnel without asking for an address of its own', async () => {
  const portal = {
    id: 4,
    name: 'tr1',
    role: 'portal',
    engine: 'xray',
    enable: true,
    address: '203.0.113.10',
    port: 8443,
    uuid: '8d6f3a0e-5c1b-4f0e-9a51-2f6d1c7b9e10',
    security: 'none',
    serverName: '',
    realityPublicKey: '',
    shortId: '',
    routeInbounds: null,
    exitOutbound: '',
    transport: '',
    fallbackTransports: null,
    forwardPorts: null,
    acceptUdp: false,
    relayPort: 0,
    managePort: 0,
    parentId: 0,
  };
  vi.spyOn(HttpUtil, 'get').mockImplementation(
    async (url) => new Msg(true, '', url === '/panel/api/tunnels/list' ? [portal] : listed),
  );
  fireEvent.change(await openDialog(), { target: { value: 'tr2' } });
  fireEvent.mouseDown(await screen.findByLabelText('Tunnel that carries it'));
  fireEvent.click(await screen.findByText('tr1 · 203.0.113.10:8443'));

  expect(screen.queryByLabelText('Address')).toBeNull();
  fireEvent.click(screen.getByRole('button', { name: 'Create command' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/enrolments/add',
      { name: 'tr2', panelUrl: `${window.location.origin}/`, tunnelId: 4 },
      { headers: { 'Content-Type': 'application/json' } },
    ),
  );
});

test('makes a tunnel for the node by default and leaves it out when unticked', async () => {
  fireEvent.change(await openDialog(), { target: { value: 'tr1' } });
  fireEvent.click(screen.getByLabelText('None (not encrypted)'));
  fireEvent.click(screen.getByLabelText('Also make a tunnel for this node\u2019s traffic'));
  fireEvent.click(screen.getByRole('button', { name: 'Create command' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/enrolments/add',
      expect.not.objectContaining({ withTunnel: true }),
      expect.anything(),
    ),
  );
});
