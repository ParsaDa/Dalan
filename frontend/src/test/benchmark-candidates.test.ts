import { expect, test } from 'vitest';

import type { ForwardingEngine } from '@/generated/zod';
import { benchmarkCandidates } from '@/pages/tunnels/benchmarkCandidates';

const tr = (id: string, reverse: boolean, direct: boolean) => ({ id, label: id, reverse, direct });
const catalog: ForwardingEngine[] = [
  {
    id: 'gost',
    label: 'GOST',
    pinned: 'v3',
    transports: [tr('tls', true, true), tr('grpc', true, false)],
    options: [],
  },
  {
    id: 'singbox',
    label: 'sing-box',
    pinned: 'v1',
    transports: [tr('wss', false, true)],
    options: [],
  },
];

test('a forwarding engine contributes only the transports that run in the chosen direction', () => {
  expect(benchmarkCandidates(['gost'], 'reverse', catalog)).toEqual([
    { engine: 'gost', transport: 'tls', mode: 'reverse' },
    { engine: 'gost', transport: 'grpc', mode: 'reverse' },
  ]);
  expect(benchmarkCandidates(['gost'], 'direct', catalog)).toEqual([
    { engine: 'gost', transport: 'tls', mode: 'direct' },
  ]);
});

test('both directions list reverse methods first and skip an engine that cannot run one', () => {
  expect(benchmarkCandidates(['singbox', 'gost'], 'both', catalog)).toEqual([
    { engine: 'singbox', transport: 'wss', mode: 'direct' },
    { engine: 'gost', transport: 'tls', mode: 'reverse' },
    { engine: 'gost', transport: 'grpc', mode: 'reverse' },
    { engine: 'gost', transport: 'tls', mode: 'direct' },
  ]);
});

test('BackPack and Rakhsh are tried as reverse tunnels over stream transports, never raw carriers', () => {
  const backpack = benchmarkCandidates(['backpack'], 'reverse', catalog);
  expect(backpack.length).toBeGreaterThan(0);
  expect(backpack.every((c) => c.engine === 'backpack' && c.mode === 'reverse')).toBe(true);
  expect(benchmarkCandidates(['backpack', 'rakhsh'], 'direct', catalog)).toEqual([]);

  const rakhsh = benchmarkCandidates(['rakhsh'], 'reverse', catalog).map((c) => c.transport);
  expect(rakhsh).toContain('tls');
  expect(rakhsh).not.toContain('faketcp');
  expect(rakhsh).not.toContain('shabdiz-gre');
});

test('an engine the catalog has not delivered yet adds nothing', () => {
  expect(benchmarkCandidates(['gost'], 'reverse', [])).toEqual([]);
  expect(benchmarkCandidates(['not-an-engine'], 'reverse', catalog)).toEqual([]);
});
