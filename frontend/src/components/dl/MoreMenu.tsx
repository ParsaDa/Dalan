import { useRef, useState } from 'react';

import { FloatingMenu } from './FloatingMenu';
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
        <FloatingMenu
          anchor={box}
          align="end"
          onClose={() => setOpen(false)}
          role="menu"
          aria-label={label}
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
        </FloatingMenu>
      )}
    </div>
  );
}
