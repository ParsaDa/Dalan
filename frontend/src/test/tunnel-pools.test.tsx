import { fireEvent, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import TunnelPools from '@/pages/tunnels/TunnelPools';
import { HttpUtil, Msg } from '@/utils';
import { renderWithProviders } from './test-utils';

const entry = (id: number, name: string, extra: Record<string, unknown> = {}) => ({
  id,
  name,
  role: 'portal',
  engine: 'xray',
  enable: true,
  address: '203.0.113.10',
  port: 8440 + id,
  uuid: '8d6f3a0e-5c1b-4f0e-9a51-2f6d1c7b9e10',
  security: 'none',
  serverName: '',
  realityPublicKey: '',
  shortId: '',
  routeInbounds: [],
  exitOutbound: '',
  transport: '',
  fallbackTransports: [],
  forwardPorts: [],
  acceptUdp: false,
  managePort: 0,
  parentId: 0,
  nodeId: 0,
  relayPort: 0,
  up: 0,
  down: 0,
  ...extra,
});
const tunnels = [
  entry(1, 'de1'),
  entry(2, 'nl1'),
  entry(3, 'node-x', { managePort: 61000 }),
  entry(4, 'ir1', { role: 'bridge' }),
];
let pools: Record<string, unknown>[];

beforeEach(() => {
  pools = [
    { id: 7, name: 'main', members: ['de1', 'nl1'], routeInbounds: ['in-1'], active: 'nl1' },
  ];
  vi.spyOn(HttpUtil, 'get').mockImplementation(async (url) =>
    url === '/panel/api/tunnels/pools/list' ? new Msg(true, '', pools) : new Msg(true, '', []),
  );
  vi.spyOn(HttpUtil, 'post').mockImplementation(async () => new Msg(true, '', {}));
});

afterEach(() => {
  vi.restoreAllMocks();
});

test('shows each pool with the tunnel that carries its traffic marked', async () => {
  renderWithProviders(<TunnelPools tunnels={tunnels} />);

  expect(await screen.findByText('main')).toBeTruthy();
  expect(screen.getByText('de1')).toBeTruthy();
  expect(screen.getByText('nl1 · in use')).toBeTruthy();
  expect(screen.getByText('in-1')).toBeTruthy();
});

// An open select renders its options twice (listbox and visible list); the visible one comes last.
async function pickOption(label: string) {
  const matches = await screen.findAllByText(label);
  fireEvent.click(matches[matches.length - 1]);
}

test('creates a pool from entry tunnels only, in the order they were picked', async () => {
  pools = [];
  renderWithProviders(<TunnelPools tunnels={tunnels} />);

  fireEvent.click(await screen.findByRole('button', { name: 'New pool' }));
  fireEvent.change(await screen.findByLabelText('Name'), { target: { value: 'main' } });
  fireEvent.mouseDown(screen.getByLabelText('Tunnels'));
  expect(screen.queryByText('node-x')).toBeNull();
  expect(screen.queryByText('ir1')).toBeNull();
  await pickOption('nl1');
  await pickOption('de1');
  fireEvent.click(screen.getByRole('button', { name: 'OK' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/pools/add',
      { name: 'main', members: ['nl1', 'de1'], routeInbounds: [] },
      { headers: { 'Content-Type': 'application/json' } },
    ),
  );
});

test('deletes a pool after confirmation', async () => {
  renderWithProviders(<TunnelPools tunnels={tunnels} />);

  fireEvent.click(await screen.findByRole('button', { name: 'Delete main' }));
  fireEvent.click(await screen.findByRole('button', { name: 'OK' }));

  await waitFor(() => expect(HttpUtil.post).toHaveBeenCalledWith('/panel/api/tunnels/pools/del/7'));
});
