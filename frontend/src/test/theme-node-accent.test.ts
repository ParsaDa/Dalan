import { describe, expect, test } from 'vitest';

import { resolveNodeAccent } from '@/lib/theme/nodeAccent';

describe('resolveNodeAccent', () => {
  test('an explicit preset wins over the country', () => {
    expect(resolveNodeAccent('cobalt', 'NL')).toEqual({ kind: 'preset', id: 'cobalt' });
  });

  test('an allowed custom colour is used as given, lower-cased', () => {
    expect(resolveNodeAccent('#4C8DFF', 'NL')).toEqual({ kind: 'custom', hex: '#4c8dff' });
  });

  test('a custom colour on the success hue is refused and the country accent applies', () => {
    expect(resolveNodeAccent('#34d399', 'NL')).toEqual({ kind: 'preset', id: 'tulip' });
  });

  test('without an accent the country picks one, case-insensitively', () => {
    expect(resolveNodeAccent('', 'de')).toEqual({ kind: 'preset', id: 'sun' });
  });

  test('a country with no mapping gets sand, not the panel default', () => {
    expect(resolveNodeAccent('', 'BR')).toEqual({ kind: 'preset', id: 'sand' });
  });

  test('a node with neither accent nor country keeps the panel default', () => {
    expect(resolveNodeAccent('', '')).toEqual({ kind: 'preset', id: 'saffron' });
  });

  test('an unknown accent name falls through to the country', () => {
    expect(resolveNodeAccent('teal', 'FR')).toEqual({ kind: 'preset', id: 'cobalt' });
  });
});
