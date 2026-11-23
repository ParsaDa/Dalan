import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { Link, useLocation } from 'react-router';

import { PANEL_TITLE_KEYS } from '@/routes/panelRoutes';

import { useMediaQuery } from '@/hooks/useMediaQuery';
import { commandPaletteStore } from '@/components/command-palette/useCommandPalette';
import Mark from '../Mark';
import { IconBtn } from '../primitives';
import Header from './Header';
import NodeTabs from './NodeTabs';
import Rail from './Rail';
import TabBar from './TabBar';
import { useCurrentUser, useGreetingPart } from './data';
import { useFrameStatus } from './useFrameStatus';

export const PHONE_MAX_PX = 700;

// Off the dashboard a phone shows only the page name in the top bar, as in the design.
function PhoneTitleTop({ title }: { title: string }) {
  const { t } = useTranslation();
  return (
    <div className="ptop">
      <Link to="/" aria-label={t('shell.brand')} style={{ display: 'flex', color: 'inherit' }}>
        <Mark size={30} />
      </Link>
      <h1 className="h2" style={{ flex: 1, fontSize: 22 }}>
        {title}
      </h1>
      <IconBtn
        small
        icon="search"
        label={t('shell.search.label')}
        onClick={() => commandPaletteStore.open()}
      />
    </div>
  );
}

function PhoneTop() {
  const { t } = useTranslation();
  const user = useCurrentUser();
  const frame = useFrameStatus();
  const part = useGreetingPart();
  return (
    <>
      <div className="ptop">
        <Link to="/" className="row" style={{ gap: 10, color: 'inherit', textDecoration: 'none' }}>
          <Mark size={30} />
          <b className="wm" style={{ fontSize: 18 }}>
            {t('shell.brand')}
          </b>
        </Link>
        <div className="row" style={{ gap: 8 }}>
          <IconBtn
            small
            icon="search"
            label={t('shell.search.label')}
            onClick={() => commandPaletteStore.open()}
          />
          <IconBtn small icon="bell" label={t('shell.notifications')} onClick={frame.openAlerts} />
        </div>
      </div>
      <div>
        <h1 className="h1">
          {user ? t(`shell.greeting.${part}`, { name: user }) : t(`shell.greeting.${part}Bare`)}
        </h1>
        <div className="statusline" style={{ fontSize: 13 }}>
          <span className="live" aria-hidden="true" />
          <b>{frame.name}</b>
          <span className={frame.tone === 'ok' ? 'mut' : `pill ${frame.tone}`}>{frame.word}</span>
        </div>
      </div>
    </>
  );
}

// Every panel page renders inside this frame; `bare` drops the greeting and node tabs.
export default function Shell({ children, bare }: { children: ReactNode; bare?: boolean }) {
  const { t } = useTranslation();
  const { pathname } = useLocation();
  const { isMobile: phone } = useMediaQuery(PHONE_MAX_PX);
  if (phone) {
    const titleKey = pathname === '/' ? undefined : PANEL_TITLE_KEYS[pathname];
    return (
      <>
        <main className={`main pmain${titleKey ? ' ptitled' : ''}`}>
          {!bare && (titleKey ? <PhoneTitleTop title={t(titleKey)} /> : <PhoneTop />)}
          {!bare && <NodeTabs phone />}
          {children}
        </main>
        <TabBar />
      </>
    );
  }
  return (
    <div className="shell">
      <Rail />
      <main className="main">
        {!bare && <Header />}
        {!bare && <NodeTabs />}
        {children}
      </main>
    </div>
  );
}
