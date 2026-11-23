import { useTranslation } from 'react-i18next';
import { useQueryClient } from '@tanstack/react-query';

import { keys } from '@/api/queryKeys';

import AccentSwatches from '@/components/ui/AccentSwatches';
import { Seg, TileHead, Toggle } from '@/components/dl/primitives';
import { shellPrefs, useShellPrefs } from '@/components/dl/shell/shellPrefs';
import { useTheme } from '@/hooks/useTheme';
import type { EffectsPref } from '@/lib/theme/effects';
import { HttpUtil } from '@/utils';

const LEVELS: EffectsPref[] = ['auto', 'full', 'light', 'off'];

// The Appearance section of Settings: effects, theme, panel mode and accent, all kept per device.
export default function AppearanceSettings() {
  const { t } = useTranslation();
  const { effectsPref, setEffectsPref, accent, setAccent, isDark, toggleTheme } = useTheme();
  const { simple } = useShellPrefs();
  const queryClient = useQueryClient();

  // Kept on this device at once, then saved so every browser of this panel shows the same colour.
  async function pickAccent(id: typeof accent) {
    setAccent(id);
    const saved = await HttpUtil.post('/panel/api/setting/profile', { accent: id });
    if (saved?.success) await queryClient.invalidateQueries({ queryKey: keys.settings.all() });
  }
  return (
    <>
      <section className="tile">
        <TileHead
          icon="fx"
          title={t('pages.settings.effects')}
          right={
            <span className="mut" style={{ fontSize: 12.5, fontWeight: 650 }}>
              {t('settingsPage.thisDevice')}
            </span>
          }
        />
        <div
          role="radiogroup"
          aria-label={t('pages.settings.effects')}
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))',
            gap: 12,
          }}
        >
          {LEVELS.map((level) => (
            <label
              key={level}
              className="tile"
              style={{
                padding: 14,
                cursor: 'pointer',
                display: 'flex',
                flexDirection: 'column',
                gap: 8,
                outline: effectsPref === level ? '2px solid var(--acc-line)' : undefined,
                outlineOffset: -1,
              }}
            >
              <span className="row sp">
                <b>{t(`pages.settings.effects_${level}`)}</b>
                <input
                  type="radio"
                  name="effects"
                  checked={effectsPref === level}
                  onChange={() => setEffectsPref(level)}
                  style={{ accentColor: 'var(--acc)' }}
                />
              </span>
              <span className="mut" style={{ fontSize: 12.5 }}>
                {t(`settingsPage.effects.${level}`)}
              </span>
            </label>
          ))}
        </div>
      </section>
      <section className="tile">
        <TileHead icon="sun" title={t('settingsPage.theme')} />
        <Seg
          label={t('settingsPage.theme')}
          value={isDark ? 'dark' : 'light'}
          onChange={(v) => {
            if ((v === 'dark') !== isDark) toggleTheme();
          }}
          options={[
            { value: 'dark', label: t('settingsPage.dark') },
            { value: 'light', label: t('settingsPage.light') },
          ]}
        />
      </section>
      <section className="tile">
        <TileHead icon="layers" title={t('settingsPage.panelMode')} />
        <div className="row sp wrap" style={{ gap: 12 }}>
          <div>
            <b>{t('settingsPage.simpleMode')}</b>
            <div className="mut" style={{ fontSize: 13, maxWidth: 520 }}>
              {t('settingsPage.simpleModeHint')}
            </div>
          </div>
          <Toggle
            on={simple}
            label={t('settingsPage.simpleMode')}
            onChange={(on) => shellPrefs.setSimple(on)}
          />
        </div>
      </section>
      <section className="tile">
        <TileHead icon="fx" title={t('pages.settings.accentColour')} />
        <div className="accent-row" role="radiogroup" aria-label={t('pages.settings.accentColour')}>
          <AccentSwatches value={accent} onChange={(id) => pickAccent(id as typeof accent)} />
        </div>
      </section>
    </>
  );
}
