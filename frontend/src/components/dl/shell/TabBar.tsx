import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Link, useLocation, useNavigate } from 'react-router';

import Icon from '../Icon';
import type { IconName } from '../Icon';
import { Sheet } from '../overlays';
import { isActivePath } from './navigation';
import { useFrameStatus } from './useFrameStatus';

const TABS: { path: string; icon: IconName; key: string }[] = [
  { path: '/', icon: 'dash', key: 'shell.nav.home' },
  { path: '/clients', icon: 'users', key: 'shell.nav.users' },
  { path: '/tunnels', icon: 'tunnel', key: 'menu.tunnels' },
  { path: '/more', icon: 'menu', key: 'shell.nav.more' },
];

const QUICK: { to: string; icon: IconName; key: string }[] = [
  { to: '/clients?add=1', icon: 'users', key: 'shell.quick.user' },
  { to: '/inbounds?add=1', icon: 'inbound', key: 'shell.quick.connection' },
  { to: '/nodes?add=1', icon: 'nodes', key: 'shell.quick.server' },
];

export default function TabBar() {
  const { t } = useTranslation();
  const { pathname } = useLocation();
  const navigate = useNavigate();
  const frame = useFrameStatus();
  const [add, setAdd] = useState(false);
  const link = (tab: (typeof TABS)[number]) => {
    const on = isActivePath(tab.path, pathname);
    return (
      <Link
        key={tab.path}
        to={tab.path}
        className={on ? 'on' : undefined}
        aria-current={on ? 'page' : undefined}
      >
        <Icon name={tab.icon} />
        {t(tab.key)}
      </Link>
    );
  };
  return (
    <>
      <nav className="tabbar" aria-label={t('shell.nav.main')}>
        {link(TABS[0])}
        {link(TABS[1])}
        <button
          type="button"
          aria-label={t('shell.quick.label')}
          onClick={() => setAdd(true)}
          style={{
            flex: 'none',
            width: 56,
            height: 56,
            marginTop: -14,
            borderRadius: 18,
            background: 'var(--acc)',
            color: 'var(--on-acc)',
            display: 'grid',
            placeItems: 'center',
            padding: 0,
            boxShadow: '0 12px 26px -10px var(--acc)',
          }}
        >
          <Icon name="plus" />
        </button>
        {link(TABS[2])}
        {link(TABS[3])}
      </nav>
      <Sheet open={add} onClose={() => setAdd(false)} label={t('shell.quick.label')}>
        <h2 className="h3" style={{ marginBottom: 10 }}>
          {t('shell.quick.title', { name: frame.name })}
        </h2>
        <div className="list">
          {QUICK.map((q) => (
            <button
              key={q.to}
              type="button"
              className="nav"
              style={{ height: 56 }}
              onClick={() => {
                setAdd(false);
                navigate(q.to);
              }}
            >
              <span className="ico acc">
                <Icon name={q.icon} />
              </span>
              <span className="f1" style={{ color: 'var(--tx)' }}>
                {t(q.key)}
              </span>
            </button>
          ))}
        </div>
      </Sheet>
    </>
  );
}
