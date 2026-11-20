import { fireEvent, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import EnrolNode, { EnrolNodeDialog } from '@/pages/nodes/EnrolNode';
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
  fireEvent.click(screen.getByRole('button', { name: 'Create command' }));

  expect(await screen.findByText(installCommand)).toBeTruthy();
  fireEvent.click(screen.getByRole('button', { name: 'Copy install command' }));

  await waitFor(() => expect(ClipboardManager.copyText).toHaveBeenCalledWith(installCommand));
});

test('creates a BackPack tunnel on wssmux for the node by default and shows the one-time command', async () => {
  vi.spyOn(ClipboardManager, 'copyText').mockResolvedValue(true);
  fireEvent.change(await openDialog(), { target: { value: 'tr1' } });
  fireEvent.click(screen.getByRole('button', { name: 'Create command' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/enrolments/add',
      {
        name: 'tr1',
        panelUrl: `${window.location.origin}/`,
        address: window.location.hostname,
        port: 8443,
        engine: 'backpack',
        transport: 'wssmux',
      },
      { headers: { 'Content-Type': 'application/json' } },
    ),
  );
  expect(await screen.findByText(command)).toBeTruthy();
  expect(screen.queryByRole('button', { name: 'Create command' })).toBeNull();

  fireEvent.click(screen.getByRole('button', { name: 'Copy command' }));
  await waitFor(() => expect(ClipboardManager.copyText).toHaveBeenCalledWith(command));
});

test('asks for a Rakhsh tunnel on wss when Rakhsh is chosen', async () => {
  fireEvent.change(await openDialog(), { target: { value: 'tr1' } });
  fireEvent.click(screen.getByLabelText('Rakhsh'));
  fireEvent.click(screen.getByRole('button', { name: 'Create command' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/enrolments/add',
      expect.objectContaining({ name: 'tr1', engine: 'rakhsh', transport: 'wss' }),
      expect.anything(),
    ),
  );
});

test('has no Xray, REALITY or site-name choice', async () => {
  await openDialog();

  expect(screen.queryByLabelText('Server name (SNI)')).toBeNull();
  expect(screen.queryByLabelText('Xray')).toBeNull();
  expect(screen.queryByText('REALITY')).toBeNull();
});

test('refuses a name the tunnel cannot carry without calling the panel', async () => {
  fireEvent.change(await openDialog(), { target: { value: 'DE 1' } });
  fireEvent.click(screen.getByRole('button', { name: 'Create command' }));

  expect(
    await screen.findByText('Lower-case letters, digits and dashes, at most 26 characters.'),
  ).toBeTruthy();
  expect(HttpUtil.post).not.toHaveBeenCalled();
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

const entryTunnel = (id: number, name: string, routeInbounds: string[] | null) => ({
  id,
  name,
  role: 'portal',
  engine: 'backpack',
  enable: true,
  address: '203.0.113.10',
  port: 8443 + id,
  uuid: '8d6f3a0e-5c1b-4f0e-9a51-2f6d1c7b9e10',
  routeInbounds,
  exitOutbound: '',
  transport: 'wssmux',
  fallbackTransports: null,
  forwardPorts: null,
  acceptUdp: false,
  relayPort: 0,
  managePort: 0,
  mode: 'reverse',
  options: null,
  parentId: 0,
});

test('carries a node on an existing tunnel, marks the one users ride, and asks for no address', async () => {
  vi.spyOn(HttpUtil, 'get').mockImplementation(
    async (url) =>
      new Msg(
        true,
        '',
        url === '/panel/api/tunnels/list'
          ? [entryTunnel(4, 'tr1', ['inbound-443']), entryTunnel(5, 'spare', null)]
          : listed,
      ),
  );
  fireEvent.change(await openDialog(), { target: { value: 'tr2' } });
  fireEvent.mouseDown(await screen.findByLabelText('How should it connect?'));

  expect(await screen.findByText('spare · 203.0.113.10:8448')).toBeTruthy();
  fireEvent.click(await screen.findByText('tr1 · 203.0.113.10:8447 · carries your users'));

  expect(screen.queryByLabelText('Address')).toBeNull();
  expect(screen.queryByLabelText('Engine')).toBeNull();
  fireEvent.click(screen.getByRole('button', { name: 'Create command' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/enrolments/add',
      { name: 'tr2', panelUrl: `${window.location.origin}/`, tunnelId: 4 },
      { headers: { 'Content-Type': 'application/json' } },
    ),
  );
});

test('opens prefilled with the name of a node that has to enrol again', async () => {
  renderWithProviders(<EnrolNodeDialog open initialName="de1" onClose={() => {}} />);

  expect(((await screen.findByLabelText('Node name')) as HTMLInputElement).value).toBe('de1');
});
