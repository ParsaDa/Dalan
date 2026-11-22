import { describe, it, expect } from 'vitest';
import { screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router';

import { ActivityList } from '@/pages/index/dashboard/ActivityList';
import { AlertsTile } from '@/pages/index/dashboard/AlertsTile';
import { tunnelRow } from '@/pages/index/dashboard/tunnelRow';
import { renderWithProviders } from './test-utils';

const nodes = [
  { id: 3, name: 'Paris', country: 'FR' },
  { id: 4, name: 'Frankfurt', country: 'DE' },
];

describe('dashboard activity list', () => {
  it('writes each entry as a sentence with the actor', () => {
    const now = Math.floor(Date.now() / 1000);
    renderWithProviders(
      <MemoryRouter>
        <ActivityList
          nodes={nodes}
          items={[
            {
              id: 1,
              actor: 'you',
              action: 'client.add',
              target: 'sara.k',
              nodeId: 0,
              at: now - 120,
            },
            {
              id: 2,
              actor: 'system',
              action: 'node.offline',
              target: 'Paris',
              nodeId: 3,
              at: now - 3600,
            },
            { id: 3, actor: 'you', action: 'mystery.thing', target: 'x', nodeId: 0, at: now },
          ]}
        />
      </MemoryRouter>,
    );
    expect(screen.getByText(/added client/).closest('.f1')?.textContent).toBe(
      'You added client sara.k',
    );
    expect(screen.getByText(/went offline/).closest('.f1')?.textContent).toBe(
      'System: Paris went offline',
    );
    expect(screen.getByText(/mystery\.thing/)).toBeTruthy();
    expect(screen.getByText('2 min')).toBeTruthy();
  });
});

describe('dashboard alerts tile', () => {
  it('lists open alerts in words and counts them', () => {
    renderWithProviders(
      <MemoryRouter>
        <AlertsTile
          nodes={nodes}
          alerts={[
            {
              id: 1,
              kind: 'node.offline',
              subject: 'node:3',
              nodeId: 3,
              value: 0,
              startedAt: Math.floor(Date.now() / 1000) - 360,
              updatedAt: 0,
              resolvedAt: 0,
            },
            {
              id: 2,
              kind: 'clients.expiring',
              subject: 'local',
              nodeId: 0,
              value: 14,
              startedAt: 0,
              updatedAt: 0,
              resolvedAt: 0,
            },
          ]}
        />
      </MemoryRouter>,
    );
    expect(screen.getByText('Paris is offline')).toBeTruthy();
    expect(screen.getByText('14 users expire this week')).toBeTruthy();
    expect(screen.getByText('2 need attention')).toBeTruthy();
  });

  it('says all clear when nothing is open', () => {
    renderWithProviders(
      <MemoryRouter>
        <AlertsTile nodes={nodes} alerts={[]} />
      </MemoryRouter>,
    );
    expect(screen.getByText('Nothing needs attention')).toBeTruthy();
  });
});

describe('dashboard tunnel rows', () => {
  it('names the far node and spots layer 3', () => {
    const base = { id: 1, name: 't1', enable: true, role: 'portal', options: {}, nodeId: 4 };
    expect(tunnelRow({ ...base, engine: 'backpack', mode: 'reverse' }, nodes)).toMatchObject({
      to: 'Frankfurt',
      country: 'DE',
      layer3: false,
      mode: 'reverse',
    });
    expect(tunnelRow({ ...base, engine: 'backpack', mode: 'direct' }, nodes).layer3).toBe(true);
    expect(
      tunnelRow(
        { ...base, engine: 'rakhsh', mode: 'direct', options: { l3_addr: '10.0.0.1/30' } },
        nodes,
      ).layer3,
    ).toBe(true);
    expect(tunnelRow({ ...base, engine: 'rakhsh', mode: 'direct' }, nodes).layer3).toBe(false);
    expect(
      tunnelRow({ ...base, engine: 'rakhsh', mode: 'reverse', nodeId: 0, name: 'local-x' }, nodes)
        .to,
    ).toBe('local-x');
  });
});
