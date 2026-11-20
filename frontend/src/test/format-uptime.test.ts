import { expect, test } from 'vitest';

import { formatUptime } from '@/lib/format-uptime';

test.each([
  [90_061, 'days and hours', '1d 1h'],
  [7_500, 'hours and minutes', '2h 5m'],
  [300, 'minutes only', '5m'],
])('%i seconds reads as %s', (secs, _label, want) => {
  expect(formatUptime(secs)).toBe(want);
});

test('an unavailable metric is a dash, distinct from the plain dash for zero', () => {
  expect(formatUptime(90_061, false)).toBe('—');
  expect(formatUptime(0)).toBe('-');
});
