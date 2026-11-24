import type { Candidate, ForwardingEngine } from '@/generated/zod';
import type { TunnelMode } from '@/api/queries/useTunnels';
import { BACKPACK_TRANSPORTS } from './backpackTransports';
import { RAKHSH_STREAM_CARRIERS } from './rakhshCarriers';

// Mirrors benchmark.MaxCandidates; the panel refuses a longer list.
export const BENCHMARK_MAX = 60;

export type BenchmarkDirection = TunnelMode | 'both';

// BackPack and Rakhsh are compared as reverse tunnels over their stream transports only:
// a raw carrier forges packets and needs the peer's address, which a benchmark has no way to know.
function ownTransports(engine: string): string[] {
  if (engine === 'backpack') return BACKPACK_TRANSPORTS.map((o) => o.value);
  if (engine === 'rakhsh') return RAKHSH_STREAM_CARRIERS.map((o) => o.value);
  return [];
}

export function benchmarkCandidates(
  engines: readonly string[],
  direction: BenchmarkDirection,
  catalog: readonly ForwardingEngine[],
): Candidate[] {
  const modes: TunnelMode[] = direction === 'both' ? ['reverse', 'direct'] : [direction];
  const out: Candidate[] = [];
  for (const engine of engines) {
    if (engine === 'backpack' || engine === 'rakhsh') {
      if (!modes.includes('reverse')) continue;
      for (const transport of ownTransports(engine)) {
        out.push({ engine, transport, mode: 'reverse' });
      }
      continue;
    }
    const def = catalog.find((e) => e.id === engine);
    if (!def) continue;
    for (const mode of modes) {
      for (const tr of def.transports) {
        if (mode === 'direct' ? tr.direct : tr.reverse) {
          out.push({ engine, transport: tr.id, mode });
        }
      }
    }
  }
  return out;
}
