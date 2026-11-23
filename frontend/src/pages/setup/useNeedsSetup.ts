import { useAllSettings } from '@/api/queries/useAllSettings';
import { useInboundOptions } from '@/api/queries/useInboundOptions';

// A fresh panel (setup not finished and no inbound yet) starts with the first-run wizard.
export function useNeedsSetup(): boolean {
  const settings = useAllSettings();
  const inbounds = useInboundOptions();
  return (
    settings.fetched &&
    !settings.allSetting.setupDone &&
    inbounds.data !== undefined &&
    inbounds.data.length === 0
  );
}
