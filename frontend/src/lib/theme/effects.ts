export type EffectsPref = 'auto' | 'full' | 'light' | 'off';
export type Effects = 'full' | 'light' | 'off';

export interface EffectsEnv {
  reducedMotion: boolean;
  cores?: number;
  memoryGb?: number;
}

const WEAK_CORES = 4;
const WEAK_MEMORY_GB = 4;

// Reduced motion always wins; "auto" only downgrades to Light when the device reports weak hardware.
export function resolveEffects(pref: EffectsPref, env: EffectsEnv): Effects {
  if (env.reducedMotion) return 'off';
  if (pref !== 'auto') return pref;
  const weakCpu = env.cores !== undefined && env.cores <= WEAK_CORES;
  const weakMemory = env.memoryGb !== undefined && env.memoryGb <= WEAK_MEMORY_GB;
  return weakCpu || weakMemory ? 'light' : 'full';
}

export function readEffectsEnv(): EffectsEnv {
  const nav = navigator as Navigator & { deviceMemory?: number };
  return {
    reducedMotion: window.matchMedia('(prefers-reduced-motion: reduce)').matches,
    cores: nav.hardwareConcurrency || undefined,
    memoryGb: nav.deviceMemory,
  };
}
