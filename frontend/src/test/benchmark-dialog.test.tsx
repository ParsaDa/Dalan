import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import TunnelsPage from '@/pages/tunnels/TunnelsPage';
import { HttpUtil, Msg } from '@/utils';
import { renderWithProviders } from './test-utils';

const JSON_HEADERS = { headers: { 'Content-Type': 'application/json' } };
const transport = (id: string, label: string) => ({ id, label, reverse: true, direct: false });
const catalog = [
  {
    id: 'gost',
    label: 'GOST',
    pinned: 'v3.3.3',
    transports: [transport('tls', 'TLS'), transport('grpc', 'gRPC')],
    options: [],
  },
];
const method = (transportId: string, down: number, score: number) => ({
  engine: 'gost',
  transport: transportId,
  mode: 'reverse',
  ok: true,
  error: '',
  leftover: '',
  connectMs: 900,
  pingMs: 41,
  jitterMs: 4,
  downMbps: down,
  upMbps: 20,
  score,
});
const idle = {
  running: false,
  cancelled: false,
  objective: 'balanced',
  total: 0,
  done: 0,
  phase: '',
  results: [] as unknown[],
};

let benchmarkStatus: Record<string, unknown>;

function openBenchmark() {
  renderWithProviders(
    <MemoryRouter>
      <TunnelsPage />
    </MemoryRouter>,
  );
  return screen.findByRole('button', { name: 'Compare methods' }).then((button) => {
    fireEvent.click(button);
  });
}

beforeEach(() => {
  benchmarkStatus = idle;
  vi.spyOn(HttpUtil, 'get').mockImplementation(async (url) => {
    if (url === '/panel/api/nodes/list')
      return new Msg(true, '', [{ id: 9, name: 'nl1', enable: true }]);
    if (url === '/panel/api/tunnels/list') return new Msg(true, '', []);
    if (url === '/panel/api/tunnels/engines') return new Msg(true, '', catalog);
    if (url === '/panel/api/tunnels/benchmark/status') return new Msg(true, '', benchmarkStatus);
    if (url === '/panel/api/tunnels/engine')
      return new Msg(true, '', { installed: true, version: 'v1.0.0', pinned: 'v1.0.0' });
    return new Msg(true, '', []);
  });
  vi.spyOn(HttpUtil, 'post').mockImplementation(async () => new Msg(true, '', {}));
});

afterEach(() => {
  vi.restoreAllMocks();
});

test('starts a run with the node, address, ranking and the methods of the chosen engines', async () => {
  await openBenchmark();
  const dialog = await screen.findByRole('dialog');
  fireEvent.click(within(dialog).getByRole('button', { name: 'BackPack' }));
  fireEvent.click(within(dialog).getByRole('button', { name: 'GOST' }));
  fireEvent.change(within(dialog).getByLabelText("This server's address"), {
    target: { value: ' 203.0.113.10 ' },
  });
  await waitFor(() => expect(within(dialog).getByText('2 methods')).toBeTruthy());
  fireEvent.click(within(dialog).getByRole('button', { name: 'Start' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith(
      '/panel/api/tunnels/benchmark/start',
      {
        nodeId: 9,
        address: '203.0.113.10',
        objective: 'balanced',
        candidates: [
          { engine: 'gost', transport: 'tls', mode: 'reverse' },
          { engine: 'gost', transport: 'grpc', mode: 'reverse' },
        ],
      },
      JSON_HEADERS,
    ),
  );
});

test('cannot start a reverse run before this server has an address', async () => {
  await openBenchmark();
  const dialog = await screen.findByRole('dialog');

  expect(
    (within(dialog).getByRole('button', { name: 'Start' }) as HTMLButtonElement).disabled,
  ).toBe(true);
  expect(HttpUtil.post).not.toHaveBeenCalled();
});

test('a finished run names the best method and fills the new tunnel form with the one picked', async () => {
  benchmarkStatus = {
    ...idle,
    total: 2,
    done: 2,
    results: [method('tls', 40, 55), method('grpc', 95, 90)],
    best: { engine: 'gost', transport: 'grpc', mode: 'reverse' },
  };
  await openBenchmark();
  const dialog = await screen.findByRole('dialog');
  fireEvent.change(within(dialog).getByLabelText("This server's address"), {
    target: { value: '203.0.113.10' },
  });

  const rows = await within(dialog).findAllByRole('row');
  const grpc = rows.find((row) => within(row).queryByText(/· grpc ·/))!;
  expect(within(grpc).getByText(/Best/)).toBeTruthy();
  expect(
    within(rows.find((row) => within(row).queryByText(/· tls ·/))!).queryByText(/Best/),
  ).toBeNull();
  fireEvent.click(within(grpc).getByRole('button', { name: 'Use this method' }));

  expect(await screen.findByLabelText('Address')).toHaveProperty('value', '203.0.113.10');
  await waitFor(() =>
    expect(screen.getByLabelText('Transport').closest('.ant-select')?.textContent).toBe('gRPC'),
  );
});

test('the Start button and the progress stay outside the part of the dialog that scrolls', async () => {
  benchmarkStatus = {
    ...idle,
    running: true,
    total: 40,
    done: 3,
    phase: 'measure',
    current: { engine: 'gost', transport: 'tls', mode: 'reverse' },
    results: Array.from({ length: 30 }, (_, i) => method(`t${i}`, 80, 70)),
  };
  await openBenchmark();
  const dialog = await screen.findByRole('dialog');
  const scroller = await within(dialog).findByTestId('benchmark-scroll');

  expect(scroller.style.overflowY).toBe('auto');
  expect(within(scroller).getByRole('table')).toBeTruthy();
  const progress = within(dialog).getByText('3 of 40 done');
  expect(scroller.contains(progress)).toBe(false);
  expect(scroller.contains(within(dialog).getByRole('button', { name: 'Stop' }))).toBe(false);
  expect(dialog.style.maxHeight).not.toBe('');
});

test('Start is reachable without scrolling, whatever the settings above it hold', async () => {
  await openBenchmark();
  const dialog = await screen.findByRole('dialog');
  const scroller = await within(dialog).findByTestId('benchmark-scroll');

  expect(scroller.contains(within(dialog).getByRole('button', { name: 'Start' }))).toBe(false);
});
