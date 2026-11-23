import { useEffect } from 'react';

import { useAllSettings } from '@/api/queries/useAllSettings';
import { useTheme } from '@/hooks/useTheme';
import { parseAccentId } from '@/lib/theme/accents';

// The panel accent lives on the server; this device only caches it so the first paint is right.
export function useServerAccent() {
  const { allSetting, fetched } = useAllSettings();
  const { setAccent } = useTheme();
  const saved = parseAccentId(allSetting.accent);
  useEffect(() => {
    if (fetched && saved) setAccent(saved);
  }, [fetched, saved, setAccent]);
}
