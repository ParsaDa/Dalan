import { ACCENTS } from '@/lib/theme/accents';
import './AccentSwatches.css';

interface AccentSwatchesProps {
  value: string;
  onChange: (id: string) => void;
}

export default function AccentSwatches({ value, onChange }: AccentSwatchesProps) {
  return (
    <>
      {ACCENTS.map((accent) => (
        <button
          key={accent.id}
          type="button"
          role="radio"
          aria-checked={value === accent.id}
          aria-label={accent.id}
          className={`accent-swatch${value === accent.id ? ' on' : ''}`}
          style={{ background: accent.fill }}
          onClick={() => onChange(accent.id)}
        />
      ))}
    </>
  );
}
