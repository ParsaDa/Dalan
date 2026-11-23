import { Outlet, useLocation } from 'react-router';
import { ConfigProvider } from 'antd';

import { useWebSocketBridge } from '@/api/websocketBridge';
import { usePageTitle } from '@/hooks/usePageTitle';
import { useTheme } from '@/hooks/useTheme';
import CommandPalette from '@/components/command-palette/CommandPalette';
import Shell from '@/components/dl/shell/Shell';
import { ToastProvider } from '@/components/dl/overlays';

export default function PanelLayout() {
  useWebSocketBridge();
  usePageTitle();
  const { pathname } = useLocation();
  const { antdThemeConfig } = useTheme();
  // Form fields kept from Ant Design follow the panel theme on every page.
  return (
    <ConfigProvider theme={antdThemeConfig}>
      <ToastProvider>
        <div className="bgfx" aria-hidden="true">
          <div className="glow g1" />
          <div className="glow g2" />
          <div className="glow g3" />
        </div>
        <Shell bare={pathname === '/more'}>
          <div key={pathname} className="route col" style={{ gap: 18 }}>
            <Outlet />
          </div>
        </Shell>
        <CommandPalette />
      </ToastProvider>
    </ConfigProvider>
  );
}
