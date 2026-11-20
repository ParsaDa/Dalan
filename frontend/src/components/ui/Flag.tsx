import type { ReactNode } from 'react';

const H = (colors: string[]): ReactNode =>
  colors.map((c, i) => (
    <rect
      key={i}
      y={(24 / colors.length) * i}
      width="24"
      height={24 / colors.length + 0.1}
      fill={c}
    />
  ));

const V = (colors: string[]): ReactNode =>
  colors.map((c, i) => (
    <rect
      key={i}
      x={(24 / colors.length) * i}
      width={24 / colors.length + 0.1}
      height="24"
      fill={c}
    />
  ));

const FLAGS: Record<string, ReactNode> = {
  NL: H(['#AE1C28', '#fff', '#21468B']),
  DE: H(['#111', '#DD0000', '#FFCE00']),
  FR: V(['#0055A4', '#fff', '#EF4135']),
  AT: H(['#ED2939', '#fff', '#ED2939']),
  PL: H(['#fff', '#DC143C']),
  AR: H(['#74ACDF', '#fff', '#74ACDF']),
  JP: (
    <>
      <rect width="24" height="24" fill="#fff" />
      <circle cx="12" cy="12" r="5.5" fill="#BC002D" />
    </>
  ),
  CH: (
    <>
      <rect width="24" height="24" fill="#DA291C" />
      <path d="M10.5 5h3v5.5H19v3h-5.5V19h-3v-5.5H5v-3h5.5z" fill="#fff" />
    </>
  ),
  CA: (
    <>
      {V(['#D52B1E', '#fff', '#D52B1E'])}
      <path
        d="M12 6l1.6 3.2 2.4-.8-1 3.4 2.2 1-4 1.2v2.8h-2.4v-2.8l-4-1.2 2.2-1-1-3.4 2.4.8z"
        fill="#D52B1E"
      />
    </>
  ),
  SE: (
    <>
      <rect width="24" height="24" fill="#006AA7" />
      <rect x="7" width="3.5" height="24" fill="#FECC00" />
      <rect y="10.25" width="24" height="3.5" fill="#FECC00" />
    </>
  ),
  FI: (
    <>
      <rect width="24" height="24" fill="#fff" />
      <rect x="7" width="3.8" height="24" fill="#003580" />
      <rect y="10.1" width="24" height="3.8" fill="#003580" />
    </>
  ),
  TR: (
    <>
      <rect width="24" height="24" fill="#E30A17" />
      <circle cx="10" cy="12" r="5" fill="#fff" />
      <circle cx="11.4" cy="12" r="4" fill="#E30A17" />
      <circle cx="15.2" cy="12" r="1.3" fill="#fff" />
    </>
  ),
  GB: (
    <>
      <rect width="24" height="24" fill="#012169" />
      <path d="M0 0l24 24M24 0L0 24" stroke="#fff" strokeWidth="4" />
      <path d="M0 0l24 24M24 0L0 24" stroke="#C8102E" strokeWidth="1.6" />
      <path d="M12 0v24M0 12h24" stroke="#fff" strokeWidth="6" />
      <path d="M12 0v24M0 12h24" stroke="#C8102E" strokeWidth="3.4" />
    </>
  ),
  US: (
    <>
      {H(['#B22234', '#fff', '#B22234', '#fff', '#B22234', '#fff', '#B22234'])}
      <rect width="11" height="13" fill="#3C3B6E" />
    </>
  ),
  GR: (
    <>
      {H(['#0D5EAF', '#fff', '#0D5EAF', '#fff', '#0D5EAF', '#fff', '#0D5EAF', '#fff', '#0D5EAF'])}
      <rect width="10.7" height="10.7" fill="#0D5EAF" />
      <path d="M4.3 0h2.1v10.7H4.3zM0 4.3h10.7v2.1H0z" fill="#fff" />
    </>
  ),
  UY: (
    <>
      {H(['#fff', '#0038A8', '#fff', '#0038A8', '#fff', '#0038A8', '#fff', '#0038A8', '#fff'])}
      <rect width="10.7" height="12" fill="#fff" />
      <circle cx="5.3" cy="6" r="2.6" fill="#FCD116" />
    </>
  ),
};

export const FLAG_COUNTRIES = Object.keys(FLAGS);

interface FlagProps {
  country: string;
  size?: number;
}

// Round flags for the countries operators actually host nodes in; anything else gets a plain globe.
export default function Flag({ country, size = 20 }: FlagProps) {
  const code = country.trim().toUpperCase();
  const art = FLAGS[code];
  const id = `dl-flag-${code || 'x'}`;
  return (
    <svg
      className="dl-flag"
      width={size}
      height={size}
      viewBox="0 0 24 24"
      role="img"
      aria-label={code}
    >
      <clipPath id={id}>
        <circle cx="12" cy="12" r="12" />
      </clipPath>
      <g clipPath={`url(#${id})`}>
        {art ?? (
          <>
            <rect width="24" height="24" fill="var(--dl-track-hi)" />
            <path
              d="M0 12h24M12 0c5 5 5 19 0 24M12 0c-5 5-5 19 0 24"
              stroke="var(--dl-muted)"
              strokeWidth="1.2"
              fill="none"
            />
          </>
        )}
      </g>
      <circle
        cx="12"
        cy="12"
        r="11.6"
        fill="none"
        stroke="var(--dl-glass-border)"
        strokeWidth="0.8"
      />
    </svg>
  );
}
