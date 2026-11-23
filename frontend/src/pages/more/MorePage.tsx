import { useTranslation } from 'react-i18next';

import { useAllSettings } from '@/api/queries/useAllSettings';
import { useInboundOptions } from '@/api/queries/useInboundOptions';
import { fmtInt } from '@/components/dl/numbers';
import { Link } from 'react-router';

import Icon from '@/components/dl/Icon';
import type { IconName } from '@/components/dl/Icon';
import Mark from '@/components/dl/Mark';
import { IconBtn } from '@/components/dl/primitives';
import { shellPrefs, useShellPrefs } from '@/components/dl/shell/shellPrefs';
import { commandPaletteStore } from '@/components/command-palette/useCommandPalette';
import { useTheme } from '@/hooks/useTheme';
import type { EffectsPref } from '@/lib/theme/effects';

interface Row {
  icon: IconName;
  label: string;
  value?: string;
  to?: string;
  onClick?: () => void;
}

const EFFECT_ORDER: EffectsPref[] = ['full', 'light', 'off'];

// The phone's "More" tab: everything the bottom bar has no room for, grouped like the rail.
export default function MorePage() {
  const { t, i18n } = useTranslation();
  const { isDark, toggleTheme, effects, setEffectsPref } = useTheme();
  const { simple } = useShellPrefs();
  const settings = useAllSettings();
  const inbounds = useInboundOptions();
  const groups: { title: string; rows: Row[] }[] = [
    {
      title: t('morePage.panel'),
      rows: [
        {
          icon: 'fx',
          label: t('shell.effects.label'),
          value: t(`shell.effects.${effects}`),
          onClick: () => setEffectsPref(EFFECT_ORDER[(EFFECT_ORDER.indexOf(effects) + 1) % 3]),
        },
        {
          icon: 'sun',
          label: t('menu.theme'),
          value: t(isDark ? 'morePage.dark' : 'morePage.light'),
          onClick: toggleTheme,
        },
        {
          icon: 'layers',
          label: t('shell.mode.label'),
          value: t(simple ? 'shell.mode.simple' : 'shell.mode.advanced'),
          onClick: () => shellPrefs.setSimple(!simple),
        },
        {
          icon: 'globe2',
          label: t('morePage.language'),
          value:
            new Intl.DisplayNames([i18n.language], { type: 'language' }).of(i18n.language) ?? '',
          to: '/settings',
        },
      ],
    },
    {
      title: t('morePage.server'),
      rows: [
        {
          icon: 'inbound',
          label: t('menu.inbounds'),
          value: inbounds.data ? fmtInt(inbounds.data.length) : undefined,
          to: '/inbounds',
        },
        { icon: 'groups', label: t('menu.groups'), to: '/groups' },
        { icon: 'hosts', label: t('menu.hosts'), to: '/hosts' },
        { icon: 'nodes', label: t('menu.nodes'), to: '/nodes' },
        { icon: 'outbound', label: t('morePage.outboundsRouting'), to: '/outbound' },
        { icon: 'xray', label: t('menu.xray'), to: '/xray' },
        { icon: 'logs', label: t('menu.logs'), to: '/logs' },
      ],
    },
    {
      title: t('morePage.account'),
      rows: [
        { icon: 'settings', label: t('menu.settings'), to: '/settings' },
        {
          icon: 'lock',
          label: t('settingsPage.nav.security'),
          value: settings.fetched
            ? t(settings.allSetting.twoFactorEnable ? 'morePage.twoStepOn' : 'morePage.twoStepOff')
            : undefined,
          to: '/settings#security',
        },
        { icon: 'server', label: t('settingsPage.nav.backups'), to: '/settings#backups' },
        { icon: 'code', label: t('menu.apiDocs'), to: '/api-docs' },
        {
          icon: 'users',
          label: t('settingsPage.nav.users'),
          value: t('settingsPage.comingLater'),
          to: '/settings#users',
        },
      ],
    },
  ];
  return (
    <div className="col stag" style={{ gap: 22 }}>
      <div className="ptop">
        <Mark size={30} />
        <h1 className="h2" style={{ flex: 1, fontSize: 22 }}>
          {t('shell.nav.more')}
        </h1>
        <IconBtn
          small
          icon="search"
          label={t('shell.search.label')}
          onClick={() => commandPaletteStore.open()}
        />
      </div>
      {groups.map((g) => (
        <div key={g.title} className="col" style={{ gap: 6 }}>
          <div className="navg" style={{ padding: '6px 4px 0' }}>
            {g.title}
          </div>
          <section className="tile" style={{ padding: '4px 14px' }}>
            {g.rows.map((r) => {
              const inner = (
                <>
                  <span className="ico">
                    <Icon name={r.icon} />
                  </span>
                  <b className="f1" style={{ textAlign: 'start' }}>
                    {r.label}
                  </b>
                  {r.value && (
                    <span className="mut" style={{ fontSize: 13, fontWeight: 650 }}>
                      {r.value}
                    </span>
                  )}
                  <span className="mut">
                    <Icon name="chev" size={16} className="flipx" />
                  </span>
                </>
              );
              const style = {
                color: 'inherit',
                textDecoration: 'none',
                width: '100%',
                background: 'none',
                border: 0,
                font: 'inherit',
                cursor: 'pointer',
              };
              return r.to ? (
                <Link key={r.label} className="li" to={r.to} style={style}>
                  {inner}
                </Link>
              ) : (
                <button
                  key={r.label}
                  type="button"
                  className="li"
                  style={style}
                  onClick={r.onClick}
                >
                  {inner}
                </button>
              );
            })}
          </section>
        </div>
      ))}
    </div>
  );
}
