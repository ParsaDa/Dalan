import { describe, it, expect, vi } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router';
import { fireEvent } from '@testing-library/react';

import OutboundsTab from '@/pages/xray/outbounds/OutboundsTab';
import type { XraySettingsValue } from '@/hooks/useXraySetting';

import { renderWithProviders } from './test-utils';

const torOutbounds = [
  { tag: 'tor-de', country: 'de', enable: true },
  { tag: 'tor-nl', country: 'nl', enable: true },
  { tag: 'tor-fr', country: 'fr', enable: true },
  { tag: 'tor-gb', country: 'gb', enable: true },
];
const torStatus = {
  'tor-de': { state: 'connected', latencyMs: 1400, error: '', progress: 100 },
  'tor-nl': {
    state: 'failed',
    latencyMs: 0,
    error: 'socks connect: general failure',
    progress: 100,
  },
  'tor-fr': { state: 'starting', latencyMs: 0, error: '', progress: 40 },
  'tor-gb': { state: 'not_installed', latencyMs: 0, error: '', progress: 0 },
};

function renderTab(isMobile: boolean, onShowTor = vi.fn()) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  renderWithProviders(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <OutboundsTab
          templateSettings={
            { outbounds: [{ tag: 'proxy-a', protocol: 'vmess' }] } as unknown as XraySettingsValue
          }
          setTemplateSettings={vi.fn()}
          tunnelOutbounds={[{ tag: 'tunnel-de1', name: 'de1', engine: 'xray' }]}
          torOutbounds={torOutbounds}
          torStatus={torStatus}
          outboundsTraffic={[{ tag: 'tor-de', up: 2048, down: 4096 }]}
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
          onShowTor={onShowTor}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return onShowTor;
}

function rowFor(tag: string, selector: string) {
  const found = [...document.querySelectorAll<HTMLElement>(selector)].find((el) =>
    el.textContent?.includes(tag),
  );
  if (!found) throw new Error(`no ${selector} for ${tag}`);
  return found;
}

describe('OutboundsTab Tor rows', () => {
  it('lists every exit after the stored outbounds and tunnels with its country and traffic', () => {
    renderTab(false);

    const tableRows = document.querySelectorAll('tr.ob-row');
    expect(tableRows.length).toBe(6);
    const tags = [...tableRows].map((r) => r.querySelector('.tag-name')?.textContent);
    expect(tags).toEqual(['proxy-a', 'tunnel-de1', 'tor-de', 'tor-nl', 'tor-fr', 'tor-gb']);
    const de = rowFor('tor-de', 'tr.ob-row');
    expect(de.textContent).toContain('tor');
    expect(de.textContent).toContain('Germany');
    expect(de.textContent).toContain('2.00 KB');
  });

  it('says connected with the latency, or not connected with the reason, or starting with progress', () => {
    renderTab(false);

    const de = rowFor('tor-de', 'tr.ob-row').textContent;
    expect(de).toContain('Connected');
    expect(de).toContain('1400 ms');
    expect(de).not.toContain('Not connected');

    const nl = rowFor('tor-nl', 'tr.ob-row').textContent;
    expect(nl).toContain('Not connected');
    expect(nl).toContain('socks connect: general failure');

    const fr = rowFor('tor-fr', 'tr.ob-row').textContent;
    expect(fr).toContain('Starting');
    expect(fr).toContain('40%');
    expect(fr).not.toContain('Not connected');

    const gb = rowFor('tor-gb', 'tr.ob-row').textContent;
    expect(gb).toContain('Not connected');
    expect(gb).toContain('Tor is not installed on this server');
  });

  it('offers Manage Tor but no edit, menu or check, and Manage opens the Tor dialog', () => {
    const onShowTor = renderTab(false);
    const de = rowFor('tor-de', 'tr.ob-row');

    expect(de.querySelector('button[aria-label="Edit"]')).toBeNull();
    expect(de.querySelector('button[aria-label^="Actions"]')).toBeNull();
    expect((de.querySelector('button[aria-label="Check"]') as HTMLButtonElement).disabled).toBe(
      true,
    );
    fireEvent.click(de.querySelector('button[aria-label="Manage Tor"]') as HTMLElement);
    expect(onShowTor).toHaveBeenCalledTimes(1);
  });

  it('lists the exits as cards on mobile with the same status and a Manage button', () => {
    const onShowTor = renderTab(true);

    expect(document.querySelectorAll('.outbound-card').length).toBe(6);
    const nl = rowFor('tor-nl', '.outbound-card');
    expect(nl.textContent).toContain('Not connected');
    expect(nl.textContent).toContain('socks connect: general failure');
    expect(nl.textContent).toContain('Netherlands');
    const de = rowFor('tor-de', '.outbound-card');
    expect(de.textContent).toContain('1400 ms');
    expect(de.querySelector('button[aria-label^="Actions"]')).toBeNull();
    fireEvent.click(de.querySelector('button[aria-label="Manage Tor"]') as HTMLElement);
    expect(onShowTor).toHaveBeenCalledTimes(1);
  });
});
