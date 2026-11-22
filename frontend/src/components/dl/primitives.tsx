import type { ButtonHTMLAttributes, CSSProperties, ReactNode } from 'react';
import { useId } from 'react';

import Flag from '@/components/ui/Flag';
import Icon from './Icon';
import { useNum } from './numbers';
import type { IconName } from './Icon';

export type Status = 'ok' | 'warn' | 'bad' | 'off';
const STATUS_ICON: Record<Status, IconName> = {
  ok: 'okc',
  warn: 'alert',
  bad: 'offc',
  off: 'clock',
};

export function Pill({
  status,
  tone,
  icon,
  children,
  flash,
  style,
}: {
  status?: Status;
  tone?: 'acc' | 'neutral';
  icon?: IconName;
  children: ReactNode;
  flash?: boolean;
  style?: CSSProperties;
}) {
  const kind = status ?? tone ?? '';
  const glyph = icon ?? (status ? STATUS_ICON[status] : undefined);
  return (
    <span className={`pill ${kind}${flash ? ' flash' : ''}`} style={style}>
      {glyph && <Icon name={glyph} />}
      {children}
    </span>
  );
}

export function Badge({ acc, children }: { acc?: boolean; children: ReactNode }) {
  return <span className={`badge${acc ? ' acc' : ''}`}>{children}</span>;
}

export function Bar({
  value,
  tone,
  large,
}: {
  value: number;
  tone?: 'warn' | 'bad';
  large?: boolean;
}) {
  const v = Math.max(0, Math.min(1, Number.isFinite(value) ? value : 0));
  return (
    <div className={`bar${tone ? ` ${tone}` : ''}${large ? ' lg' : ''}`}>
      <i style={{ transform: `scaleX(${v.toFixed(3)})` }} />
    </div>
  );
}

export const HIGH_USAGE = 0.85;

export function Meter({
  label,
  value,
  shown,
  note,
  highLabel,
  small,
}: {
  label: ReactNode;
  value: number;
  shown?: string;
  note?: ReactNode;
  highLabel?: string;
  small?: boolean;
}) {
  const high = value >= HIGH_USAGE;
  const num = useNum();
  return (
    <div className="meter">
      <div className="row" style={small ? { fontSize: 12.5 } : undefined}>
        <span>{label}</span>
        {high && highLabel && (
          <Pill status="warn" style={{ height: 20, padding: '0 7px' }}>
            {highLabel}
          </Pill>
        )}
        <span className="v num">{shown ?? num.pct(value)}</span>
      </div>
      <Bar value={value} tone={high ? 'warn' : undefined} />
      {note && (
        <small className="mut" style={{ fontSize: 12 }}>
          {note}
        </small>
      )}
    </div>
  );
}

export function Toggle({
  on,
  label,
  busy,
  onChange,
  disabled,
}: {
  on: boolean;
  label: string;
  busy?: boolean;
  onChange?: (next: boolean) => void;
  disabled?: boolean;
}) {
  return (
    <button
      type="button"
      className={`tg${busy ? ' busy' : ''}`}
      role="switch"
      aria-checked={on}
      aria-label={label}
      disabled={disabled || busy}
      onClick={() => onChange?.(!on)}
    />
  );
}

type BtnKind = 'pri' | 'ghost' | 'dng';
export function Btn({
  kind,
  size,
  icon,
  children,
  className,
  ...rest
}: ButtonHTMLAttributes<HTMLButtonElement> & {
  kind?: BtnKind;
  size?: 'sm' | 'lg';
  icon?: IconName;
}) {
  const cls = ['btn', kind, size, className].filter(Boolean).join(' ');
  return (
    <button type="button" className={cls} {...rest}>
      {icon && <Icon name={icon} />}
      {children}
    </button>
  );
}

export function IconBtn({
  icon,
  label,
  small,
  className,
  ...rest
}: ButtonHTMLAttributes<HTMLButtonElement> & { icon: IconName; label: string; small?: boolean }) {
  return (
    <button
      type="button"
      className={`ib${small ? ' sm' : ''}${className ? ` ${className}` : ''}`}
      aria-label={label}
      title={label}
      {...rest}
    >
      <Icon name={icon} />
    </button>
  );
}

export function Seg<T extends string>({
  value,
  options,
  onChange,
  label,
  lead,
}: {
  value: T;
  options: readonly { value: T; label: ReactNode }[];
  onChange: (v: T) => void;
  label: string;
  lead?: ReactNode;
}) {
  return (
    <div className="seg" role="group" aria-label={label}>
      {lead}
      {options.map((o) => (
        <button
          key={o.value}
          type="button"
          className={o.value === value ? 'on' : ''}
          aria-pressed={o.value === value}
          onClick={() => onChange(o.value)}
        >
          {o.label}
        </button>
      ))}
    </div>
  );
}

