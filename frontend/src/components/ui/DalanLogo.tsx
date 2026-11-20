import './DalanLogo.css';

interface DalanLogoProps {
  size?: number;
}

// Three nested arches; the innermost doorway takes the active accent.
export default function DalanLogo({ size = 28 }: DalanLogoProps) {
  return (
    <svg className="dl-logo" width={size} height={size} viewBox="0 0 32 32" fill="none" aria-hidden>
      <path className="dl-logo-a1" d="M3 29V15a13 13 0 0 1 26 0v14" />
      <path className="dl-logo-a2" d="M8 29V15.5a8 8 0 0 1 16 0V29" />
      <path className="dl-logo-door" d="M13 29v-13a3 3 0 0 1 6 0v13" />
    </svg>
  );
}
