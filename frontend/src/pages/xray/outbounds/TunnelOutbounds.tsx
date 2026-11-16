import { useTranslation } from 'react-i18next';
import { Table, Tag } from 'antd';
import type { ColumnsType } from 'antd/es/table';

import { SizeFormatter } from '@/utils';
import type { OutboundTrafficRow } from '@/hooks/useXraySetting';
import type { TunnelOutbound } from '@/schemas/xray';

interface TunnelOutboundsProps {
  tunnelOutbounds: TunnelOutbound[];
  outboundsTraffic: OutboundTrafficRow[];
}

// Read-only: tunnels are managed on the Tunnels page and added to the config at runtime.
export default function TunnelOutbounds({
  tunnelOutbounds,
  outboundsTraffic,
}: TunnelOutboundsProps) {
  const { t } = useTranslation();
  if (tunnelOutbounds.length === 0) return null;

  const columns: ColumnsType<TunnelOutbound> = [
    {
      title: t('pages.xray.outbound.tag'),
      dataIndex: 'tag',
      key: 'tag',
      render: (tag: string, record) => (
        <>
          <span className="tag-name">{tag}</span> <Tag color="green">{record.engine}</Tag>
        </>
      ),
    },
    {
      title: t('pages.inbounds.traffic'),
      key: 'traffic',
      render: (_: unknown, record) => {
        const row = outboundsTraffic.find((r) => r.tag === record.tag);
        return (
          <>
            <span className="traffic-up">↑ {SizeFormatter.sizeFormat(row?.up ?? 0)}</span>
            <span className="traffic-sep" />
            <span className="traffic-down">↓ {SizeFormatter.sizeFormat(row?.down ?? 0)}</span>
          </>
        );
      },
    },
  ];

  return (
    <div className="subscription-outbounds" style={{ marginTop: 16 }}>
      <div className="subscription-outbounds-head">
        <div className="subscription-outbounds-title">
          {t('pages.xray.outboundSub.tunnelsTitle')}
        </div>
        <div className="subscription-outbounds-desc">{t('pages.xray.outboundSub.tunnelsDesc')}</div>
      </div>
      <Table<TunnelOutbound>
        rowKey="tag"
        columns={columns}
        dataSource={tunnelOutbounds}
        pagination={false}
        size="small"
      />
    </div>
  );
}
