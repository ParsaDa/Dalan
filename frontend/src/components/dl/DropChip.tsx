import { useEffect, useId, useRef, useState } from 'react';

import Icon from './Icon';
import type { IconName } from './Icon';

export interface DropOption {
  value: string;
  label: string;
}

export function DropChip({
  label,
  value,
  options,
  onChange,
  icon,
}: {
  label: string;
  value: string;
  options: readonly DropOption[];
  onChange: (value: string) => void;
  icon?: IconName;
}) {
  const [open, setOpen] = useState(false);
  const box = useRef<HTMLDivElement>(null);
  const listId = useId();
  const current = options.find((o) => o.value === value) ?? options[0];
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
        className={`chip${value ? ' on' : ''}`}
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-controls={listId}
        onClick={() => setOpen((o) => !o)}
      >
        {icon && <Icon name={icon} />}
        {`${label}: ${current?.label ?? ''}`}
        <Icon name="chevd" />
      </button>
      {open && (
        <div
          className="dropmenu"
          role="listbox"
          id={listId}
          aria-label={label}
          tabIndex={-1}
          onKeyDown={(e) => {
            if (e.key === 'Escape') setOpen(false);
          }}
        >
          {options.map((o) => (
            <button
              key={o.value}
              type="button"
              role="option"
              aria-selected={o.value === value}
              className={`nav${o.value === value ? ' on' : ''}`}
              onClick={() => {
                setOpen(false);
                if (o.value !== value) onChange(o.value);
              }}
            >
              <span className="f1">{o.label}</span>
              {o.value === value && <Icon name="check" />}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
