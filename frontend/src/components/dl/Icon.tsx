import { createElement } from 'react';

import { ICONS } from './icons';
import type { IconName } from './icons';

export type { IconName };

interface IconProps {
  name: IconName;
  size?: number;
  className?: string;
}

export default function Icon({ name, size, className }: IconProps) {
  return (
    <svg
      viewBox="0 0 24 24"
      width={size}
      height={size}
      className={className}
      fill="none"
      stroke="currentColor"
      strokeWidth="1.75"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {ICONS[name].map(([tag, attrs], i) => createElement(tag, { key: i, ...attrs }))}
    </svg>
  );
}
