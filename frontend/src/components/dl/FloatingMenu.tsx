import { useEffect, useLayoutEffect, useRef, useState } from 'react';
import type { HTMLAttributes, ReactNode, RefObject } from 'react';
import { createPortal } from 'react-dom';

const GAP = 6;
const EDGE = 8;
const MAX_HEIGHT = 320;

interface Place {
  top: number;
  left: number;
  maxHeight: number;
}

// Drawn in the body so no table, card or dialog can clip it; it flips up near the bottom edge.
export function FloatingMenu({
  anchor,
  align = 'start',
  onClose,
  children,
  ...rest
}: {
  anchor: RefObject<HTMLElement | null>;
  align?: 'start' | 'end';
  onClose: () => void;
  children: ReactNode;
} & Omit<HTMLAttributes<HTMLDivElement>, 'className' | 'style'>) {
  const menu = useRef<HTMLDivElement>(null);
  const [place, setPlace] = useState<Place | null>(null);

  const onCloseRef = useRef(onClose);
  useEffect(() => {
    onCloseRef.current = onClose;
  });

  useLayoutEffect(() => {
    const reposition = () => {
      const a = anchor.current;
      const m = menu.current;
      if (!a || !m) return;
      m.style.maxHeight = '';
      const r = a.getBoundingClientRect();
      const size = m.getBoundingClientRect();
      const below = window.innerHeight - r.bottom - GAP - EDGE;
      const above = r.top - GAP - EDGE;
      const up = size.height > below && above > below;
      const maxHeight = Math.min(MAX_HEIGHT, Math.max(up ? above : below, 0));
      const height = Math.min(size.height, maxHeight);
      const rtl = getComputedStyle(a).direction === 'rtl';
      const edgeOnRight = (align === 'end') !== rtl;
      const left = Math.min(
        Math.max(edgeOnRight ? r.right - size.width : r.left, EDGE),
        Math.max(window.innerWidth - size.width - EDGE, EDGE),
      );
      const next = { top: up ? r.top - GAP - height : r.bottom + GAP, left, maxHeight };
      setPlace((prev) =>
        prev &&
        prev.top === next.top &&
        prev.left === next.left &&
        prev.maxHeight === next.maxHeight
          ? prev
          : next,
      );
    };
    const away = (e: MouseEvent) => {
      const target = e.target as Node;
      if (!anchor.current?.contains(target) && !menu.current?.contains(target))
        onCloseRef.current();
    };
    const key = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onCloseRef.current();
    };
    const follow = (e: Event) => {
      if (!menu.current?.contains(e.target as Node)) reposition();
    };
    reposition();
    document.addEventListener('mousedown', away);
    document.addEventListener('keydown', key);
    window.addEventListener('resize', reposition);
    window.addEventListener('scroll', follow, true);
    return () => {
      document.removeEventListener('mousedown', away);
      document.removeEventListener('keydown', key);
      window.removeEventListener('resize', reposition);
      window.removeEventListener('scroll', follow, true);
    };
  }, [anchor, align]);

  return createPortal(
    <div
      ref={menu}
      className="dropmenu float"
      tabIndex={-1}
      style={{
        top: place?.top ?? 0,
        left: place?.left ?? 0,
        maxHeight: place?.maxHeight,
        visibility: place ? undefined : 'hidden',
      }}
      {...rest}
    >
      {children}
    </div>,
    document.body,
  );
}
