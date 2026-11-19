import { useTranslation } from 'react-i18next';
import { Tag } from 'antd';

export interface TorStatusInfo {
  state: string;
  latencyMs: number;
  error: string;
  progress?: number;
}

// Says whether Tor is carrying traffic through an exit; anything else is "not connected" with its reason.
export default function TorStatusChip({ status }: { status?: TorStatusInfo }) {
  const { t } = useTranslation();
  if (!status) return <span className="empty">—</span>;
  if (status.state === 'connected') {
    return (
      <Tag color="success" className="tor-status">
        {t('pages.tor.status.connected')} · {status.latencyMs} ms
      </Tag>
    );
  }
  if (status.state === 'starting') {
    return (
      <Tag color="processing" className="tor-status">
        {t('pages.tor.status.starting')} {status.progress ?? 0}%
      </Tag>
    );
  }
  const reason =
    status.state === 'not_installed'
      ? t('pages.tor.reason.notInstalled')
      : status.state === 'stopped'
        ? t('pages.tor.reason.stopped')
        : status.error || t('pages.tor.reason.failed');
  return (
    <div className="tor-status-stack">
      <Tag color="error" className="tor-status">
        {t('pages.tor.status.notConnected')}
      </Tag>
      <span className="tor-reason">{reason}</span>
    </div>
  );
}
