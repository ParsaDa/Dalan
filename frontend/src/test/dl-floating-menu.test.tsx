import { afterEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';

import { DropChip } from '@/components/dl/DropChip';
import { MoreMenu } from '@/components/dl/MoreMenu';

const actions = [
  { key: 'edit', label: 'Edit', onSelect: vi.fn() },
  { key: 'delete', label: 'Delete', danger: true, onSelect: vi.fn() },
];

// jsdom has no layout, so give the anchor a box and the open menu a height.
function layout(anchor: { top: number; bottom: number; left: number; right: number }) {
  vi.spyOn(window, 'innerHeight', 'get').mockReturnValue(768);
  vi.spyOn(window, 'innerWidth', 'get').mockReturnValue(1024);
  vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function (
    this: HTMLElement,
  ) {
    const isMenu = this.matches('[role="menu"], [role="listbox"]');
    const box = isMenu
      ? { top: 0, left: 0, width: 200, height: 160 }
      : {
          top: anchor.top,
          left: anchor.left,
          width: anchor.right - anchor.left,
          height: anchor.bottom - anchor.top,
        };
    return {
      ...box,
      right: box.left + box.width,
      bottom: box.top + box.height,
      x: box.left,
      y: box.top,
      toJSON: () => ({}),
    } as DOMRect;
  });
}

function Clipped({ children }: { children: React.ReactNode }) {
  return (
    <div data-testid="clip" style={{ overflow: 'hidden' }}>
      {children}
    </div>
  );
}

afterEach(() => vi.restoreAllMocks());

describe('floating menus', () => {
  it('open outside a clipping container, so a table or dialog cannot cut them off', () => {
    layout({ top: 100, bottom: 130, left: 800, right: 830 });
    render(
      <Clipped>
        <MoreMenu label="Row actions" actions={actions} />
      </Clipped>,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Row actions' }));
    const menu = screen.getByRole('menu');
    expect(screen.getByTestId('clip').contains(menu)).toBe(false);
    expect(menu.parentElement).toBe(document.body);
  });

  it('open upward when the row is at the bottom of the screen', () => {
    layout({ top: 720, bottom: 750, left: 800, right: 830 });
    render(<MoreMenu label="Row actions" actions={actions} />);
    fireEvent.click(screen.getByRole('button', { name: 'Row actions' }));
    const top = parseFloat(screen.getByRole('menu').style.top);
    expect(top + 160).toBeLessThanOrEqual(720);
  });

  it('open downward when there is room below', () => {
    layout({ top: 100, bottom: 130, left: 800, right: 830 });
    render(<MoreMenu label="Row actions" actions={actions} />);
    fireEvent.click(screen.getByRole('button', { name: 'Row actions' }));
    expect(parseFloat(screen.getByRole('menu').style.top)).toBeGreaterThanOrEqual(130);
  });

  it('stay inside the window on the right edge', () => {
    layout({ top: 100, bottom: 130, left: 990, right: 1020 });
    render(<MoreMenu label="Row actions" actions={actions} />);
    fireEvent.click(screen.getByRole('button', { name: 'Row actions' }));
    const left = parseFloat(screen.getByRole('menu').style.left);
    expect(left + 200).toBeLessThanOrEqual(1024);
    expect(left).toBeGreaterThanOrEqual(0);
  });

  it('keep working: an item click runs its action and closes the menu', () => {
    layout({ top: 100, bottom: 130, left: 800, right: 830 });
    const onSelect = vi.fn();
    render(<MoreMenu label="Row actions" actions={[{ key: 'x', label: 'Edit', onSelect }]} />);
    fireEvent.click(screen.getByRole('button', { name: 'Row actions' }));
    fireEvent.mouseDown(screen.getByRole('menuitem', { name: 'Edit' }));
    fireEvent.click(screen.getByRole('menuitem', { name: 'Edit' }));
    expect(onSelect).toHaveBeenCalledTimes(1);
    expect(screen.queryByRole('menu')).toBeNull();
  });

  it('close on a press outside, and not on a press inside the menu', () => {
    layout({ top: 100, bottom: 130, left: 800, right: 830 });
    render(<MoreMenu label="Row actions" actions={actions} />);
    fireEvent.click(screen.getByRole('button', { name: 'Row actions' }));
    fireEvent.mouseDown(screen.getByRole('menu'));
    expect(screen.queryByRole('menu')).not.toBeNull();
    fireEvent.mouseDown(document.body);
    expect(screen.queryByRole('menu')).toBeNull();
  });

  it('give a dropdown inside a dialog the same treatment', () => {
    layout({ top: 300, bottom: 330, left: 300, right: 500 });
    render(
      <Clipped>
        <DropChip
          label="Download on"
          value=""
          options={[
            { value: '', label: 'This server' },
            { value: '2', label: 'Node 2' },
          ]}
          onChange={() => {}}
        />
      </Clipped>,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Download on: This server' }));
    expect(screen.getByTestId('clip').contains(screen.getByRole('listbox'))).toBe(false);
  });
});
