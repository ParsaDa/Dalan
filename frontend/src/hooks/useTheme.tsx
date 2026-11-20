import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useLayoutEffect,
  useMemo,
  useState,
} from 'react';
import type { ReactNode } from 'react';
import { theme as antdTheme } from 'antd';
import type { ThemeConfig } from 'antd';

import { DEFAULT_ACCENT } from '@/lib/theme/accents';
import type { AccentId } from '@/lib/theme/accents';
import { readEffectsEnv, resolveEffects } from '@/lib/theme/effects';
import type { Effects, EffectsPref } from '@/lib/theme/effects';
import { deriveAccent } from '@/lib/theme/deriveAccent';
import type { ResolvedAccent } from '@/lib/theme/nodeAccent';

const DL_ROOT_PREFIXES = ['t-', 'fx-', 'ac-'];
const CUSTOM_VARS = ['--acc', '--acc-line', '--acc-text', '--on-acc'];

// <body> is the design's .dl root, so portals (modals, toasts, popups) inherit theme and accent.
function applyDlRoot(isDark: boolean, effects: Effects, accent: ResolvedAccent) {
  const body = document.body;
  for (const cls of [...body.classList]) {
    if (DL_ROOT_PREFIXES.some((p) => cls.startsWith(p))) body.classList.remove(cls);
  }
  const id = accent.kind === 'preset' ? accent.id : 'saffron';
  body.classList.add('dl', isDark ? 't-dark' : 't-light', `fx-${effects}`, `ac-${id}`);
  for (const v of CUSTOM_VARS) body.style.removeProperty(v);
  if (accent.kind === 'custom') {
    const d = deriveAccent(accent.hex, isDark ? 'dark' : 'light');
    body.style.setProperty('--acc', d.fill);
    body.style.setProperty('--acc-line', d.line);
    body.style.setProperty('--acc-text', d.text);
    body.style.setProperty('--on-acc', d.on);
  }
}

const STORAGE_DARK = 'dark-mode';
const STORAGE_ULTRA = 'isUltraDarkThemeEnabled';
const STORAGE_EFFECTS = 'dalan-effects';
const EFFECTS_PREFS: readonly EffectsPref[] = ['auto', 'full', 'light', 'off'];

function readEffectsPref(): EffectsPref {
  const raw = localStorage.getItem(STORAGE_EFFECTS);
  return EFFECTS_PREFS.find((pref) => pref === raw) ?? 'auto';
}

function readBool(key: string, fallback: boolean): boolean {
  const raw = localStorage.getItem(key);
  if (raw === null) return fallback;
  return raw === 'true';
}

function applyDom(isDark: boolean, isUltra: boolean) {
  document.body.classList.remove('dark', 'light');
  document.body.classList.add(isDark ? 'dark' : 'light');
  // Native scrollbars read color-scheme, not the body class.
  document.documentElement.style.colorScheme = isDark ? 'dark' : 'light';
  if (isUltra) {
    document.documentElement.setAttribute('data-theme', 'ultra-dark');
  } else {
    document.documentElement.removeAttribute('data-theme');
  }
  const msg = document.getElementById('message');
  if (msg) {
    msg.classList.remove('dark', 'light');
    msg.classList.add(isDark ? 'dark' : 'light');
  }
}

// module load so the document is in the right theme before React mounts.
const initialDark = readBool(STORAGE_DARK, true);
const initialUltra = readBool(STORAGE_ULTRA, false);
applyDom(initialDark, initialUltra);

const FONT_FAMILY = "'Manrope Variable', 'Vazirmatn Variable', system-ui, sans-serif";

const SHARED_TOKENS = {
  fontFamily: FONT_FAMILY,
  borderRadius: 12,
  borderRadiusLG: 18,
  borderRadiusSM: 8,
  controlHeight: 40,
  controlHeightLG: 52,
  motionDurationFast: '0.14s',
  motionDurationMid: '0.24s',
  motionDurationSlow: '0.42s',
  motionEaseOut: 'cubic-bezier(0.22, 1, 0.36, 1)',
};

