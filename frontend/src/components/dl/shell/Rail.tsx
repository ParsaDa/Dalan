import { Fragment } from 'react';
import { useTranslation } from 'react-i18next';
import { Link, useLocation } from 'react-router';

import { PANEL_ROUTE_PATHS } from '@/routes/panelRoutes';
import Icon from '../Icon';
import Mark from '../Mark';
import { useClientTotal } from './data';
import { isActivePath, navFor } from './navigation';
import { shellPrefs, useShellPrefs } from './shellPrefs';

const ROUTED = new Set(PANEL_ROUTE_PATHS);
const DOCS_URL = 'https://github.com/ParsaDa/Dalan#readme';

export default function Rail() {
  const { t, i18n } = useTranslation();
  const { pathname } = useLocation();
  const { simple, slim } = useShellPrefs();
  const clients = useClientTotal();
  const groups = navFor(simple, ROUTED);
  const fmt = new Intl.NumberFormat(i18n.language);
  return (
    <nav className={`rail${slim ? ' slim' : ''}`} aria-label={t('shell.nav.main')}>
      <Link className="brand" to="/">
        <Mark size={34} motion="anim" />
        <span className="wm">{t('shell.brand')}</span>
      </Link>
      {groups.map((g, gi) => (
        <Fragment key={gi}>
          {g.titleKey ? (
            <div className="navg">{t(g.titleKey)}</div>
          ) : (
            gi > 0 && <div className="sep" style={{ margin: '10px 0' }} />
          )}
          {g.items.map((item) => {
            const on = isActivePath(item.path, pathname);
            const label = t(simple && item.simpleLabelKey ? item.simpleLabelKey : item.labelKey);
            const count = item.count === 'clients' && clients !== null ? fmt.format(clients) : null;
            return (
              <Link
                key={item.path}
                to={item.path}
                className={`nav${on ? ' on' : ''}`}
                aria-current={on ? 'page' : undefined}
                title={slim ? label : undefined}
              >
                <Icon name={item.icon} />
                <span className="nl">{label}</span>
                {count && <span className="cnt nl">{count}</span>}
              </Link>
            );
          })}
        </Fragment>
      ))}
      <div className="railfoot">
        <a
          className="nav"
          href={DOCS_URL}
          target="_blank"
          rel="noreferrer"
          title={slim ? t('shell.nav.help') : undefined}
        >
          <Icon name="help" />
          <span className="nl">{t('shell.nav.help')}</span>
        </a>
        <button
          type="button"
          className="nav"
          aria-label={t(slim ? 'shell.nav.expand' : 'shell.nav.collapse')}
          onClick={() => shellPrefs.setSlim(!slim)}
        >
          <span
            style={{
              display: 'inline-flex',
              transform: slim ? 'scaleX(-1)' : undefined,
              transition: 'transform 240ms',
            }}
          >
            <Icon name="back" className="flipx" />
          </span>
          <span className="nl">{t('shell.nav.collapseShort')}</span>
        </button>
      </div>
    </nav>
  );
}
