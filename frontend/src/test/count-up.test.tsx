import { act, render } from '@testing-library/react';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import CountUp from '@/components/ui/CountUp';
import { ThemeProvider } from '@/hooks/useTheme';

function stubMotion(reduce: boolean) {
  vi.stubGlobal('matchMedia', (query: string) => ({
    matches: query.includes('reduce') ? reduce : false,
    media: query,
    addEventListener: () => {},
    removeEventListener: () => {},
  }));
}

const show = (value: number) =>
  render(
    <ThemeProvider>
      <CountUp value={value} format={(v) => v.toFixed(0)} />
    </ThemeProvider>,
  );

beforeEach(() => {
  vi.useFakeTimers({ toFake: ['requestAnimationFrame', 'cancelAnimationFrame', 'performance'] });
  localStorage.clear();
});

afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
  localStorage.clear();
});

test('counts up from zero and lands exactly on the value after 700 ms', () => {
  stubMotion(false);
  const view = show(80);
  expect(view.container.querySelector('[aria-hidden]')?.textContent).toBe('0');

  act(() => {
    vi.advanceTimersByTime(350);
  });
  const mid = Number(view.container.querySelector('[aria-hidden]')?.textContent);
  expect(mid).toBeGreaterThan(0);
  expect(mid).toBeLessThan(80);

  act(() => {
    vi.advanceTimersByTime(400);
  });
  expect(view.container.querySelector('[aria-hidden]')?.textContent).toBe('80');
});

test('assistive tech gets the final value from the first frame, not the animated one', () => {
  stubMotion(false);
  const view = show(80);
  expect(view.container.querySelector('.dl-sr')?.textContent).toBe('80');
  expect(view.container.querySelector('[aria-hidden]')?.textContent).toBe('0');
});

test('shows the value at once when motion is reduced', () => {
  stubMotion(true);
  expect(show(80).container.querySelector('[aria-hidden]')?.textContent).toBe('80');
});

test('an updated value animates from the one on screen, not from zero', () => {
  stubMotion(false);
  const view = show(80);
  act(() => {
    vi.advanceTimersByTime(800);
  });
  view.rerender(
    <ThemeProvider>
      <CountUp value={40} format={(v) => v.toFixed(0)} />
    </ThemeProvider>,
  );
  act(() => {
    vi.advanceTimersByTime(100);
  });
  expect(Number(view.container.querySelector('[aria-hidden]')?.textContent)).toBeGreaterThan(40);
});
