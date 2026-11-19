import { describe, it, expect } from 'vitest';

import { TOR_COUNTRIES } from '@/pages/xray/overrides/torCountries';
import { countryName } from '@/pages/xray/outbounds/outbounds-tab-helpers';

describe('TOR_COUNTRIES', () => {
  it('holds distinct lower-case two-letter codes, and most of the world', () => {
    expect(new Set(TOR_COUNTRIES).size).toBe(TOR_COUNTRIES.length);
    for (const code of TOR_COUNTRIES) expect(code).toMatch(/^[a-z]{2}$/);
    expect(TOR_COUNTRIES.length).toBeGreaterThanOrEqual(200);
  });

  it('has a real English name for every code, and no two codes share one', () => {
    const names = new Map<string, string>();
    for (const code of TOR_COUNTRIES) {
      const name = countryName(code, 'en');
      expect(name, `${code} has no English name`).not.toBe(code.toUpperCase());
      expect(name, `${code} is not a country`).not.toMatch(/unknown/i);
      expect(names.get(name), `${code} and ${names.get(name)} are both ${name}`).toBeUndefined();
      names.set(name, code);
    }
  });

  it('spells the United Kingdom gb, the code Tor uses', () => {
    expect(TOR_COUNTRIES).toContain('gb');
    expect(TOR_COUNTRIES).not.toContain('uk');
  });
});
