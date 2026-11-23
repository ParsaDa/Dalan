import { describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';

import { MoreMenu } from '@/components/dl/MoreMenu';

describe('MoreMenu', () => {
  it('runs the picked action once and closes', () => {
    const del = vi.fn();
    const probe = vi.fn();
    render(
      <MoreMenu
        label="More actions"
        actions={[
          { key: 'probe', label: 'Test connection', onSelect: probe },
          { key: 'del', label: 'Delete', danger: true, onSelect: del },
        ]}
      />,
    );
    fireEvent.click(screen.getByRole('button', { name: 'More actions' }));
    fireEvent.click(screen.getByRole('menuitem', { name: 'Delete' }));
    expect(del).toHaveBeenCalledTimes(1);
    expect(probe).not.toHaveBeenCalled();
    expect(screen.queryByRole('menu')).toBeNull();
  });

  it('does not run a disabled action', () => {
    const upd = vi.fn();
    render(
      <MoreMenu
        label="More"
        actions={[{ key: 'u', label: 'Update panel', disabled: true, onSelect: upd }]}
      />,
    );
    fireEvent.click(screen.getByRole('button', { name: 'More' }));
    fireEvent.click(screen.getByRole('menuitem', { name: 'Update panel' }));
    expect(upd).not.toHaveBeenCalled();
  });
});
