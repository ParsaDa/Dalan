import { describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';

import { DropChip } from '@/components/dl/DropChip';

describe('DropChip', () => {
  const options = [
    { value: '', label: 'Anyone' },
    { value: 'you', label: 'You' },
    { value: 'system', label: 'System' },
  ];

  it('shows the chosen option and reports a new pick', () => {
    const onChange = vi.fn();
    render(<DropChip label="Who" value="" options={options} onChange={onChange} />);
    const chip = screen.getByRole('button', { name: 'Who: Anyone' });
    expect(chip.getAttribute('aria-expanded')).toBe('false');
    fireEvent.click(chip);
    fireEvent.click(screen.getByRole('option', { name: 'System' }));
    expect(onChange).toHaveBeenCalledWith('system');
    expect(screen.queryByRole('listbox')).toBeNull();
  });

  it('closes on Escape without changing the value', () => {
    const onChange = vi.fn();
    render(<DropChip label="Who" value="you" options={options} onChange={onChange} />);
    fireEvent.click(screen.getByRole('button', { name: 'Who: You' }));
    fireEvent.keyDown(screen.getByRole('listbox'), { key: 'Escape' });
    expect(screen.queryByRole('listbox')).toBeNull();
    expect(onChange).not.toHaveBeenCalled();
  });
});