const DARK_TOKENS = {
  ...SHARED_TOKENS,
  colorPrimary: '#ffb020',
  colorBgBase: '#111114',
  colorBgLayout: 'transparent',
  colorBgContainer: '#1b1b20',
  colorBgElevated: '#222228',
  colorText: '#f4efe6',
  colorTextSecondary: '#afaaa0',
  colorBorder: '#8c877d',
  colorBorderSecondary: 'rgba(255, 255, 255, 0.13)',
  colorSuccess: '#34d399',
  colorWarning: '#ff7a45',
  colorError: '#f87171',
};
const ULTRA_DARK_TOKENS = {
  ...DARK_TOKENS,
  colorBgBase: '#000',
  colorBgContainer: '#101013',
  colorBgElevated: '#1a1a1e',
};
const TRANSPARENT_LAYOUT_TOKENS = {
  bodyBg: 'transparent',
  headerBg: 'transparent',
  footerBg: 'transparent',
  siderBg: 'transparent',
  triggerBg: 'transparent',
};
const DARK_LAYOUT_TOKENS = {
  ...TRANSPARENT_LAYOUT_TOKENS,
  headerColor: '#f4efe6',
  triggerColor: '#f4efe6',
};
const DARK_MENU_TOKENS = {
  darkItemBg: 'transparent',
  darkSubMenuItemBg: 'transparent',
  darkPopupBg: '#222228',
};
const ULTRA_DARK_MENU_TOKENS = { ...DARK_MENU_TOKENS, darkPopupBg: '#1a1a1e' };
const STATISTIC_TOKENS = {
  contentFontSize: 17,
  titleFontSize: 11,
};
const LIGHT_TOKENS = {
  ...SHARED_TOKENS,
  colorPrimary: '#906102',
  colorBgBase: '#f4efe6',
  colorBgLayout: 'transparent',
  colorBgContainer: '#ffffff',
  colorBgElevated: '#ffffff',
  colorText: '#111114',
  colorTextSecondary: '#6b665c',
  colorTextDescription: '#6b665c',
  colorTextTertiary: '#6b665c',
  colorTextPlaceholder: '#767676',
  colorBorder: '#857f75',
  colorBorderSecondary: 'rgba(17, 17, 20, 0.11)',
  colorSuccess: '#047857',
  colorSuccessText: '#047857',
  colorWarning: '#c2410c',
  colorError: '#b91c1c',
  colorErrorText: '#b91c1c',
};
// Text on an accent fill is ink in both themes, whatever the accent.
const PRIMARY_BUTTON_TOKENS = { primaryColor: '#111114', fontWeight: 700 };
// The light colorPrimary is a dark amber for text; ink on it fails 4.5:1, so buttons fill with saffron.
const LIGHT_PRIMARY_BUTTON_TOKENS = {
  ...PRIMARY_BUTTON_TOKENS,
  colorPrimary: '#ffb020',
  algorithm: true,
};

// hashed:false drops the `:where(.css-<hash>)` wrapper antd puts around every
// rule. It costs nothing in specificity — `:where()` contributes zero, so the
// panel's own `.ant-*` overrides still win — and it removes roughly 5,700
// wrappers, 16% of the generated stylesheet, from what the browser has to parse.
//
// cssVar.key pins the CSS-variable scope. Every panel page mounts its own
// ConfigProvider (there is no root one), and without a fixed key each mints a
// fresh useId-derived scope, so navigating re-serialises and re-injects the whole
// token block under a new class instead of reusing the one already in the head.
const SHARED_STYLE_CONFIG = {
  hashed: false,
  cssVar: { key: 'xui' },
} as const;

export function buildAntdThemeConfig(isDark: boolean, isUltra: boolean): ThemeConfig {
  if (!isDark) {
    return {
      ...SHARED_STYLE_CONFIG,
      algorithm: antdTheme.defaultAlgorithm,
      token: LIGHT_TOKENS,
      components: {
        Layout: TRANSPARENT_LAYOUT_TOKENS,
        Statistic: STATISTIC_TOKENS,
        Button: LIGHT_PRIMARY_BUTTON_TOKENS,
      },
    };
  }
  return {
    ...SHARED_STYLE_CONFIG,
    algorithm: antdTheme.darkAlgorithm,
    token: isUltra ? ULTRA_DARK_TOKENS : DARK_TOKENS,
    components: {
      Layout: DARK_LAYOUT_TOKENS,
      Menu: isUltra ? ULTRA_DARK_MENU_TOKENS : DARK_MENU_TOKENS,
      Statistic: STATISTIC_TOKENS,
      Button: PRIMARY_BUTTON_TOKENS,
    },
  };
}

