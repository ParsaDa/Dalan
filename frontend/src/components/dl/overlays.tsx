import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import type { CSSProperties, ReactNode } from 'react';
import { createPortal } from 'react-dom';

import Icon from './Icon';
import type { Status } from './primitives';

function useEscape(onClose: () => void, active: boolean) {
  useEffect(() => {
    if (!active) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onClose, active]);
}

// Grows from the button that opened it: the origin comes from the element focused at open time.
export function Modal({
  open,
  onClose,
  label,
  children,
  width,
  top,
  style,
}: {
  open: boolean;
  onClose: () => void;
  label: string;
  children: ReactNode;
  width?: number;
  top?: boolean;
  style?: CSSProperties;
}) {
  const panel = useRef<HTMLDivElement>(null);
  useEscape(onClose, open);
  useLayoutEffect(() => {
    if (!open) return;
    const opener = document.activeElement as HTMLElement | null;
    const r = opener?.getBoundingClientRect();
    if (r && r.width && panel.current) {
      const x = Math.round(((r.left + r.width / 2) / window.innerWidth) * 100);
      panel.current.style.transformOrigin = `${x}% 0%`;
    }
    const t = window.setTimeout(
      () => panel.current?.querySelector<HTMLElement>('input,select,textarea,button')?.focus(),
      30,
    );
    return () => {
      window.clearTimeout(t);
      opener?.focus?.();
    };
  }, [open]);
  if (!open) return null;
  return createPortal(
    <div
      className="scrim"
      role="presentation"
      style={{ position: 'fixed', ...(top ? { alignItems: 'start', paddingTop: 90 } : null) }}
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div
        ref={panel}
        className="modal"
        role="dialog"
        aria-modal="true"
        aria-label={label}
        style={{ ...(width ? { width: `min(${width}px, calc(100% - 32px))` } : null), ...style }}
      >
        {children}
      </div>
    </div>,
    document.body,
  );
}

export function Sheet({
  open,
  onClose,
  label,
  children,
}: {
  open: boolean;
  onClose: () => void;
  label: string;
  children: ReactNode;
}) {
  useEscape(onClose, open);
  if (!open) return null;
  return createPortal(
    <>
      <div className="scrim" role="presentation" style={{ position: 'fixed' }} onClick={onClose} />
      <div
        className="sheet"
        style={{ position: 'fixed' }}
        role="dialog"
        aria-modal="true"
        aria-label={label}
      >
        <div className="grab" />
        {children}
      </div>
    </>,
    document.body,
  );
}

interface ToastState {
  id: number;
  text: string;
  status: Status;
}
const ToastContext = createContext<(text: string, status?: Status) => void>(() => undefined);
const TOAST_ICON = { ok: 'okc', warn: 'alert', bad: 'offc', off: 'info' } as const;
const TOAST_COLOR = {
  ok: 'var(--ok)',
  warn: 'var(--warn)',
  bad: 'var(--bad)',
  off: 'var(--mut)',
} as const;
const TOAST_MS = 2600;

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toast, setToast] = useState<ToastState | null>(null);
  const timer = useRef<number | undefined>(undefined);
  const show = useCallback((text: string, status: Status = 'ok') => {
    window.clearTimeout(timer.current);
    setToast({ id: Date.now(), text, status });
    timer.current = window.setTimeout(() => setToast(null), TOAST_MS);
  }, []);
  useEffect(() => () => window.clearTimeout(timer.current), []);
  const value = useMemo(() => show, [show]);
  return (
    <ToastContext.Provider value={value}>
      {children}
      {toast &&
        createPortal(
          <div key={toast.id} className="toast" role="status" style={{ position: 'fixed' }}>
            <span style={{ color: TOAST_COLOR[toast.status] }}>
              <Icon name={TOAST_ICON[toast.status]} />
            </span>
            {toast.text}
          </div>,
          document.body,
        )}
    </ToastContext.Provider>
  );
}

export function useToast() {
  return useContext(ToastContext);
}

// A dialog in the design's shape: title row with an optional icon, body, and the action row.
export function FormModal({
  open,
  onClose,
  title,
  subtitle,
  art,
  width = 600,
  children,
  footer,
}: {
  open: boolean;
  onClose: () => void;
  title: string;
  subtitle?: ReactNode;
  art?: ReactNode;
  width?: number;
  children: ReactNode;
  footer: ReactNode;
}) {
  return (
    <Modal open={open} onClose={onClose} label={title} width={width}>
      <div className="row sp">
        <div className="row">
          {art}
          <div>
            <h2 className="h2">{title}</h2>
            {subtitle && <div className="mut">{subtitle}</div>}
          </div>
        </div>
        <button type="button" className="ib sm" aria-label="Close" onClick={onClose}>
          <Icon name="x" />
        </button>
      </div>
      <div
        className="dl-form"
        style={{
          maxHeight: 'calc(100vh - 240px)',
          overflowY: 'auto',
          overflowX: 'hidden',
          paddingInlineEnd: 4,
        }}
      >
        {children}
      </div>
      <div className="mfoot">{footer}</div>
    </Modal>
  );
}
