import { act, render } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { ThemeProvider, useTheme } from '@/hooks/useTheme';

function Probe({ onTheme }: { onTheme: (theme: ReturnType<typeof useTheme>) => void }) {
  onTheme(useTheme());
  return null;
}

function mount() {
  let current!: ReturnType<typeof useTheme>;
  render(
    <ThemeProvider>
      <Probe onTheme={(theme) => (current = theme)} />
    </ThemeProvider>,
  );
  return () => current;
}

function stubReducedMotion(matches: boolean) {
  vi.stubGlobal('matchMedia', (query: string) => ({
    matches: query.includes('reduce') ? matches : false,
    media: query,
    addEventListener: () => {},
    removeEventListener: () => {},
  }));
}

afterEach(() => {
  vi.unstubAllGlobals();
  localStorage.clear();
  delete document.body.dataset.effects;
  delete document.body.dataset.accent;
});

describe('ThemeProvider effects and accent', () => {
  it('writes the chosen effects level and accent onto the body', () => {
    stubReducedMotion(false);
    const theme = mount();
    act(() => theme().setEffectsPref('light'));
    act(() => theme().setAccent('tulip'));
    expect(document.body.dataset.effects).toBe('light');
    expect(document.body.dataset.accent).toBe('tulip');
  });

  it('remembers the effects choice for the next visit', () => {
    stubReducedMotion(false);
    const theme = mount();
    act(() => theme().setEffectsPref('off'));
    expect(localStorage.getItem('dalan-effects')).toBe('off');
  });

  it('ignores a stored value it does not know', () => {
    stubReducedMotion(false);
    localStorage.setItem('dalan-effects', 'ultra');
    const theme = mount();
    expect(theme().effectsPref).toBe('auto');
  });

  it('shows Off while the system asks for reduced motion, even if Full was chosen', () => {
    stubReducedMotion(true);
    localStorage.setItem('dalan-effects', 'full');
    mount();
    expect(document.body.dataset.effects).toBe('off');
  });
});
