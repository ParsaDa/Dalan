import { screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import Shell from '@/components/dl/shell/Shell';
import { HttpUtil, Msg } from '@/utils';
import { renderWithProviders } from './test-utils';

function openOnPhone(path: string) {
  renderWithProviders(
    <MemoryRouter initialEntries={[path]}>
      <Shell>
        <div className="pagehead" />
      </Shell>
    </MemoryRouter>,
  );
}

describe('phone top bar', () => {
  beforeEach(() => {
    const native = window.matchMedia;
    vi.spyOn(window, 'matchMedia').mockImplementation((q) => ({
      ...native(q),
      matches: q === '(max-width: 700px)',
    }));
    vi.spyOn(HttpUtil, 'get').mockResolvedValue(new Msg(true, '', []));
    vi.spyOn(HttpUtil, 'post').mockResolvedValue(new Msg(true, '', {}));
  });
  afterEach(() => vi.restoreAllMocks());

  it('names the page in the top bar instead of greeting, off the dashboard', () => {
    openOnPhone('/clients');
    expect(screen.getByRole('heading', { level: 1 }).textContent).toBe('Clients');
    expect(screen.queryByText(/^Good (morning|afternoon|evening)/)).toBeNull();
  });

  it('keeps the greeting on the dashboard', () => {
    openOnPhone('/');
    expect(screen.getByRole('heading', { level: 1 }).textContent).toMatch(
      /^Good (morning|afternoon|evening)|^Good night/,
    );
  });
});
