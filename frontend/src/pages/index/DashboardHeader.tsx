import { useTranslation } from 'react-i18next';

interface DashboardHeaderProps {
  online: number;
  total: number;
  hour: number;
}

export function greetingKey(hour: number): string {
  if (hour >= 5 && hour < 12) return 'pages.index.greetMorning';
  if (hour >= 12 && hour < 18) return 'pages.index.greetAfternoon';
  return 'pages.index.greetEvening';
}

export default function DashboardHeader({ online, total, hour }: DashboardHeaderProps) {
  const { t } = useTranslation();
  return (
    <header className="ov-head">
      <h1 className="ov-greeting">{t(greetingKey(hour))}</h1>
      {total > 0 && (
        <p className="ov-status-line">{t('pages.index.nodesStatusLine', { online, total })}</p>
      )}
    </header>
  );
}
