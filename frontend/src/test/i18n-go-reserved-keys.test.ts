import { describe, expect, it } from 'vitest';

import enUS from '../../../internal/web/translation/en-US.json';

// go-i18n reads these names as plural forms and refuses a map that mixes them with other keys.
const RESERVED = new Set([
  'id',
  'description',
  'hash',
  'leftdelim',
  'rightdelim',
  'zero',
  'one',
  'two',
  'few',
  'many',
  'other',
]);

function mixed(node: unknown, path: string[] = [], out: string[] = []): string[] {
  if (node && typeof node === 'object') {
    const keys = Object.keys(node);
    const reserved = keys.filter((k) => RESERVED.has(k.toLowerCase()));
    if (reserved.length && reserved.length < keys.length)
      out.push(`${path.join('.')}: ${reserved.join(', ')}`);
    for (const k of keys) mixed((node as Record<string, unknown>)[k], [...path, k], out);
  }
  return out;
}

describe('translations the Go backend can load', () => {
  it('never mixes go-i18n reserved names with ordinary keys in one object', () => {
    expect(mixed(enUS)).toEqual([]);
  });
});
