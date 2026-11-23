import { fireEvent, screen } from '@testing-library/react';
import { beforeAll, describe, expect, it, vi } from 'vitest';

import { ClipboardManager } from '@/utils';
import { renderWithProviders } from './test-utils';

const DAY = 86_400_000;

beforeAll(() => {
  window.__SUB_PAGE_DATA__ = {
    sId: 'abc123',
    subUrl: 'https://sub.example/abc123',
    subTitle: 'Pars Net',
    enabled: true,
    emails: ['sara.k'],
    totalByte: 50 * 1024 ** 3,
    usedByte: 34 * 1024 ** 3,
    expire: Math.floor((Date.now() + 18 * DAY - 3600_000) / 1000),
    links: ['vless://x@h:443#main'],
  } as never;
});

describe('subscription page', () => {
  it('greets the client and shows days left and usage against the limit', async () => {
    const { default: SubPage } = await import('@/pages/sub/SubPage');
    renderWithProviders(<SubPage />);
    expect(screen.getByRole('heading', { name: 'Hi sara.k' })).toBeTruthy();
    expect(screen.getByText('18')).toBeTruthy();
    expect(screen.getByText('34.00 GB')).toBeTruthy();
    expect(screen.getByText('used of 50.00 GB')).toBeTruthy();
  });

  it('copies the subscription link and says so', async () => {
    const copy = vi.spyOn(ClipboardManager, 'copyText').mockResolvedValue(true);
    const { default: SubPage } = await import('@/pages/sub/SubPage');
    renderWithProviders(<SubPage />);
    fireEvent.click(screen.getByRole('button', { name: 'Copy link' }));
    expect(copy).toHaveBeenCalledWith('https://sub.example/abc123');
    expect(await screen.findByRole('button', { name: 'Link copied' })).toBeTruthy();
  });
});
