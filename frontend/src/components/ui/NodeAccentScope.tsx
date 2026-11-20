import type { CSSProperties, ReactNode } from 'react';

import { resolveNodeAccent } from '@/lib/theme/nodeAccent';

interface NodeAccentScopeProps {
  // Null means "no node": the frame keeps the panel's own accent.
  node: { accent?: string; country?: string } | null;
  children: ReactNode;
}

// Swaps the accent for everything inside, so one node's colour never leaks into the rail or header.
export default function NodeAccentScope({ node, children }: NodeAccentScopeProps) {
  if (!node) return <div className="dl-frame">{children}</div>;
  const resolved = resolveNodeAccent(node.accent ?? '', node.country ?? '');
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
