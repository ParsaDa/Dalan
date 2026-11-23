import { useTranslation } from 'react-i18next';
import { isRouteErrorResponse, Link, useRouteError } from 'react-router';

import Icon from '@/components/dl/Icon';
import { IlloOffline } from '@/components/dl/Illustrations';

// The artboard's "lost in the corridor" page: an unknown address, or a page that failed to load.
export default function ErrorPage({ notFound = false }: { notFound?: boolean }) {
  const { t } = useTranslation();
  const error = useRouteError();
  const status = notFound ? 404 : isRouteErrorResponse(error) ? error.status : 500;
  const detail = !notFound && error instanceof Error ? error.message : '';
  return (
    <div style={{ minHeight: '70vh', display: 'grid', placeItems: 'center', padding: '40px 20px' }}>
      <div className="empty" style={{ maxWidth: 520 }}>
        <IlloOffline size={220} />
        <h1 className="h1">{status === 404 ? t('errorPage.lost') : t('errorPage.broken')}</h1>
        <p style={{ fontSize: 15 }}>
          {status === 404 ? t('errorPage.lostHint') : t('errorPage.brokenHint')}
        </p>
        <div className="row wrap" style={{ justifyContent: 'center' }}>
          <Link className="btn pri" to="/">
            <Icon name="dash" />
            {t('errorPage.home')}
          </Link>
          <button type="button" className="btn" onClick={() => window.location.reload()}>
            <Icon name="refresh" />
            {t('errorPage.retry')}
          </button>
        </div>
        <div className="mut mono" style={{ fontSize: 12 }}>
          {`error ${status}${detail ? ` · ${detail}` : ''}`}
        </div>
      </div>
    </div>
  );
}
