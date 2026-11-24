import type { ForwardingEngine } from '@/generated/zod';
import type { TunnelMode, TunnelOption } from '@/api/queries/useTunnels';

// Engines whose process Dalan runs with a file per end; the engine id is what the tunnel stores.
export const FORWARDER_LABELS = {
  gost: 'GOST',
  rathole: 'Rathole',
  backhaul: 'Backhaul',
  kariz: 'Kariz',
  singbox: 'sing-box',
  xraycore: 'Xray',
} as const;

export type ForwarderEngine = keyof typeof FORWARDER_LABELS;

export const FORWARDER_ENGINES = Object.keys(FORWARDER_LABELS) as ForwarderEngine[];

export function isForwarder(engine: string | undefined): engine is ForwarderEngine {
  return engine !== undefined && Object.hasOwn(FORWARDER_LABELS, engine);
}

// The modes an engine can run at least one transport in, reverse first.
export function modesOf(engine: ForwardingEngine): TunnelMode[] {
  const modes: TunnelMode[] = [];
  if (engine.transports.some((tr) => tr.reverse)) modes.push('reverse');
  if (engine.transports.some((tr) => tr.direct)) modes.push('direct');
  return modes;
}

// The mode to use after the engine changed: the one in use when the engine can run it, else its first.
export function modeAfterEngineChange(engine: ForwardingEngine, current: TunnelMode): TunnelMode {
  const modes = modesOf(engine);
  return modes.includes(current) ? current : (modes[0] ?? current);
}

export function transportsFor(
  engine: ForwardingEngine,
  mode: TunnelMode,
): { value: string; label: string }[] {
  return engine.transports
    .filter((tr) => (mode === 'direct' ? tr.direct : tr.reverse))
    .map((tr) => ({ value: tr.id, label: tr.label }));
}

// The transport to keep after the engine or the mode changed: the current one if it still runs, else the first.
export function transportAfterChange(
  engine: ForwardingEngine,
  mode: TunnelMode,
  current: string | undefined,
): string {
  const options = transportsFor(engine, mode);
  return options.some((o) => o.value === current) ? (current as string) : (options[0]?.value ?? '');
}

// An engine's settings in the shape the shared option form reads; each one is a choice with a default.
export function settingsAsOptions(engine: ForwardingEngine | undefined): TunnelOption[] {
  return (engine?.options ?? []).map((setting) => ({
    key: setting.key,
    kind: 'choice',
    role: 'both',
    mode: 'both',
    group: 'general',
    choices: setting.choices,
    min: 0,
    max: 0,
    hintEn: `${setting.label}. Default: ${setting.default}.`,
    hintFa: `${setting.label}. Default: ${setting.default}.`,
  }));
}
