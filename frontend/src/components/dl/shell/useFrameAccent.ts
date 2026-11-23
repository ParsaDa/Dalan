import { useEffect } from 'react';

import { useTheme } from '@/hooks/useTheme';
import { resolveNodeAccent } from '@/lib/theme/nodeAccent';
import { useActiveNode } from './data';

// The whole frame wears the open node's accent; This server keeps the panel accent.
export function useFrameAccent() {
  const { active } = useActiveNode();
  const { setFrameAccent } = useTheme();
  const open = active !== null;
  const accent = active?.accent ?? '';
  const country = active?.country ?? '';
  useEffect(() => {
    setFrameAccent(open ? resolveNodeAccent(accent, country) : null);
    return () => setFrameAccent(null);
  }, [open, accent, country, setFrameAccent]);
}
