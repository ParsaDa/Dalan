interface MarkProps {
  size?: number;
  // 'anim' draws the arches once; 'loop' breathes as a loader.
  motion?: 'anim' | 'loop';
  ink?: string;
  accent?: string;
  title?: string;
}

// Three nested arches (a dalan, a corridor) with the accent doorway and the light above it.
export default function Mark({
  size = 34,
  motion,
  ink = 'var(--tx)',
  accent = 'var(--acc)',
  title,
}: MarkProps) {
  return (
    <svg
      className={`mark${motion ? ` ${motion}` : ''}`}
      viewBox="0 0 48 48"
      width={size}
      height={size}
      role={title ? 'img' : undefined}
      aria-label={title}
      aria-hidden={title ? undefined : true}
    >
      <path className="a1" d="M7 43V22a17 17 0 0 1 34 0v21" stroke={ink} strokeWidth="4.2" />
      <path
        className="a2"
        d="M14.5 43V24a9.5 9.5 0 0 1 19 0v19"
        stroke={ink}
        strokeWidth="3.6"
        opacity=".55"
      />
      <path className="door" d="M20 43V27.5a4 4 0 0 1 8 0V43z" fill={accent} />
      <circle className="spk" cx="24" cy="12.5" r="2.2" fill={accent} />
    </svg>
  );
}
