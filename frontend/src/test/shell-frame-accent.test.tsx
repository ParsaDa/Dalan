import { useState } from 'react';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import Shell from '@/components/dl/shell/Shell';
import { HttpUtil, Msg } from '@/utils';
import { renderWithProviders } from './test-utils';

const nodes = [
  { id: 2, name: 'Frankfurt', enable: true, status: 'online', country: 'US', accent: 'iris' },
  { id: 3, name: 'Berlin', enable: true, status: 'online', country: 'DE', accent: '' },
];

function Page() {
  const [shown, setShown] = useState(true);
  return (
    <>
      <button type="button" onClick={() => setShown(false)}>
        leave
      </button>
      {shown && (
        <Shell>
          <div className="pagehead" />
        </Shell>
      )}
    </>
  );
}

function open(path: string) {
  return renderWithProviders(
    <MemoryRouter initialEntries={[path]}>
      <Page />
    </MemoryRouter>,
  );
}

const frameAccent = () => [...document.body.classList].find((c) => c.startsWith('ac-'));

describe('frame accent follows the open node', () => {
  beforeEach(() => {
    vi.spyOn(HttpUtil, 'get').mockImplementation(async (url: string) =>
      url === '/panel/api/nodes/list' ? new Msg(true, '', nodes) : new Msg(true, '', []),
    );
    vi.spyOn(HttpUtil, 'post').mockResolvedValue(new Msg(true, '', {}));
  });
  afterEach(() => {
    vi.restoreAllMocks();
    localStorage.clear();
  });

  it('uses the panel accent on this server', async () => {
    open('/');
    await waitFor(() => expect(frameAccent()).toBe('ac-saffron'));
    await new Promise((r) => setTimeout(r, 30));
    expect(frameAccent()).toBe('ac-saffron');
  });

  it('takes the colour set on the node whose tab is open', async () => {
    open('/?node=2');
    await waitFor(() => expect(frameAccent()).toBe('ac-iris'));
  });

  it('takes the colour of the node country when the node has none', async () => {
    open('/?node=3');
    await waitFor(() => expect(frameAccent()).toBe('ac-sun'));
  });

  it('gives the colour back when the page is left', async () => {
    open('/?node=2');
    await waitFor(() => expect(frameAccent()).toBe('ac-iris'));
    fireEvent.click(screen.getByRole('button', { name: 'leave' }));
    await waitFor(() => expect(frameAccent()).toBe('ac-saffron'));
  });
});
