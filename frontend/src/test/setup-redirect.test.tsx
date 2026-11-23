import { renderHook, waitFor } from '@testing-library/react';
import { QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { useNeedsSetup } from '@/pages/setup/useNeedsSetup';
import { HttpUtil, Msg } from '@/utils';
import { makeTestQueryClient } from './test-utils';

function serve(setupDone: boolean, inbounds: unknown[]) {
  vi.spyOn(HttpUtil, 'post').mockResolvedValue(new Msg(true, '', { setupDone }));
  vi.spyOn(HttpUtil, 'get').mockResolvedValue(new Msg(true, '', inbounds));
  const client = makeTestQueryClient();
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
  return renderHook(() => useNeedsSetup(), { wrapper });
}

describe('first-run redirect', () => {
  afterEach(() => vi.restoreAllMocks());

  it('sends a fresh panel with no inbounds to the wizard', async () => {
    const { result } = serve(false, []);
    await waitFor(() => expect(result.current).toBe(true));
  });

  it('leaves a panel alone once setup is done, even with no inbounds', async () => {
    const { result } = serve(true, []);
    await waitFor(() => expect(vi.mocked(HttpUtil.get)).toHaveBeenCalled());
    await new Promise((r) => setTimeout(r, 30));
    expect(result.current).toBe(false);
  });

  it('leaves an upgraded panel alone when it already has inbounds', async () => {
    const { result } = serve(false, [{ id: 1, remark: 'main' }]);
    await waitFor(() => expect(vi.mocked(HttpUtil.get)).toHaveBeenCalled());
    await new Promise((r) => setTimeout(r, 30));
    expect(result.current).toBe(false);
  });
});
