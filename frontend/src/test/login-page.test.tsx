import { fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import LoginPage from '@/pages/login/LoginPage';
import { HttpUtil, Msg } from '@/utils';
import { renderWithProviders } from './test-utils';

function mockServer(twoFactor: boolean) {
  return vi
    .spyOn(HttpUtil, 'post')
    .mockImplementation(async (url) =>
      url === '/getTwoFactorEnable'
        ? new Msg(true, '', twoFactor)
        : new Msg(false, 'wrong password', null),
    );
}

describe('login page', () => {
  it('asks for the two-step code only when the panel uses it', async () => {
    mockServer(true);
    renderWithProviders(<LoginPage />);
    expect(await screen.findByLabelText('Code')).toBeTruthy();
  });

  it('sends the username and password it was given', async () => {
    const post = mockServer(false);
    renderWithProviders(<LoginPage />);
    fireEvent.change(await screen.findByLabelText('Username'), { target: { value: 'parsa' } });
    fireEvent.change(screen.getByLabelText('Password'), { target: { value: 'secretsecret' } });
    fireEvent.click(screen.getByRole('button', { name: 'Log In' }));
    await waitFor(() =>
      expect(post).toHaveBeenCalledWith(
        '/login',
        expect.objectContaining({ username: 'parsa', password: 'secretsecret' }),
      ),
    );
  });

  it('does not send an empty form', async () => {
    const post = mockServer(false);
    renderWithProviders(<LoginPage />);
    fireEvent.click(await screen.findByRole('button', { name: 'Log In' }));
    await waitFor(() => expect(screen.getAllByText(/./, { selector: 'small' }).length).toBe(2));
    expect(post).not.toHaveBeenCalledWith('/login', expect.anything());
  });
});
