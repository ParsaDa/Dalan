import { fireEvent, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import NodeFormModal from '@/pages/nodes/NodeFormModal';
import type { NodeRecord } from '@/api/queries/useNodesQuery';
import { HttpUtil, Msg } from '@/utils';
import { renderWithProviders } from './test-utils';

const dialledIn = {
  id: 4,
  name: 'de1',
  scheme: 'http',
  address: '127.0.0.1',
  publicAddress: '198.51.100.9',
  port: 61000,
  basePath: '/',
  hasApiToken: true,
  enable: true,
  allowPrivateAddress: true,
} as NodeRecord;

beforeEach(() => {
  vi.spyOn(HttpUtil, 'get').mockImplementation(async () => new Msg(true, '', {}));
});

afterEach(() => {
  vi.restoreAllMocks();
});

function openEditor() {
  const save = vi.fn(async () => new Msg(true, '', null));
  renderWithProviders(
    <NodeFormModal
      open
      mode="edit"
      node={dialledIn}
      testConnection={async () => new Msg(true, '', { status: 'online' })}
      fetchFingerprint={async () => new Msg(true, '', '')}
      fetchInbounds={async () => new Msg(true, '', [])}
      save={save}
      onOpenChange={() => {}}
    />,
  );
  return save;
}

test('shows the public address of a node that is reached on loopback', async () => {
  openEditor();
  const input = await screen.findByLabelText<HTMLInputElement>('Public address');
  expect(input.value).toBe('198.51.100.9');
});

test('saves a changed public address next to the address the panel reaches the node on', async () => {
  const save = openEditor();
  fireEvent.change(await screen.findByLabelText('Public address'), {
    target: { value: ' node.example.com ' },
  });
  fireEvent.click(screen.getByRole('button', { name: 'Save' }));

  await waitFor(() => expect(save).toHaveBeenCalled());
  expect(save).toHaveBeenCalledWith(
    expect.objectContaining({ address: '127.0.0.1', publicAddress: 'node.example.com' }),
  );
});
