import { fireEvent, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';

import NodeFormModal from '@/pages/nodes/NodeFormModal';
import type { NodeRecord } from '@/api/queries/useNodesQuery';
import { HttpUtil, Msg } from '@/utils';
import { renderWithProviders } from './test-utils';

const amsterdam = {
  id: 7,
  name: 'nl1',
  scheme: 'http',
  address: '127.0.0.1',
  port: 61000,
  basePath: '/',
  hasApiToken: true,
  enable: true,
  allowPrivateAddress: true,
  country: 'NL',
  accent: 'tulip',
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
      node={amsterdam}
      testConnection={async () => new Msg(true, '', { status: 'online' })}
      fetchFingerprint={async () => new Msg(true, '', '')}
      fetchInbounds={async () => new Msg(true, '', [])}
      save={save}
      onOpenChange={() => {}}
    />,
  );
  return save;
}

test('saves a picked accent swatch with the stored country', async () => {
  const save = openEditor();
  fireEvent.click(await screen.findByRole('radio', { name: 'cobalt' }));
  fireEvent.click(screen.getByRole('button', { name: 'Save' }));

  await waitFor(() => expect(save).toHaveBeenCalled());
  expect(save).toHaveBeenCalledWith(expect.objectContaining({ country: 'NL', accent: 'cobalt' }));
});

test('auto accent clears the stored accent so the country decides', async () => {
  const save = openEditor();
  fireEvent.click(await screen.findByRole('radio', { name: 'Auto (from country)' }));
  fireEvent.click(screen.getByRole('button', { name: 'Save' }));

  await waitFor(() => expect(save).toHaveBeenCalled());
  expect(save).toHaveBeenCalledWith(expect.objectContaining({ country: 'NL', accent: '' }));
});

test('a custom colour on the success hue is refused and never saved', async () => {
  const save = openEditor();
  fireEvent.change(await screen.findByLabelText('Custom #rrggbb'), {
    target: { value: '#34d399' },
  });

  expect(await screen.findByText(/too close to the warning or success colour/)).toBeTruthy();
  fireEvent.click(screen.getByRole('button', { name: 'Save' }));
  await waitFor(() => expect(save).toHaveBeenCalled());
  expect(save).toHaveBeenCalledWith(expect.objectContaining({ accent: 'tulip' }));
});
