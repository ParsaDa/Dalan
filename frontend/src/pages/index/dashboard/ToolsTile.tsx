import { useTranslation } from 'react-i18next';

import { Badge, Btn, TileHead } from '@/components/dl/primitives';
import { formatPanelVersion } from '@/lib/panel-version';

export interface ToolsActions {
  onAmneziaWGLogs?: () => void;
  onConfig: () => void;
  onHistory: () => void;
  onMetrics: () => void;
  onUpdate: () => void;
}

// Advanced-mode tools with no page of their own; updates and backups live in Settings.
export function ToolsTile({
  panelVersion,
  latestVersion,
  updateAvailable,
  xrayVersion,
  actions,
}: {
  panelVersion: string;
  latestVersion: string;
  updateAvailable: boolean;
  xrayVersion: string;
  actions: ToolsActions;
}) {
  const { t } = useTranslation();
  return (
    <section className="tile s12" aria-label={t('dashboard.tools.title')}>
      <TileHead
        icon="settings"
        title={t('dashboard.tools.title')}
        right={
          <span className="mut num" style={{ fontSize: 12.5, fontWeight: 650 }}>
            {t('dashboard.tools.versions', {
              panel: formatPanelVersion(panelVersion),
              xray: xrayVersion || '–',
            })}
          </span>
        }
      />
      <div className="row wrap" style={{ gap: 8 }}>
        <Btn size="sm" icon="refresh" onClick={actions.onUpdate}>
          {t('settingsPage.nav.updates')}
          {updateAvailable && <Badge acc>{formatPanelVersion(latestVersion)}</Badge>}
        </Btn>
        {actions.onAmneziaWGLogs && (
          <Btn size="sm" icon="logs" onClick={actions.onAmneziaWGLogs}>
            {t('dashboard.tools.awgLogs')}
          </Btn>
        )}
        <Btn size="sm" icon="code" onClick={actions.onConfig}>
          {t('pages.index.config')}
        </Btn>
        <Btn size="sm" icon="sparkline" onClick={actions.onHistory}>
          {t('pages.index.systemHistoryTitle')}
        </Btn>
        <Btn size="sm" icon="activity" onClick={actions.onMetrics}>
          {t('dashboard.tools.metrics')}
        </Btn>
      </div>
    </section>
  );
}
