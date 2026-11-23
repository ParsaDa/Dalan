import { screen } from '@testing-library/react';
import { createMemoryRouter, RouterProvider } from 'react-router';
import { describe, expect, it } from 'vitest';

import ErrorPage from '@/pages/errors/ErrorPage';
import { renderWithProviders } from './test-utils';

function Boom(): never {
  throw new Error('render failed');
}

describe('error page', () => {
  it('says the page does not exist for an unknown address', () => {
    const router = createMemoryRouter(
      [
        { path: '/', element: <div /> },
        { path: '*', element: <ErrorPage notFound /> },
      ],
      { initialEntries: ['/nowhere'] },
    );
    renderWithProviders(<RouterProvider router={router} />);
    expect(screen.getByRole('heading', { name: 'Lost in the corridor' })).toBeTruthy();
    expect(screen.getByText('error 404')).toBeTruthy();
  });

  it('shows a crash with its message and a way back', () => {
    const router = createMemoryRouter([
      { path: '/', element: <Boom />, errorElement: <ErrorPage /> },
    ]);
    renderWithProviders(<RouterProvider router={router} />);
    expect(screen.getByRole('heading', { name: 'This page stopped working' })).toBeTruthy();
    expect(screen.getByText('error 500 · render failed')).toBeTruthy();
    expect(screen.getByRole('link', { name: 'Go to dashboard' }).getAttribute('href')).toBe('/');
  });
});
