export const XRAY_RESTART_STEPS = 3;

// Stage 0 sends the restart, 1 waits until status reports the core running, 2 is done.
export async function runXrayRestart({
  restart,
  readState,
  onStep,
  pollMs = 800,
  timeoutMs = 20000,
}: {
  restart: () => Promise<{ success?: boolean; msg?: string } | null | undefined>;
  readState: () => Promise<string>;
  onStep: (step: number) => void;
  pollMs?: number;
  timeoutMs?: number;
}): Promise<{ ok: true } | { ok: false; error: string }> {
  onStep(0);
  const res = await restart();
  if (!res?.success) return { ok: false, error: res?.msg || 'restart failed' };
  onStep(1);
  const deadline = Date.now() + timeoutMs;
  let state = '';
  while (Date.now() < deadline) {
    state = await readState();
    if (state === 'running') {
      onStep(2);
      return { ok: true };
    }
    await new Promise((r) => setTimeout(r, pollMs));
  }
  return { ok: false, error: state || 'timeout' };
}
