import { act, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { useServerAccent } from '@/hooks/useServerAccent';
import { useTheme } from '@/hooks/useTheme';
import { HttpUtil, Msg } from '@/utils';
import { renderWithProviders } from './test-utils';

function Probe({ onTheme }: { onTheme: (theme: ReturnType<typeof useTheme>) => void }) {
  useServerAccent();
  onTheme(useTheme());
  return null;
}

function mount() {
  let current!: ReturnType<typeof useTheme>;
  renderWithProviders(<Probe onTheme={(next) => (current = next)} />);
  return () => current;
}

function serve(accent: string) {
  vi.spyOn(HttpUtil, 'post').mockResolvedValue(new Msg(true, '', { webPort: 2053, accent }));
}

beforeEach(() => {
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

describe('panel accent from the server', () => {
  it('adopts the accent the panel saved, so a new device or browser shows it too', async () => {
    serve('iris');
    const theme = mount();

    await waitFor(() => expect(theme().accent).toBe('iris'));
    expect(localStorage.getItem('dalan-accent')).toBe('iris');
  });

  it('ignores a saved value it does not know', async () => {
    serve('not-a-colour');
    const theme = mount();

    await waitFor(() => expect(vi.mocked(HttpUtil.post)).toHaveBeenCalled());
    await new Promise((r) => setTimeout(r, 30));
    expect(theme().accent).toBe('saffron');
  });

  it('does not overwrite a pick made here while the saved value has not changed', async () => {
    serve('iris');
    const theme = mount();
    await waitFor(() => expect(theme().accent).toBe('iris'));

    act(() => theme().setAccent('cobalt'));
    await new Promise((r) => setTimeout(r, 30));
    expect(theme().accent).toBe('cobalt');
  });
});
