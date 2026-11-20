import { ACCENTS, DEFAULT_ACCENT, accentForCountry, isCustomAccentAllowed } from './accents';
import type { AccentId } from './accents';

export type ResolvedAccent = { kind: 'preset'; id: AccentId } | { kind: 'custom'; hex: string };

export function resolveNodeAccent(accent: string, country: string): ResolvedAccent {
  const value = accent.trim().toLowerCase();
  const preset = ACCENTS.find((item) => item.id === value);
  if (preset) return { kind: 'preset', id: preset.id };
  if (isCustomAccentAllowed(value)) return { kind: 'custom', hex: value };
  if (country.trim()) return { kind: 'preset', id: accentForCountry(country) };
  return { kind: 'preset', id: DEFAULT_ACCENT };
}
