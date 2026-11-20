import { fireEvent, render, screen } from '@testing-library/react';
import { expect, test, vi } from 'vitest';

import NodeTabs from '@/pages/index/NodeTabs';
import type { NodeRecord } from '@/api/queries/useNodesQuery';

const nodes = [
  { id: 1, name: 'nl1', country: 'NL', status: 'online', enable: true },
  { id: 2, name: 'de1', country: 'DE', status: 'offline', enable: true },
  { id: 3, name: 'parked', country: 'FR', status: 'online', enable: false },
] as NodeRecord[];

test('lists this server first, then each enabled node, and skips disabled ones', () => {
  render(<NodeTabs nodes={nodes} activeId={null} onSelect={() => {}} />);
  const tabs = screen.getAllByRole('tab').map((tab) => tab.textContent);
  expect(tabs).toEqual(['This server', 'nl1', 'de1']);
  expect(screen.getByRole('tab', { name: 'This server' }).getAttribute('aria-selected')).toBe(
    'true',
  );
});

test('picking a node reports its id and picking this server reports null', () => {
  const onSelect = vi.fn();
  render(<NodeTabs nodes={nodes} activeId={2} onSelect={onSelect} />);
  expect(screen.getByRole('tab', { name: 'de1' }).getAttribute('aria-selected')).toBe('true');

  fireEvent.click(screen.getByRole('tab', { name: 'nl1' }));
  fireEvent.click(screen.getByRole('tab', { name: 'This server' }));
  expect(onSelect.mock.calls).toEqual([[1], [null]]);
});
