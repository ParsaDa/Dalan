// The arch motif in four moods; inks are theme tokens so one drawing serves both themes.
const ROUND = { strokeLinecap: 'round' } as const;

export function IlloEmpty({ size = 150 }: { size?: number }) {
  return (
    <svg
      className="illo"
      viewBox="0 0 160 120"
      width={size}
      height={Math.round(size * 0.75)}
      aria-hidden="true"
    >
      <circle cx="112" cy="30" r="14" fill="var(--acc)" opacity=".9" />
      <path
        d="M24 112V64a56 56 0 0 1 112 0v48"
        fill="none"
        stroke="var(--tx)"
        strokeWidth="3"
        {...ROUND}
        opacity=".7"
      />
      <path
        d="M50 112V72a30 30 0 0 1 60 0v40"
        fill="none"
        stroke="var(--tx)"
        strokeWidth="3"
        {...ROUND}
        opacity=".35"
      />
      <path d="M68 112V84a12 12 0 0 1 24 0v28z" fill="var(--acc)" opacity=".85" />
      <path d="M10 112h140" stroke="var(--glass-b)" strokeWidth="2" {...ROUND} />
      <path
        d="M60 100h-30M130 100h-28"
        stroke="var(--mut)"
        strokeWidth="2"
        {...ROUND}
        strokeDasharray="2 6"
        opacity=".7"
      />
    </svg>
  );
}

export function IlloOffline({ size = 150 }: { size?: number }) {
  return (
    <svg
      className="illo"
      viewBox="0 0 160 120"
      width={size}
      height={Math.round(size * 0.75)}
      aria-hidden="true"
    >
      <path
        d="M20 112V62a60 60 0 0 1 120 0v50"
        fill="none"
        stroke="var(--mut)"
        strokeWidth="3"
        {...ROUND}
        opacity=".55"
      />
      <path
        d="M46 112V70a34 34 0 0 1 68 0v42"
        fill="none"
        stroke="var(--mut)"
        strokeWidth="3"
        {...ROUND}
        opacity=".35"
      />
      <path
        d="M66 112V80a14 14 0 0 1 28 0v32z"
        fill="var(--track)"
        stroke="var(--bad)"
        strokeWidth="2.5"
        strokeDasharray="4 5"
      />
      <circle cx="80" cy="36" r="10" fill="none" stroke="var(--bad)" strokeWidth="3" />
      <path d="m75 31 10 10M85 31 75 41" stroke="var(--bad)" strokeWidth="3" {...ROUND} />
      <path d="M8 112h144" stroke="var(--glass-b)" strokeWidth="2" {...ROUND} />
    </svg>
  );
}

export function IlloFirstRun({ size = 150 }: { size?: number }) {
  return (
    <svg viewBox="0 0 160 120" width={size} height={Math.round(size * 0.75)} aria-hidden="true">
      <circle cx="80" cy="30" r="16" fill="var(--acc)" />
      <path
        d="M30 112V70a50 50 0 0 1 100 0v42"
        fill="none"
        stroke="var(--tx)"
        strokeWidth="3"
        {...ROUND}
        opacity=".7"
      />
      <path
        d="M52 112V78a28 28 0 0 1 56 0v34"
        fill="none"
        stroke="var(--tx)"
        strokeWidth="3"
        {...ROUND}
        opacity=".4"
      />
      <path d="M10 112h140" stroke="var(--glass-b)" strokeWidth="2" {...ROUND} />
    </svg>
  );
}

const CORRIDOR = [60, 46, 34, 24, 16];
export function IlloCorridor({ size = 150 }: { size?: number }) {
  return (
    <svg viewBox="0 0 160 120" width={size} height={Math.round(size * 0.75)} aria-hidden="true">
      {CORRIDOR.map((w) => (
        <path
          key={w}
          d={`M${80 - w} 112V${Math.round(112 - w * 1.5)}a${w} ${w} 0 0 1 ${w * 2} 0V112`}
          fill="none"
          stroke="var(--tx)"
          strokeWidth={(1 + w / 20).toFixed(1)}
          {...ROUND}
          opacity={(0.15 + w / 80).toFixed(2)}
        />
      ))}
      <path d="M72 112V96a8 8 0 0 1 16 0v16z" fill="var(--acc)" />
      <path d="M10 112h140" stroke="var(--glass-b)" strokeWidth="2" {...ROUND} />
    </svg>
  );
}
