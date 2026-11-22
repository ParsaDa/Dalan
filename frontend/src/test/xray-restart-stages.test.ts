import { describe, it, expect } from 'vitest';

import { runXrayRestart } from '@/pages/index/dashboard/xrayRestart';

describe('Xray restart stages', () => {
  it('waits for the core to report running before calling it done', async () => {
    const states = ['stop', 'stop', 'running'];
    const steps: number[] = [];
    const result = await runXrayRestart({
      restart: async () => ({ success: true }),
      readState: async () => states.shift() ?? 'running',
      onStep: (s) => steps.push(s),
      pollMs: 1,
      timeoutMs: 1000,
    });
    expect(result).toEqual({ ok: true });
    expect(steps).toEqual([0, 1, 2]);
    expect(states).toEqual([]);
  });

  it('stops at the first stage when the panel refuses the restart', async () => {
    const steps: number[] = [];
    const result = await runXrayRestart({
      restart: async () => ({ success: false, msg: 'config invalid' }),
      readState: async () => 'running',
      onStep: (s) => steps.push(s),
      pollMs: 1,
      timeoutMs: 1000,
    });
    expect(result).toEqual({ ok: false, error: 'config invalid' });
    expect(steps).toEqual([0]);
  });

  it('reports the Xray error when it never comes back', async () => {
    const result = await runXrayRestart({
      restart: async () => ({ success: true }),
      readState: async () => 'error',
      onStep: () => {},
      pollMs: 1,
      timeoutMs: 20,
    });
    expect(result).toEqual({ ok: false, error: 'error' });
  });
});
