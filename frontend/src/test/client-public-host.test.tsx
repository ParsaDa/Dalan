import type { ReactNode } from 'react';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClientProvider } from '@tanstack/react-query';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { useClients } from '@/hooks/useClients';
import { makeTestQueryClient } from '@/test/test-utils';
import { HttpUtil, Msg } from '@/utils';

afterEach(() => vi.restoreAllMocks());

async function publicHostFor(defaults: Record<string, unknown>) {
  vi.spyOn(HttpUtil, 'get').mockResolvedValue(new Msg(true, '', []));
  vi.spyOn(HttpUtil, 'post').mockImplementation(async (url: string) =>
    url.includes('/setting/defaultSettings') ? new Msg(true, '', defaults) : new Msg(true, '', []),
  );
  const queryClient = makeTestQueryClient();
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  const { result } = renderHook(() => useClients(), { wrapper });
  await waitFor(() => expect(result.current.settingsReady).toBe(true));
  return result.current.subSettings.publicHost;
}

describe('share-link host', () => {
  it('uses the address saved in the first-run wizard when no domain is set', async () => {
    expect(await publicHostFor({ serverAddress: '203.0.113.7' })).toBe('203.0.113.7');
  });

  it('prefers a subscription or panel domain over the wizard address', async () => {
    expect(
      await publicHostFor({ serverAddress: '203.0.113.7', subDomain: 'sub.example.com' }),
    ).toBe('sub.example.com');
    expect(
      await publicHostFor({ serverAddress: '203.0.113.7', webDomain: 'panel.example.com' }),
    ).toBe('panel.example.com');
  });
});
