import type { CSSProperties } from 'react';

import { deriveAccent } from '@/lib/theme/deriveAccent';
import { resolveNodeAccent } from '@/lib/theme/nodeAccent';

type AccentVars = Record<'--acc' | '--acc-line' | '--acc-text' | '--on-acc', string>;

// Class and inline vars that give one element a node's accent inside a page in another accent.
export function accentScope(
  accent: string,
  country: string,
  isDark: boolean,
): { className: string; style?: CSSProperties & Partial<AccentVars> } {
  const theme = isDark ? 't-dark' : 't-light';
  const resolved = resolveNodeAccent(accent ?? '', country ?? '');
  if (resolved.kind === 'preset')
    return { className: `ac-${resolved.id} ${theme}`, style: undefined };
  const d = deriveAccent(resolved.hex, isDark ? 'dark' : 'light');
  return {
    className: `ac-custom ${theme}`,
    style: { '--acc': d.fill, '--acc-line': d.line, '--acc-text': d.text, '--on-acc': d.on },
  };
}
