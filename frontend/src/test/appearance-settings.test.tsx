import { fireEvent, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import { ThemeProvider } from '@/hooks/useTheme';
import AppearanceSettings from '@/pages/settings/AppearanceSettings';

beforeEach(() => {
  vi.stubGlobal('matchMedia', (query: string) => ({
    matches: false,
    media: query,
    addEventListener: () => {},
    removeEventListener: () => {},
  }));
});

afterEach(() => {
  vi.unstubAllGlobals();
  localStorage.clear();
  delete document.body.dataset.effects;
  delete document.body.dataset.accent;
});

const mount = () =>
  render(
    <ThemeProvider>
      <AppearanceSettings />
    </ThemeProvider>,
  );

test('choosing an effects level applies it to the page and remembers it', () => {
  mount();
  fireEvent.click(screen.getByRole('radio', { name: /^Light/ }));

  expect(document.body.dataset.effects).toBe('light');
  expect(localStorage.getItem('dalan-effects')).toBe('light');
});

test('choosing a swatch changes the panel accent', () => {
  mount();
  fireEvent.click(screen.getByRole('radio', { name: 'cobalt' }));

  expect(document.body.dataset.accent).toBe('cobalt');
  expect(screen.getByRole('radio', { name: 'cobalt' }).getAttribute('aria-checked')).toBe('true');
});

test('the panel mode switch turns simple mode on and keeps it', () => {
  mount();
  fireEvent.click(screen.getByRole('switch', { name: 'Simple mode' }));

  expect(localStorage.getItem('dalan-simple-mode')).toBe('true');
  expect(screen.getByRole('switch', { name: 'Simple mode' }).getAttribute('aria-checked')).toBe(
    'true',
  );
});
