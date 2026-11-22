import { afterEach, describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import i18next from 'i18next';

import { Meter } from '@/components/dl/primitives';

// The Persian artboards show Persian digits; the panel printed Latin ones beside Persian text.
describe('dl numbers follow the panel language', () => {
  afterEach(() => i18next.changeLanguage('en-US'));

  it('prints a meter percentage in Persian digits for fa-IR', async () => {
    await i18next.changeLanguage('fa-IR');
    render(<Meter label="CPU" value={0.6} />);
    expect(screen.getByText('۶۰٪')).toBeTruthy();
  });

  it('keeps Latin digits for English', () => {
    render(<Meter label="CPU" value={0.6} />);
    expect(screen.getByText('60%')).toBeTruthy();
  });
});