export function pauseAnimationsUntilLeave(elementId: string): void {
  document.documentElement.setAttribute('data-theme-animations', 'off');
  const el = document.getElementById(elementId);
  if (!el) return;
  const restore = () => {
    document.documentElement.removeAttribute('data-theme-animations');
    el.removeEventListener('mouseleave', restore);
    el.removeEventListener('touchend', restore);
  };
  el.addEventListener('mouseleave', restore);
  el.addEventListener('touchend', restore);
}

interface ThemeContextValue {
  isDark: boolean;
  isUltra: boolean;
  toggleTheme: () => void;
  toggleUltra: () => void;
  antdThemeConfig: ThemeConfig;
  effectsPref: EffectsPref;
  effects: Effects;
  setEffectsPref: (pref: EffectsPref) => void;
  accent: AccentId;
  setAccent: (accent: AccentId) => void;
  // The open node's accent replaces the panel accent in the whole frame; null restores it.
  setFrameAccent: (accent: ResolvedAccent | null) => void;
}

const ThemeContext = createContext<ThemeContextValue | null>(null);

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [isDark, setIsDark] = useState<boolean>(initialDark);
  const [isUltra, setIsUltra] = useState<boolean>(initialUltra);

  useLayoutEffect(() => {
    applyDom(isDark, isUltra);
    localStorage.setItem(STORAGE_DARK, String(isDark));
    localStorage.setItem(STORAGE_ULTRA, String(isUltra));
  }, [isDark, isUltra]);

  const [effectsPref, setEffectsPrefState] = useState<EffectsPref>(readEffectsPref);
  const [reducedMotion, setReducedMotion] = useState(() => readEffectsEnv().reducedMotion);
  const [accent, setAccent] = useState<AccentId>(DEFAULT_ACCENT);

  useEffect(() => {
    const query = window.matchMedia('(prefers-reduced-motion: reduce)');
    const onChange = () => setReducedMotion(query.matches);
    query.addEventListener('change', onChange);
    return () => query.removeEventListener('change', onChange);
  }, []);

  const effects = useMemo(
    () => resolveEffects(effectsPref, { ...readEffectsEnv(), reducedMotion }),
    [effectsPref, reducedMotion],
  );

  useLayoutEffect(() => {
    document.body.dataset.effects = effects;
  }, [effects]);

  useLayoutEffect(() => {
    document.body.dataset.accent = accent;
  }, [accent]);

  const [frameAccent, setFrameAccent] = useState<ResolvedAccent | null>(null);
  useLayoutEffect(() => {
    applyDlRoot(isDark, effects, frameAccent ?? { kind: 'preset', id: accent });
  }, [isDark, effects, accent, frameAccent]);

  // Ambient motion and the 5 s refresh both stop while the tab is hidden.
  useEffect(() => {
    const sync = () => {
      document.body.classList.toggle('dl-paused', document.hidden);
      document.body.classList.toggle('paused', document.hidden);
    };
    document.addEventListener('visibilitychange', sync);
    return () => document.removeEventListener('visibilitychange', sync);
  }, []);

  const setEffectsPref = useCallback((pref: EffectsPref) => {
    localStorage.setItem(STORAGE_EFFECTS, pref);
    setEffectsPrefState(pref);
  }, []);

  const toggleTheme = useCallback(() => setIsDark((v) => !v), []);
  const toggleUltra = useCallback(() => setIsUltra((v) => !v), []);

  const antdThemeConfig = useMemo(() => buildAntdThemeConfig(isDark, isUltra), [isDark, isUltra]);

  const value = useMemo<ThemeContextValue>(
    () => ({
      isDark,
      isUltra,
      toggleTheme,
      toggleUltra,
      antdThemeConfig,
      effectsPref,
      effects,
      setEffectsPref,
      accent,
      setAccent,
      setFrameAccent,
    }),
    [
      isDark,
      isUltra,
      toggleTheme,
      toggleUltra,
      antdThemeConfig,
      effectsPref,
      effects,
      setEffectsPref,
      accent,
    ],
  );

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme(): ThemeContextValue {
  const ctx = useContext(ThemeContext);
  if (!ctx) throw new Error('useTheme must be used inside <ThemeProvider>');
  return ctx;
}
