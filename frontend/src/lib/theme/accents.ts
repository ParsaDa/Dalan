export type AccentId =
  | 'saffron'
  | 'rose'
  | 'sun'
  | 'tulip'
  | 'cobalt'
  | 'iris'
  | 'lime'
  | 'cyan'
  | 'amethyst'
  | 'ice'
  | 'sand'
  | 'steel';

export interface Accent {
  id: AccentId;
  fill: string;
}

export const DEFAULT_ACCENT: AccentId = 'saffron';

export const ACCENTS: readonly Accent[] = [
  { id: 'saffron', fill: '#FFB020' },
  { id: 'rose', fill: '#FF3D6E' },
  { id: 'sun', fill: '#FFC83D' },
  { id: 'tulip', fill: '#F062C8' },
  { id: 'cobalt', fill: '#4C8DFF' },
  { id: 'iris', fill: '#9B8CFF' },
  { id: 'lime', fill: '#B6F23C' },
  { id: 'cyan', fill: '#2CCCE4' },
  { id: 'amethyst', fill: '#C27CF5' },
  { id: 'ice', fill: '#8EC2FF' },
  { id: 'sand', fill: '#E2BE8A' },
  { id: 'steel', fill: '#9AA8BC' },
];

const COUNTRY_ACCENT: Record<string, AccentId> = {
  TR: 'rose',
  CA: 'rose',
  JP: 'rose',
  CH: 'rose',
  AT: 'rose',
  PL: 'rose',
  DE: 'sun',
  NL: 'tulip',
  FR: 'cobalt',
  US: 'cobalt',
  SE: 'cobalt',
  GB: 'iris',
  FI: 'cyan',
  AR: 'cyan',
  GR: 'ice',
  UY: 'ice',
};

export function accentForCountry(country: string): AccentId {
  return COUNTRY_ACCENT[country.trim().toUpperCase()] ?? 'sand';
}

const WARNING_HUE = 16;
const SUCCESS_HUE = 160;
const FORBIDDEN_HUE_RANGE = 20;

const HEX6 = /^#[0-9a-f]{6}$/i;

export function hueOf(hex: string): number {
  const r = parseInt(hex.slice(1, 3), 16) / 255;
  const g = parseInt(hex.slice(3, 5), 16) / 255;
  const b = parseInt(hex.slice(5, 7), 16) / 255;
  const max = Math.max(r, g, b);
  const delta = max - Math.min(r, g, b);
  if (delta === 0) return 0;
  let hue: number;
  if (max === r) hue = ((g - b) / delta) % 6;
  else if (max === g) hue = (b - r) / delta + 2;
  else hue = (r - g) / delta + 4;
  return (hue * 60 + 360) % 360;
}

function hueDistance(a: number, b: number): number {
  const d = Math.abs(a - b) % 360;
  return Math.min(d, 360 - d);
}

// A custom accent that sits on the warning or success hue would read as a status, so it is refused.
export function isCustomAccentAllowed(hex: string): boolean {
  if (!HEX6.test(hex)) return false;
  const hue = hueOf(hex);
  return (
    hueDistance(hue, WARNING_HUE) > FORBIDDEN_HUE_RANGE &&
    hueDistance(hue, SUCCESS_HUE) > FORBIDDEN_HUE_RANGE
  );
}
