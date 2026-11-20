// Same rule as the design system: lower OKLCH lightness at the same hue until contrast holds.
type Rgb = [number, number, number];

const lin = (c: number) => (c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4);
const unlin = (c: number) => (c <= 0.0031308 ? 12.92 * c : 1.055 * c ** (1 / 2.4) - 0.055);

function hexToRgb(hex: string): Rgb {
  const h = hex.replace('#', '');
  return [0, 2, 4].map((i) => parseInt(h.slice(i, i + 2), 16) / 255) as Rgb;
}
function rgbToHex(rgb: Rgb): string {
  return (
    '#' +
    rgb
      .map((c) =>
        Math.round(Math.max(0, Math.min(1, c)) * 255)
          .toString(16)
          .padStart(2, '0'),
      )
      .join('')
      .toUpperCase()
  );
}

function luminance(hex: string): number {
  const [r, g, b] = hexToRgb(hex).map(lin);
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

export function contrastRatio(a: string, b: string): number {
  const la = luminance(a);
  const lb = luminance(b);
  return (Math.max(la, lb) + 0.05) / (Math.min(la, lb) + 0.05);
}

function toOklch(hex: string): [number, number, number] {
  const [r, g, b] = hexToRgb(hex).map(lin);
  const l = Math.cbrt(0.4122214708 * r + 0.5363325363 * g + 0.0514459929 * b);
  const m = Math.cbrt(0.2119034982 * r + 0.6806995451 * g + 0.1073969566 * b);
  const s = Math.cbrt(0.0883024619 * r + 0.2817188376 * g + 0.6299787005 * b);
  const L = 0.2104542553 * l + 0.793617785 * m - 0.0040720468 * s;
  const A = 1.9779984951 * l - 2.428592205 * m + 0.4505937099 * s;
  const B = 0.0259040371 * l + 0.7827717662 * m - 0.808675766 * s;
  return [L, Math.hypot(A, B), Math.atan2(B, A)];
}

function fromOklch(L: number, C: number, H: number): string {
  for (let c = C; c >= 0; c -= 0.002) {
    const A = c * Math.cos(H);
    const B = c * Math.sin(H);
    const l = (L + 0.3963377774 * A + 0.2158037573 * B) ** 3;
    const m = (L - 0.1055613458 * A - 0.0638541728 * B) ** 3;
    const s = (L - 0.0894841775 * A - 1.291485548 * B) ** 3;
    const rgb: Rgb = [
      4.0767416621 * l - 3.3077115913 * m + 0.2309699292 * s,
      -1.2684380046 * l + 2.6097574011 * m - 0.3413193965 * s,
      -0.0041960863 * l - 0.7034186147 * m + 1.707614701 * s,
    ];
    if (rgb.every((x) => x >= -1e-4 && x <= 1 + 1e-4)) return rgbToHex(rgb.map(unlin) as Rgb);
  }
  return rgbToHex([L, L, L].map(unlin) as Rgb);
}

const MARGIN = 0.15;
const LIGHT_GROUNDS = ['#F4EFE6', '#FFFFFF'];
const DARK_GROUNDS = ['#111114', '#1B1B20'];

function shift(hex: string, target: number, grounds: readonly string[], dir: 1 | -1): string {
  const [L0, C, H] = toOklch(hex);
  for (let L = L0; L > 0.05 && L < 0.99; L += 0.004 * dir) {
    const c = fromOklch(L, C, H);
    if (grounds.every((g) => contrastRatio(c, g) >= target + MARGIN)) return c;
  }
  return dir < 0 ? '#000000' : '#FFFFFF';
}

export interface DerivedAccent {
  fill: string;
  line: string;
  text: string;
  on: string;
}

export function deriveAccent(hex: string, theme: 'dark' | 'light'): DerivedAccent {
  const fill = hex.toUpperCase();
  const on =
    contrastRatio(fill, '#111114') >= contrastRatio(fill, '#FFFFFF') ? '#111114' : '#FFFFFF';
  if (theme === 'light') {
    return {
      fill,
      on,
      line: shift(fill, 3, LIGHT_GROUNDS, -1),
      text: shift(fill, 4.5, LIGHT_GROUNDS, -1),
    };
  }
  const passes = DARK_GROUNDS.every((g) => contrastRatio(fill, g) >= 4.5 + MARGIN);
  return { fill, on, line: fill, text: passes ? fill : shift(fill, 4.5, DARK_GROUNDS, 1) };
}
