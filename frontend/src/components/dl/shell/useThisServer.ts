import { useTranslation } from 'react-i18next';

import { useAllSettings } from '@/api/queries/useAllSettings';

// The name and flag set in the first-run wizard, else the generic label.
export function useThisServer(): { name: string; country: string } {
  const { t } = useTranslation();
  const { allSetting } = useAllSettings();
  return {
    name: allSetting.serverName?.trim() || t('pages.index.thisServer'),
    country: allSetting.serverCountry ?? '',
  };
}
