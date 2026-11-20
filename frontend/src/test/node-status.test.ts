import { describe, expect, it } from 'vitest';

import { nodeStatus } from '@/components/dl/shell/useFrameStatus';

describe('nodeStatus', () => {
  it.each([
    [{ id: 1, status: 'offline', cpuPct: 10 }, 'bad'],
    [{ id: 1, status: 'online', cpuPct: 91, memPct: 40 }, 'warn'],
    [{ id: 1, status: 'online', cpuPct: 40, memPct: 85 }, 'warn'],
    [{ id: 1, status: 'online', cpuPct: 84, memPct: 84 }, 'ok'],
    [{ id: 1, status: 'unknown' }, 'off'],
  ] as const)('%o is %s', (node, want) => {
    expect(nodeStatus(node)).toBe(want);
  });
});