export function Chip({
  on,
  count,
  icon,
  children,
  onClick,
}: {
  on?: boolean;
  count?: ReactNode;
  icon?: IconName;
  children: ReactNode;
  onClick?: () => void;
}) {
  return (
    <button
      type="button"
      className={`chip${on ? ' on' : ''}`}
      aria-pressed={!!on}
      onClick={onClick}
    >
      {icon && <Icon name={icon} />}
      {children}
      {count !== undefined && <span className="c num">{count}</span>}
    </button>
  );
}

export function TileHead({
  icon,
  title,
  right,
}: {
  icon?: IconName;
  title: ReactNode;
  right?: ReactNode;
}) {
  return (
    <div className="th">
      <span className="tt">
        {icon && <Icon name={icon} />}
        {title}
      </span>
      {right}
    </div>
  );
}

export function FlagDot({ country, size = 28 }: { country?: string; size?: number }) {
  return (
    <span
      className="fl"
      style={{
        width: size,
        height: size,
        borderRadius: '50%',
        overflow: 'hidden',
        display: 'grid',
        placeItems: 'center',
        flex: 'none',
      }}
    >
      {country ? <Flag country={country} size={size} /> : <Icon name="server" size={size * 0.62} />}
    </span>
  );
}

export function Sk({
  w,
  h,
  r,
  style,
}: {
  w?: number | string;
  h: number;
  r?: number;
  style?: CSSProperties;
}) {
  return <div className="sk" style={{ width: w, height: h, borderRadius: r, ...style }} />;
}

export function Empty({
  art,
  title,
  children,
  actions,
}: {
  art?: ReactNode;
  title: ReactNode;
  children?: ReactNode;
  actions?: ReactNode;
}) {
  return (
    <div className="empty">
      {art}
      <h3 className="h3">{title}</h3>
      {children && <p style={{ fontSize: 13 }}>{children}</p>}
      {actions && (
        <div className="row wrap" style={{ justifyContent: 'center' }}>
          {actions}
        </div>
      )}
    </div>
  );
}

export function sparkPath(values: readonly number[], w: number, h: number, pad = 4) {
  const vals = values.length > 1 ? values : [values[0] ?? 0, values[0] ?? 0];
  const mx = Math.max(...vals);
  const mn = Math.min(...vals);
  const rng = mx - mn || 1;
  const step = (w - pad * 2) / (vals.length - 1);
  const pts = vals.map(
    (v, i) => [pad + i * step, pad + (h - pad * 2) * (1 - (v - mn) / rng)] as const,
  );
  const line = 'M' + pts.map(([x, y]) => `${x.toFixed(1)} ${y.toFixed(1)}`).join(' L');
  const last = pts[pts.length - 1];
  const area = `${line} L${last[0].toFixed(1)} ${h} L${pts[0][0].toFixed(1)} ${h} Z`;
  return { line, area, last };
}

export function Spark({
  values,
  w = 300,
  h = 64,
  compare,
}: {
  values: readonly number[];
  w?: number;
  h?: number;
  compare?: readonly number[];
}) {
  const gid = useId().replace(/:/g, '');
  const { line, area, last } = sparkPath(values, w, h);
  return (
    <svg
      className="spark"
      viewBox={`0 0 ${w} ${h}`}
      height={h}
      preserveAspectRatio="none"
      aria-hidden="true"
    >
      <defs>
        <linearGradient id={gid} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" style={{ stopColor: 'var(--acc)', stopOpacity: 0.28 }} />
          <stop offset="1" style={{ stopColor: 'var(--acc)', stopOpacity: 0 }} />
        </linearGradient>
      </defs>
      <path className="ar" d={area} fill={`url(#${gid})`} />
      {compare && <path className="ln2" d={sparkPath(compare, w, h).line} />}
      <path className="ln" d={line} />
      <circle className="hd" cx={last[0].toFixed(1)} cy={last[1].toFixed(1)} r="4" />
    </svg>
  );
}

export function Ring({
  value,
  size = 150,
  children,
}: {
  value: number;
  size?: number;
  children?: ReactNode;
}) {
  const c = 314.2;
  const v = Math.max(0, Math.min(1, value));
  return (
    <div style={{ position: 'relative', width: size, height: size, flex: 'none' }}>
      <svg className="ring" viewBox="0 0 120 120" width={size} height={size} aria-hidden="true">
        <circle className="tr" cx="60" cy="60" r="50" />
        <circle
          className="vl"
          cx="60"
          cy="60"
          r="50"
          style={{ strokeDasharray: c, strokeDashoffset: c * (1 - v) }}
        />
      </svg>
      <div
        style={{
          position: 'absolute',
          inset: 0,
          display: 'grid',
          placeItems: 'center',
          textAlign: 'center',
        }}
      >
        {children}
      </div>
    </div>
  );
}
