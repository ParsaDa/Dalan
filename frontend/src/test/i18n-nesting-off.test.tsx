import { afterEach, describe, expect, it, vi } from 'vitest';

describe('i18next nesting', () => {
  afterEach(() => vi.restoreAllMocks());

  // Closed references only: an unclosed "$t(" would hang a regressed run instead of failing it.
  it('is off, so "$t(...)" in a missing key or an interpolated value stays plain text', async () => {
    const utils = await import('@/utils');
    vi.spyOn(utils.LanguageManager, 'getLanguage').mockReturnValue('en-US');
    const { readyI18n } = await import('@/i18n/react');
    const i18n = await readyI18n();

    expect(i18n.t('$t(dashboard.nodes.thisNode)')).toBe('$t(dashboard.nodes.thisNode)');
    expect(i18n.t('dashboard.nodes.behindTitle', { name: '$t(dashboard.nodes.thisNode)' })).toBe(
      'Nodes behind $t(dashboard.nodes.thisNode)',
    );
  });
});
