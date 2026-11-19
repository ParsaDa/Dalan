import { describe, it, expect, vi } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router';

import OutboundsTab from '@/pages/xray/outbounds/OutboundsTab';
import type { XraySettingsValue } from '@/hooks/useXraySetting';

import { renderWithProviders } from './test-utils';

function renderTab(isMobile: boolean) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return renderWithProviders(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <OutboundsTab
          templateSettings={
            { outbounds: [{ tag: 'proxy-a', protocol: 'vmess' }] } as unknown as XraySettingsValue
          }
          setTemplateSettings={vi.fn()}
          tunnelOutbounds={[{ tag: 'tunnel-de1', name: 'de1', engine: 'xray' }]}
          outboundsTraffic={[{ tag: 'tunnel-de1', up: 2048, down: 4096 }]}
          outboundTestStates={{}}
          subscriptionTestStates={{}}
          testingAll={false}
          inboundTags={[]}
          isMobile={isMobile}
          onResetTraffic={vi.fn()}
          onTest={vi.fn()}
          onTestSubscription={vi.fn()}
          onTestAll={vi.fn()}
          onShowWarp={vi.fn()}
          onShowNord={vi.fn()}
          onShowPia={vi.fn()}
          onShowTor={vi.fn()}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe('OutboundsTab tunnel rows', () => {
  it('lists a tunnel in the main table with its traffic, and offers no edit, delete or check', () => {
    renderTab(false);

    const tableRows = document.querySelectorAll('.ant-table-tbody tr.ant-table-row');
    expect(tableRows.length).toBe(2);
    const tunnelRow = tableRows[1] as HTMLElement;
    expect(tunnelRow.textContent).toContain('tunnel-de1');
    expect(tunnelRow.textContent).toContain('2.00 KB');
    expect(tunnelRow.textContent).toContain('Reverse');
    expect(tunnelRow.querySelector('button[aria-label="Edit"]')).toBeNull();
    expect(tunnelRow.querySelector('button[aria-label="More"]')).toBeNull();
    const check = tunnelRow.querySelector('button[aria-label="Check"]') as HTMLButtonElement;
    expect(check.disabled).toBe(true);
    expect(tunnelRow.querySelector('a[href$="/tunnels"]')).not.toBeNull();
  });

  it('does not render a separate tunnels table', () => {
    renderTab(false);
    expect(document.querySelectorAll('.ant-table-tbody').length).toBe(1);
  });

  it('lists the tunnel as a card on mobile without a menu to edit it', () => {
    renderTab(true);
    const cards = document.querySelectorAll('.outbound-card');
    expect(cards.length).toBe(2);
    expect(cards[1].textContent).toContain('tunnel-de1');
    expect(cards[1].textContent).toContain('Reverse');
    expect(cards[1].querySelector('button[aria-label="More"]')).toBeNull();
  });
});
