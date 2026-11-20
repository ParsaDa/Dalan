import { useEffect, useRef, useState } from 'react';

import { useTheme } from '@/hooks/useTheme';

const DURATION_MS = 700;

interface CountUpProps {
  value: number;
  format?: (value: number) => string;
}

const easeOut = (t: number) => 1 - Math.pow(1 - t, 3);

export default function CountUp({ value, format = String }: CountUpProps) {
  const { effects } = useTheme();
  const animate = effects !== 'off';
  const [shown, setShown] = useState(0);
  const shownRef = useRef(0);

  useEffect(() => {
    if (!animate) {
      shownRef.current = value;
      return;
    }
    const from = shownRef.current;
    const start = performance.now();
    let frame = requestAnimationFrame(function tick(now) {
      const progress = Math.min(1, (now - start) / DURATION_MS);
      const next = from + (value - from) * easeOut(progress);
      shownRef.current = next;
      setShown(next);
      if (progress < 1) frame = requestAnimationFrame(tick);
    });
    return () => cancelAnimationFrame(frame);
  }, [value, animate]);

  return (
    <>
      <span aria-hidden>{format(animate ? shown : value)}</span>
      <span className="dl-sr">{format(value)}</span>
    </>
  );
}
