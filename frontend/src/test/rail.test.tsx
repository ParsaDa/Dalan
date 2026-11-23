import { fireEvent, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { afterEach, expect, test } from 'vitest';

import Rail from '@/components/dl/shell/Rail';
import { shellPrefs } from '@/components/dl/shell/shellPrefs';
import { renderWithProviders } from './test-utils';

afterEach(() => {
  shellPrefs.setSimple(false);
  shellPrefs.setSlim(false);
  localStorage.clear();
});

function renderRail(path = '/') {
  return renderWithProviders(
    <MemoryRouter initialEntries={[path]}>
      <Rail />
    </MemoryRouter>,
  );
}

test('groups the advanced menu and marks the open page', () => {
  renderRail('/clients');
  expect(screen.getByText('Users and access')).not.toBeNull();
  expect(screen.getByText('Network')).not.toBeNull();
  const clients = screen.getByRole('link', { name: /Clients/ });
  expect(clients.getAttribute('aria-current')).toBe('page');
  expect(screen.getByRole('link', { name: /Dashboard/ }).getAttribute('aria-current')).toBeNull();
});

// Simple mode is for non-technical admins: plain names, no raw Xray pages.
test('simple mode renames pages and hides the raw Xray ones', () => {
  shellPrefs.setSimple(true);
  renderRail();
  expect(screen.getByRole('link', { name: 'Users' })).not.toBeNull();
  expect(screen.getByRole('link', { name: 'Servers' })).not.toBeNull();
  expect(screen.queryByRole('link', { name: /Clients/ })).toBeNull();
  expect(screen.queryByRole('link', { name: 'Xray config' })).toBeNull();
  expect(screen.queryByText('Users and access')).toBeNull();
});

test('collapses to the icon rail and remembers it', () => {
  const view = renderRail();
  fireEvent.click(screen.getByRole('button', { name: 'Collapse menu' }));
  expect(view.container.querySelector('nav.rail')?.classList.contains('slim')).toBe(true);
  expect(localStorage.getItem('dalan-rail-slim')).toBe('true');
  expect(screen.getByRole('button', { name: 'Expand menu' })).not.toBeNull();
});
