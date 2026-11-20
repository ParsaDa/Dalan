import { useTranslation } from 'react-i18next';
import { Segmented } from 'antd';

import AccentSwatches from '@/components/ui/AccentSwatches';
import SettingListItem from '@/components/ui/SettingListItem';
import { useTheme } from '@/hooks/useTheme';
import type { EffectsPref } from '@/lib/theme/effects';

const LEVELS: EffectsPref[] = ['auto', 'full', 'light', 'off'];

export default function AppearanceSettings() {
  const { t } = useTranslation();
  const { effectsPref, setEffectsPref, accent, setAccent } = useTheme();
  return (
    <>
      <SettingListItem
        paddings="small"
        title={t('pages.settings.effects')}
        description={t('pages.settings.effectsDesc')}
      >
        <Segmented
          block
          value={effectsPref}
          onChange={(value) => setEffectsPref(value as EffectsPref)}
          options={LEVELS.map((level) => ({
            value: level,
            label: t(`pages.settings.effects_${level}`),
          }))}
        />
      </SettingListItem>
      <SettingListItem paddings="small" title={t('pages.settings.accentColour')}>
        <div className="accent-row" role="radiogroup" aria-label={t('pages.settings.accentColour')}>
          <AccentSwatches value={accent} onChange={(id) => setAccent(id as typeof accent)} />
        </div>
      </SettingListItem>
    </>
  );
}
