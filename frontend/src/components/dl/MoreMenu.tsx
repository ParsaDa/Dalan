import { useEffect, useRef, useState } from 'react';

import Icon from './Icon';
import type { IconName } from './Icon';

export interface MenuAction {
  key: string;
  label: string;
  icon?: IconName;
  danger?: boolean;
  disabled?: boolean;
  onSelect: () => void;
}

export function MoreMenu({
  label,
  actions,
  icon = 'more',
}: {
  label: string;
  actions: readonly MenuAction[];
  icon?: IconName;
}) {
  const [open, setOpen] = useState(false);
  const box = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!open) return;
    const away = (e: MouseEvent) => {
      if (box.current && !box.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener('mousedown', away);
    return () => document.removeEventListener('mousedown', away);
  }, [open]);
  return (
    <div className="dropchip" ref={box}>
      <button
        type="button"
        className="ib sm"
        aria-label={label}
        aria-haspopup="menu"
        aria-expanded={open}
        onClick={() => setOpen((o) => !o)}
      >
        <Icon name={icon} />
      </button>
      {open && (
        <div
          className="dropmenu end"
          role="menu"
          aria-label={label}
          tabIndex={-1}
          onKeyDown={(e) => {
            if (e.key === 'Escape') setOpen(false);
          }}
        >
          {actions.map((a) => (
            <button
              key={a.key}
              type="button"
              role="menuitem"
              className={`nav${a.danger ? ' dng' : ''}`}
              disabled={a.disabled}
              onClick={() => {
                setOpen(false);
                a.onSelect();
              }}
            >
              {a.icon && <Icon name={a.icon} />}
              <span className="f1">{a.label}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
