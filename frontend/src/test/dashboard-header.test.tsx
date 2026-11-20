import { render, screen } from '@testing-library/react';
import { expect, test } from 'vitest';

import DashboardHeader, { greetingKey } from '@/pages/index/DashboardHeader';

test.each([
  [4, 'pages.index.greetEvening'],
  [5, 'pages.index.greetMorning'],
  [11, 'pages.index.greetMorning'],
  [12, 'pages.index.greetAfternoon'],
  [17, 'pages.index.greetAfternoon'],
  [18, 'pages.index.greetEvening'],
])('at %i:00 the greeting is %s', (hour, key) => {
  expect(greetingKey(hour)).toBe(key);
});

test('the status line counts online nodes and is hidden when there are no nodes', () => {
  const { rerender } = render(<DashboardHeader online={2} total={3} hour={9} />);
  expect(screen.getByText('2 of 3 nodes online')).toBeTruthy();

  rerender(<DashboardHeader online={0} total={0} hour={9} />);
  expect(screen.queryByText(/nodes online/)).toBeNull();
});
