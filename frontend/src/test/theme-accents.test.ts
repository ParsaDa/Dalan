import { describe, expect, it } from 'vitest';

import { ACCENTS, accentForCountry, hueOf, isCustomAccentAllowed } from '@/lib/theme/accents';

describe('accent palette', () => {
  it('has the twelve designed accents and no orange', () => {
    expect(ACCENTS.map((a) => a.id)).toEqual([
      'saffron',
      'rose',
      'sun',
      'tulip',
      'cobalt',
      'iris',
      'lime',
      'cyan',
      'amethyst',
      'ice',
      'sand',
      'steel',
    ]);
  });

  it('gives the Netherlands tulip, not the orange that reads as a warning', () => {
    expect(accentForCountry('NL')).toBe('tulip');
    expect(accentForCountry('nl')).toBe('tulip');
  });

  it('maps flag colours and falls back to sand for an unknown or empty country', () => {
    expect(accentForCountry('TR')).toBe('rose');
    expect(accentForCountry('DE')).toBe('sun');
    expect(accentForCountry('FR')).toBe('cobalt');
    expect(accentForCountry('GB')).toBe('iris');
    expect(accentForCountry('FI')).toBe('cyan');
    expect(accentForCountry('ZZ')).toBe('sand');
    expect(accentForCountry('')).toBe('sand');
  });
});

describe('custom accent', () => {
  it('reads hue in degrees', () => {
    expect(Math.round(hueOf('#ff0000'))).toBe(0);
    expect(Math.round(hueOf('#00ff00'))).toBe(120);
    expect(Math.round(hueOf('#0000ff'))).toBe(240);
  });

  it('refuses colours within 20 degrees of the warning or success hue', () => {
    expect(isCustomAccentAllowed('#ff7a45')).toBe(false);
    expect(isCustomAccentAllowed('#ff9a3d')).toBe(false);
    expect(isCustomAccentAllowed('#34d399')).toBe(false);
    expect(isCustomAccentAllowed('#2ed17a')).toBe(false);
  });

  it('allows colours clear of both', () => {
    expect(isCustomAccentAllowed('#4c8dff')).toBe(true);
    expect(isCustomAccentAllowed('#f062c8')).toBe(true);
    expect(isCustomAccentAllowed('#9b8cff')).toBe(true);
  });

  it('refuses text that is not a six-digit hex colour', () => {
    expect(isCustomAccentAllowed('blue')).toBe(false);
    expect(isCustomAccentAllowed('#12')).toBe(false);
  });
});
