import { screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { describe, expect, it, vi } from 'vitest';

vi.mock('swagger-ui-react', () => ({ default: () => <div data-testid="swagger" /> }));

import ApiDocsPage from '@/pages/api-docs/ApiDocsPage';
import { HttpUtil, Msg } from '@/utils';
import { renderWithProviders } from './test-utils';

describe('Developer page', () => {
  it('lists the API tokens and loads the reference only when asked', async () => {
    vi.spyOn(HttpUtil, 'get').mockResolvedValue(
      new Msg(true, '', [
        { id: 1, name: 'billing-bot', enabled: true, scope: 'admin', expiresAt: 0 },
        { id: 2, name: 'grafana', enabled: false, scope: 'monitor', expiresAt: 0 },
      ]),
    );
    renderWithProviders(
      <MemoryRouter>
        <ApiDocsPage />
      </MemoryRouter>,
    );
    const bot = (await screen.findByText('billing-bot')).closest('.li') as HTMLElement;
    expect(bot.textContent).toContain('admin');
    expect(bot.textContent).toContain('Enabled');
    expect((screen.getByText('grafana').closest('.li') as HTMLElement).textContent).toContain(
      'Disabled',
    );

    expect(screen.queryByTestId('swagger')).toBeNull();
    screen.getByRole('button', { name: 'Load the API reference' }).click();
    expect(await screen.findByTestId('swagger')).toBeTruthy();
  });
});
