import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { NodeCard } from '@/pages/nodes/NodesPage';
import { NodeRecordSchema } from '@/schemas/node';
import type { NodeRecord } from '@/schemas/node';
import { renderWithProviders } from './test-utils';

const noop = () => {};
function renderCard(node: NodeRecord, behind: NodeRecord[] = []) {
  renderWithProviders(
    <NodeCard
      node={node}
      behind={behind}
      traffic={0}
      latest=""
      onOpen={noop}
      onEdit={noop}
      onProbe={noop}
      onUpdate={noop}
      onHistory={noop}
      onToggle={noop}
      onDelete={noop}
    />,
  );
}
const base = {
  id: 1,
  name: 'edge',
  status: 'online',
  enable: true,
  cpuPct: 0,
  memPct: 44.4,
  uptimeSecs: 0,
};

describe('node card', () => {
  it('does not present an unavailable CPU as a measured zero', () => {
    renderCard(NodeRecordSchema.parse({ ...base, unavailableMetrics: ['cpu', 'uptime'] }));
    expect(screen.queryByText('0%')).toBeNull();
    expect(screen.getAllByText('–').length).toBeGreaterThanOrEqual(2);
  });

  it('keeps a measured zero CPU', () => {
    renderCard(NodeRecordSchema.parse(base));
    expect(screen.getByText('0%')).toBeTruthy();
  });

  it('shows the Xray error the node reported', () => {
    renderCard(
      NodeRecordSchema.parse({ ...base, xrayState: 'error', xrayError: 'config: bad port' }),
    );
    expect(screen.getByText('config: bad port')).toBeTruthy();
  });

  it('names the chained node, not the parent it is reached through', () => {
    const parent = NodeRecordSchema.parse({ ...base, guid: 'p1' });
    const child = NodeRecordSchema.parse({
      ...base,
      id: 0,
      name: 'helsinki',
      guid: 'c1',
      parentGuid: 'p1',
      transitive: true,
    });
    renderCard(parent, [child]);
    expect(screen.getByText('helsinki')).toBeTruthy();
  });
});
