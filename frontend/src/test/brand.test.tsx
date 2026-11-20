import { act, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { afterEach, expect, test, vi } from 'vitest';

import AppSidebar from '@/layouts/AppSidebar';
import { renderWithProviders } from './test-utils';

vi.mock('@/api/queries/useAllSettings', () => ({
  useAllSettings: () => ({ allSetting: {} }),
}));

afterEach(() => {
  localStorage.clear();
  delete window.X_UI_CUR_VER;
});

async function renderSidebar() {
  const view = renderWithProviders(
    <MemoryRouter>
      <AppSidebar />
    </MemoryRouter>,
  );
  await act(async () => {});
  return view;
}

test('shows the logo mark on the collapsed rail and adds the Dalan name when expanded', async () => {
  const view = await renderSidebar();
  const brand = () => view.container.querySelector('.brand-text')?.textContent;

  expect(view.container.querySelector('.sider-brand .dl-logo')).not.toBeNull();
  expect(brand()).toBeUndefined();

  fireEvent.mouseEnter(view.container.querySelector('.ant-sidebar')!);

  expect(view.container.querySelector('.sider-brand .dl-logo')).not.toBeNull();
  expect(brand()).toBe('Dalan');
});

test('links the version badge to the Dalan repository', async () => {
  window.X_UI_CUR_VER = '3.11.0';
  const view = await renderSidebar();
  const hrefs = Array.from(view.container.querySelectorAll('a')).map((a) => a.getAttribute('href'));

  expect(hrefs).toContain('https://github.com/ParsaDa/dalan');
  expect(hrefs).not.toContain('https://github.com/MHSanaei/3x-ui');
});
