import { describe, expect, it } from 'vitest';

import { accentScope } from '@/components/dl/accentScope';

// A node card must carry its own accent: the root accent belongs to the open node, not each card.
describe('accent scope for a node card', () => {
  it('uses the preset class together with the theme', () => {
    expect(accentScope('rose', 'TR', true)).toEqual({
      className: 'ac-rose t-dark',
      style: undefined,
    });
  });

  it('falls back to the flag colour when the node has no accent', () => {
    expect(accentScope('', 'DE', false).className).toBe('ac-sun t-light');
  });

  it('derives the three shades for a custom colour', () => {
    const scope = accentScope('#3366ff', '', false);
    expect(scope.className).toBe('ac-custom t-light');
    expect(Object.keys(scope.style ?? {}).sort()).toEqual([
      '--acc',
      '--acc-line',
      '--acc-text',
      '--on-acc',
    ]);
    expect(scope.style?.['--acc']?.toLowerCase()).toBe('#3366ff');
  });
});
