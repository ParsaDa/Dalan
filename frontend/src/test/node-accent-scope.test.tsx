import { render } from '@testing-library/react';
import { expect, test } from 'vitest';

import NodeAccentScope from '@/components/ui/NodeAccentScope';

const frame = (accent: string, country: string) =>
  render(
    <NodeAccentScope accent={accent} country={country}>
      <span />
    </NodeAccentScope>,
  ).container.querySelector<HTMLElement>('.dl-frame')!;

test('a node without an accent takes its country accent in its own frame', () => {
  expect(frame('', 'NL').dataset.accent).toBe('tulip');
});

test('a custom colour is handed to the frame as a variable, not as a preset name', () => {
  const el = frame('#4c8dff', 'NL');
  expect(el.dataset.accent).toBe('custom');
  expect(el.style.getPropertyValue('--dl-custom')).toBe('#4c8dff');
});
