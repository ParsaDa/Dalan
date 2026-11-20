import { render } from '@testing-library/react';
import { expect, test } from 'vitest';

import Flag, { FLAG_COUNTRIES } from '@/components/ui/Flag';

test('every country in the flag list draws its own art, not the globe fallback', () => {
  const globe = render(<Flag country="ZZ" />).container.querySelector('g')?.innerHTML;
  for (const code of FLAG_COUNTRIES) {
    const drawn = render(<Flag country={code.toLowerCase()} />).container.querySelector(
      'g',
    )?.innerHTML;
    expect(drawn, code).not.toBe(globe);
  }
});

test('an unknown country is labelled with its code and falls back to the globe', () => {
  const { getByRole } = render(<Flag country=" zz " />);
  expect(getByRole('img').getAttribute('aria-label')).toBe('ZZ');
});
