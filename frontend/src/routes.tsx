import { lazy, Suspense } from 'react';
import { createBrowserRouter, type RouteObject } from 'react-router';

import Mark from '@/components/dl/Mark';
import PanelLayout from '@/layouts/PanelLayout';
import ErrorPage from '@/pages/errors/ErrorPage';

const SetupPage = lazy(() => import('@/pages/setup/SetupPage'));
import { PANEL_ROUTES } from '@/routes/panelRoutes';

function withSuspense(node: React.ReactNode) {
  return (
    <Suspense
      fallback={
        <div
          style={{
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
            minHeight: '60vh',
          }}
        >
          <Mark size={56} motion="loop" title="Loading" />
        </div>
      }
    >
      {node}
    </Suspense>
  );
}

export const routes: RouteObject[] = [
  {
    path: '/',
    element: <PanelLayout />,
    errorElement: <ErrorPage />,
    children: [
      ...PANEL_ROUTES.map((route) => {
        const element = withSuspense(<route.component />);
        return route.path === '/'
          ? { index: true, element }
          : { path: route.path.slice(1), element };
      }),
      { path: '*', element: <ErrorPage notFound /> },
    ],
  },
  { path: '/setup', element: withSuspense(<SetupPage />), errorElement: <ErrorPage /> },
];

function computeBasename() {
  const raw = (typeof window !== 'undefined' && window.X_UI_BASE_PATH) || '/';
  const trimmed = raw.replace(/\/+$/, '');
  return `${trimmed}/panel`;
}

export const router = createBrowserRouter(routes, {
  basename: computeBasename(),
});
