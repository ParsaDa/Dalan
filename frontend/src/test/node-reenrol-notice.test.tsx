import { fireEvent, screen } from '@testing-library/react';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import ReenrolNotices from '@/pages/nodes/ReenrolNotices';
import type { NodeRecord } from '@/schemas/node';
import { HttpUtil, Msg } from '@/utils';

import { renderWithProviders } from './test-utils';

const node = (id: number, name: string): NodeRecord => ({
  id,
  name,
  guid: `g${id}`,
  transitive: false,
  enable: true,
  status: 'offline',
});

const enrolment = (id: number, name: string, nodeId: number, needsReenrol: boolean) => ({
  id,
  name,
  expiresAt: 1700000000,
  usedAt: 1700000100,
  tunnelId: 0,
  nodeId,
  dataTunnelId: 0,
  needsReenrol,
});

beforeEach(() => {
  vi.spyOn(HttpUtil, 'get').mockImplementation(
    async (url) =>
      new Msg(
        true,
        '',
        url === '/panel/api/enrolments/list'
          ? [enrolment(1, 'de1', 7, true), enrolment(2, 'nl1', 8, false)]
          : [],
      ),
  );
});

afterEach(() => {
  vi.restoreAllMocks();
});

function renderList() {
  renderWithProviders(<ReenrolNotices nodes={[node(7, 'de1'), node(8, 'nl1')]} />);
}

test('says which node must be enrolled again and opens the enrolment prefilled with its name', async () => {
  renderList();

  expect(await screen.findByText('de1 must be enrolled again')).toBeTruthy();
  expect(screen.queryByText('nl1 must be enrolled again')).toBeNull();

  fireEvent.click(screen.getByRole('button', { name: 'Enrol again' }));

  expect(((await screen.findByLabelText('Node name')) as HTMLInputElement).value).toBe('de1');
});
