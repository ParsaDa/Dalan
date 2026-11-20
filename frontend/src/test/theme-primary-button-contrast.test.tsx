import { describe, expect, it } from 'vitest';

import { buildAntdThemeConfig } from '@/hooks/useTheme';

const channel = (v: number) => {
  const s = v / 255;
  return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4;
};
const luminance = (hex: string) => {
  const n = parseInt(hex.slice(1), 16);
  return (
    0.2126 * channel((n >> 16) & 255) + 0.7152 * channel((n >> 8) & 255) + 0.0722 * channel(n & 255)
  );
};
const contrast = (a: string, b: string) => {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x);
  return (hi + 0.05) / (lo + 0.05);
};

// Storybook's axe gate renders without panel.css, so antd's own primary fill must pass.
describe('primary button colours in the antd theme', () => {
  it.each([
    ['light', false],
    ['dark', true],
  ])('%s: label on the primary fill meets 4.5:1', (_name, isDark) => {
    const cfg = buildAntdThemeConfig(isDark, false);
    const button = cfg.components?.Button as { colorPrimary?: string; primaryColor: string };
    const fill = button.colorPrimary ?? (cfg.token?.colorPrimary as string);

    expect(contrast(fill, button.primaryColor)).toBeGreaterThanOrEqual(4.5);
  });
});
