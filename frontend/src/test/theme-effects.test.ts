import { describe, expect, it } from 'vitest';

import { resolveEffects } from '@/lib/theme/effects';

const strong = { reducedMotion: false, cores: 8, memoryGb: 8 };

describe('resolveEffects', () => {
  it('forces Off when the system asks for reduced motion, whatever was chosen', () => {
    for (const pref of ['auto', 'full', 'light', 'off'] as const) {
      expect(resolveEffects(pref, { ...strong, reducedMotion: true })).toBe('off');
    }
  });

  it('auto picks Full on a strong device and Light on a weak one', () => {
    expect(resolveEffects('auto', strong)).toBe('full');
    expect(resolveEffects('auto', { ...strong, cores: 4 })).toBe('light');
    expect(resolveEffects('auto', { ...strong, memoryGb: 4 })).toBe('light');
  });

  it('auto treats unreported hardware as strong, not weak', () => {
    expect(resolveEffects('auto', { reducedMotion: false })).toBe('full');
  });

  it('keeps an explicit choice even on a weak device', () => {
    expect(resolveEffects('full', { ...strong, cores: 2 })).toBe('full');
    expect(resolveEffects('light', strong)).toBe('light');
  });
});
