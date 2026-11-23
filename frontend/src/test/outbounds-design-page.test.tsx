import { describe, expect, it, vi } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router';
import { fireEvent, screen, within } from '@testing-library/react';

import OutboundsTab from '@/pages/xray/outbounds/OutboundsTab';
import RoutingTab from '@/pages/xray/routing/RoutingTab';
import type { XraySettingsValue } from '@/hooks/useXraySetting';
import { renderWithProviders } from './test-utils';

function wrap(node: React.ReactNode) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return renderWithProviders(
    <QueryClientProvider client={qc}>
      <MemoryRouter>{node}</MemoryRouter>
    </QueryClientProvider>,
  );
}

function renderOutbounds(onResetTraffic = vi.fn(), onTestAll = vi.fn()) {
  wrap(
    <OutboundsTab
      templateSettings={
        { outbounds: [{ tag: 'direct', protocol: 'freedom' }] } as unknown as XraySettingsValue
      }
      setTemplateSettings={vi.fn()}
      outboundsTraffic={[]}
      outboundTestStates={{}}
      subscriptionTestStates={{}}
      testingAll={false}
      inboundTags={[]}
      isMobile={false}
      onResetTraffic={onResetTraffic}
      onTest={vi.fn()}
      onTestSubscription={vi.fn()}
      onTestAll={onTestAll}
      onShowWarp={vi.fn()}
      onShowNord={vi.fn()}
      onShowPia={vi.fn()}
      onShowTor={vi.fn()}
    />,
  );
}

describe('Outbounds and routing page', () => {
  it('resets every outbound traffic counter only after the dialog is confirmed', () => {
    const onResetTraffic = vi.fn();
    renderOutbounds(onResetTraffic);

    fireEvent.click(screen.getByRole('button', { name: 'Reset Traffic' }));
    expect(onResetTraffic).not.toHaveBeenCalled();
    fireEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Reset' }));
    expect(onResetTraffic).toHaveBeenCalledWith('-alltags-');
  });

  it('tests all outbounds with the chosen mode', () => {
    const onTestAll = vi.fn();
    renderOutbounds(vi.fn(), onTestAll);

    fireEvent.click(screen.getByRole('button', { name: 'HTTP' }));
    fireEvent.click(screen.getByRole('button', { name: 'Test all' }));
    expect(onTestAll).toHaveBeenCalledWith('http');
  });

  it('lists each rule with its number, what it matches and where it sends traffic', () => {
    wrap(
      <RoutingTab
        variant="tile"
        templateSettings={
          {
            routing: {
              rules: [
                {
                  type: 'field',
                  domain: ['geosite:category-ir'],
                  outboundTag: 'direct',
                  comment: 'Iranian sites',
                },
                { type: 'field', network: 'tcp,udp', outboundTag: 'tunnel-istanbul' },
              ],
            },
          } as unknown as XraySettingsValue
        }
        setTemplateSettings={vi.fn()}
        inboundTags={[]}
        clientReverseTags={[]}
        isMobile={false}
      />,
    );

    const rows = document.querySelectorAll<HTMLElement>('.rule-row');
    expect(rows).toHaveLength(2);
    expect(within(rows[0]).getByText('Iranian sites')).toBeTruthy();
    expect(within(rows[0]).getByText('domain geosite:category-ir')).toBeTruthy();
    expect(within(rows[0]).getByText('direct')).toBeTruthy();
    expect(rows[1].querySelector('.badge')?.textContent).toBe('2');
    expect(within(rows[1]).getByText('tunnel-istanbul')).toBeTruthy();
  });
});
