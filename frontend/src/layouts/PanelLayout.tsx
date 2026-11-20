import { Outlet, useLocation } from 'react-router';

import { useWebSocketBridge } from '@/api/websocketBridge';
import { usePageTitle } from '@/hooks/usePageTitle';
import CommandPalette from '@/components/command-palette/CommandPalette';
import PanelBackground from '@/components/ui/PanelBackground';

export default function PanelLayout() {
  useWebSocketBridge();
  usePageTitle();
  const { pathname } = useLocation();
  return (
    <>
      <PanelBackground />
      <div key={pathname} className="dl-route">
        <Outlet />
      </div>
      <CommandPalette />
    </>
  );
}
