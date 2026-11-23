import { fireEvent, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import AppearanceSettings from '@/pages/settings/AppearanceSettings';
import { HttpUtil, Msg } from '@/utils';
import { renderWithProviders } from './test-utils';

beforeEach(() => {
  vi.spyOn(HttpUtil, 'post').mockResolvedValue(new Msg(true, '', null));
  vi.stubGlobal('matchMedia', (query: string) => ({
    matches: false,
    media: query,
    addEventListener: () => {},
    removeEventListener: () => {},
  }));
});

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  localStorage.clear();
  delete document.body.dataset.effects;
  delete document.body.dataset.accent;
});

const mount = () => renderWithProviders(<AppearanceSettings />);

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

test('a picked accent is saved on the server and kept on this device across a reload', async () => {
  const first = mount();
  fireEvent.click(screen.getByRole('radio', { name: 'cobalt' }));

  await waitFor(() =>
    expect(HttpUtil.post).toHaveBeenCalledWith('/panel/api/setting/profile', { accent: 'cobalt' }),
  );
  expect(localStorage.getItem('dalan-accent')).toBe('cobalt');

  first.unmount();
  delete document.body.dataset.accent;
  mount();
  expect(document.body.dataset.accent).toBe('cobalt');
});

test('the panel mode switch turns simple mode on and keeps it', () => {
  mount();
  fireEvent.click(screen.getByRole('switch', { name: 'Simple mode' }));

  expect(localStorage.getItem('dalan-simple-mode')).toBe('true');
  expect(screen.getByRole('switch', { name: 'Simple mode' }).getAttribute('aria-checked')).toBe(
    'true',
  );
});
