import { fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { ConfirmDelete } from '@/components/dl/overlays';
import { renderWithProviders } from './test-utils';

describe('type-to-confirm delete', () => {
  it('keeps Delete locked until the word is typed, then deletes once', async () => {
    const onConfirm = vi.fn().mockResolvedValue(undefined);
    renderWithProviders(
      <ConfirmDelete open title="Delete 2 clients?" onCancel={() => {}} onConfirm={onConfirm}>
        nika.m, omid lose access right away.
      </ConfirmDelete>,
    );
    const del = screen.getByRole('button', { name: 'Delete' }) as HTMLButtonElement;
    const field = screen.getByLabelText('Type DELETE to confirm');
    fireEvent.change(field, { target: { value: 'delet' } });
    expect(del.disabled).toBe(true);
    fireEvent.click(del);
    expect(onConfirm).not.toHaveBeenCalled();
    fireEvent.change(field, { target: { value: 'DELETE' } });
    expect(del.disabled).toBe(false);
    fireEvent.click(del);
    await waitFor(() => expect(onConfirm).toHaveBeenCalledTimes(1));
  });

  it('forgets what was typed when cancelled', () => {
    const onCancel = vi.fn();
    const { rerender } = renderWithProviders(
      <ConfirmDelete open title="Delete?" onCancel={onCancel} onConfirm={() => {}} />,
    );
    fireEvent.change(screen.getByLabelText('Type DELETE to confirm'), {
      target: { value: 'DELETE' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(onCancel).toHaveBeenCalled();
    rerender(<ConfirmDelete open title="Delete?" onCancel={onCancel} onConfirm={() => {}} />);
    expect((screen.getByLabelText('Type DELETE to confirm') as HTMLInputElement).value).toBe('');
  });
});
