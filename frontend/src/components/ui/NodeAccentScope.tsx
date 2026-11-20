import type { CSSProperties, ReactNode } from 'react';

import { resolveNodeAccent } from '@/lib/theme/nodeAccent';

interface NodeAccentScopeProps {
  accent: string;
  country: string;
  children: ReactNode;
}

// Swaps the accent for everything inside, so one node's colour never leaks into the rail or header.
export default function NodeAccentScope({ accent, country, children }: NodeAccentScopeProps) {
  const resolved = resolveNodeAccent(accent, country);
  const style =
    resolved.kind === 'custom' ? ({ '--dl-custom': resolved.hex } as CSSProperties) : undefined;
  return (
    <div
      className="dl-frame"
      data-accent={resolved.kind === 'custom' ? 'custom' : resolved.id}
      style={style}
    >
      {children}
    </div>
  );
}
