import { describe, expect, it } from 'vitest';

import { contrastRatio, deriveAccent } from '@/lib/theme/deriveAccent';

const CREAM = '#F4EFE6';
const WHITE = '#FFFFFF';
const INK = '#111114';

describe('deriveAccent', () => {
  // A fixed darkening percentage leaves pale customs below the text threshold on cream.
  it.each(['#FFE14D', '#7FE0FF', '#E8B8FF', '#4C8DFF'])(
    'light shades of %s reach 3:1 and 4.5:1',
    (hex) => {
      const a = deriveAccent(hex, 'light');
      for (const bg of [CREAM, WHITE]) {
        expect(contrastRatio(a.line, bg)).toBeGreaterThanOrEqual(3);
        expect(contrastRatio(a.text, bg)).toBeGreaterThanOrEqual(4.5);
      }
    },
  );

  it('keeps a dark fill as text in the dark theme only when it already passes', () => {
    expect(deriveAccent('#FFC83D', 'dark').text).toBe('#FFC83D');
    const blue = deriveAccent('#3050C0', 'dark');
    expect(blue.text).not.toBe('#3050C0');
    expect(contrastRatio(blue.text, '#1B1B20')).toBeGreaterThanOrEqual(4.5);
  });

  it('puts ink on light fills and white on deep ones', () => {
    expect(deriveAccent('#FFC83D', 'dark').on).toBe(INK);
    expect(deriveAccent('#2A1A7A', 'dark').on).toBe(WHITE);
  });
});
