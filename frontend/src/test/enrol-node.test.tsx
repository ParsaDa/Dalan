import { fireEvent, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import EnrolNode from '@/pages/nodes/EnrolNode';
import { ClipboardManager, HttpUtil, Msg } from '@/utils';
import { renderWithProviders } from './test-utils';

const command = "dalan enrol 'dalan://enrol/abc'";
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
