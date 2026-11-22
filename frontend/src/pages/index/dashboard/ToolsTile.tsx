import { useTranslation } from 'react-i18next';

import { Badge, Btn, TileHead } from '@/components/dl/primitives';
import { formatPanelVersion } from '@/lib/panel-version';

export interface ToolsActions {
  onLogs: () => void;
  onXrayLogs?: () => void;
  onAmneziaWGLogs?: () => void;
  onConfig: () => void;
  onBackup: () => void;
  onHistory: () => void;
  onMetrics: () => void;
  onUpdate: () => void;
  onVersion: () => void;
}

// Advanced-mode tools that still live on the dashboard until Logs, Settings and Xray config take them.
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
          {t('pages.index.updatePanel')}
          {updateAvailable && <Badge acc>{formatPanelVersion(latestVersion)}</Badge>}
        </Btn>
        <Btn size="sm" icon="xray" onClick={actions.onVersion}>
          {t('pages.index.xraySwitch')}
        </Btn>
        <Btn size="sm" icon="logs" onClick={actions.onLogs}>
          {t('dashboard.tools.panelLogs')}
        </Btn>
        {actions.onXrayLogs && (
          <Btn size="sm" icon="logs" onClick={actions.onXrayLogs}>
            {t('dashboard.tools.xrayLogs')}
          </Btn>
        )}
        {actions.onAmneziaWGLogs && (
          <Btn size="sm" icon="logs" onClick={actions.onAmneziaWGLogs}>
            {t('dashboard.tools.awgLogs')}
          </Btn>
        )}
        <Btn size="sm" icon="code" onClick={actions.onConfig}>
          {t('pages.index.config')}
        </Btn>
        <Btn size="sm" icon="down" onClick={actions.onBackup}>
          {t('dashboard.tools.backup')}
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
