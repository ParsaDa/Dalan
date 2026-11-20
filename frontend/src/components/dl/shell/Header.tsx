import { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { useTheme } from '@/hooks/useTheme';
import { HttpUtil } from '@/utils';
import { commandPaletteStore } from '@/components/command-palette/useCommandPalette';
import type { EffectsPref } from '@/lib/theme/effects';
import Icon from '../Icon';
import { IconBtn, Seg } from '../primitives';
import { useCurrentUser, useGreetingPart } from './data';
import { shellPrefs, useShellPrefs } from './shellPrefs';
import { useFrameStatus } from './useFrameStatus';

export default function Header() {
  const { t } = useTranslation();
  const { isDark, toggleTheme, effects, setEffectsPref } = useTheme();
  const { simple } = useShellPrefs();
  const user = useCurrentUser();
  const frame = useFrameStatus();
  const [menu, setMenu] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);
  const part = useGreetingPart();

  useEffect(() => {
    if (!menu) return;
    const close = (e: MouseEvent) => {
      if (!menuRef.current?.contains(e.target as Node)) setMenu(false);
    };
    document.addEventListener('mousedown', close);
    return () => document.removeEventListener('mousedown', close);
  }, [menu]);

  const logout = async () => {
    await HttpUtil.post('/logout');
    window.location.href = window.X_UI_BASE_PATH || '/';
  };

  return (
    <header className="hdr">
      <div>
        <h1 className="h1">
          {user ? t(`shell.greeting.${part}`, { name: user }) : t(`shell.greeting.${part}Bare`)}
        </h1>
        <div className="statusline">
          <span className="live" aria-hidden="true" />
          <b>{frame.name}</b>
          <span>·</span>
          <span className={frame.tone === 'ok' ? '' : `pill ${frame.tone}`}>{frame.word}</span>
          <span>·</span>
          <span className="num" aria-live="polite">
            {t('shell.status.updated', { n: frame.ago })}
          </span>
        </div>
      </div>
      <div className="tools">
        <button
          type="button"
          className="search"
          onClick={() => commandPaletteStore.open()}
          aria-label={t('shell.search.label')}
        >
          <Icon name="search" />
          <span>{t('shell.search.field')}</span>
          <kbd>Ctrl K</kbd>
        </button>
        <Seg
          label={t('shell.mode.label')}
          value={simple ? 'simple' : 'advanced'}
          onChange={(v) => shellPrefs.setSimple(v === 'simple')}
          options={[
            { value: 'simple', label: t('shell.mode.simple') },
            { value: 'advanced', label: t('shell.mode.advanced') },
          ]}
        />
        <Seg<EffectsPref>
          label={t('shell.effects.label')}
          value={effects}
          onChange={setEffectsPref}
          lead={
            <span
              style={{
                display: 'inline-grid',
                placeItems: 'center',
                padding: '0 4px 0 8px',
                color: 'var(--mut)',
              }}
            >
              <Icon name="fx" size={16} />
            </span>
          }
          options={[
            { value: 'full', label: t('shell.effects.full') },
            { value: 'light', label: t('shell.effects.light') },
            { value: 'off', label: t('shell.effects.off') },
          ]}
        />
        <IconBtn
          icon={isDark ? 'sun' : 'moon'}
          label={t(isDark ? 'shell.theme.toLight' : 'shell.theme.toDark')}
          onClick={toggleTheme}
        />
        <IconBtn icon="bell" label={t('shell.notifications')} onClick={() => frame.openAlerts()} />
        <div ref={menuRef} style={{ position: 'relative' }}>
          <button
            type="button"
            className="ava"
            style={{ border: 0, cursor: 'pointer' }}
            aria-label={user || t('shell.account')}
            aria-expanded={menu}
            onClick={() => setMenu((v) => !v)}
          >
            {(user || '·').slice(0, 1).toUpperCase()}
          </button>
          {menu && (
            <div
              className="tile dense"
              style={{
                position: 'absolute',
                insetInlineEnd: 0,
                top: 50,
                zIndex: 30,
                padding: 8,
                minWidth: 200,
              }}
            >
              <div className="mut" style={{ padding: '6px 10px', fontSize: 12.5, fontWeight: 650 }}>
                {user}
              </div>
              <button type="button" className="nav" onClick={logout}>
                <Icon name="power" />
                <span>{t('logout')}</span>
              </button>
            </div>
          )}
        </div>
      </div>
    </header>
  );
}
