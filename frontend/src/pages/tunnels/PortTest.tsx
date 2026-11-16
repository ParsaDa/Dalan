import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Space, Tag, message } from 'antd';

import { useRouteTest, type PortResult } from '@/api/queries/useRouteTest';

interface Props {
  nodeId: number;
  address: string;
  onPick: (port: number) => void;
}

export default function PortTest({ nodeId, address, onPick }: Props) {
  const { t } = useTranslation();
  const { run, testing } = useRouteTest();
  const [results, setResults] = useState<PortResult[]>([]);
  const [messageApi, messageContextHolder] = message.useMessage();

  const test = async () => {
    try {
      setResults(await run({ nodeId, address: address.trim() }));
    } catch (error) {
      messageApi.error((error as Error).message);
    }
  };

  return (
    <div style={{ marginBottom: 16 }}>
      {messageContextHolder}
      <Button loading={testing} disabled={!address.trim()} onClick={test}>
        {t('pages.tunnels.testPorts')}
      </Button>
      <Space wrap style={{ marginInlineStart: 8 }}>
        {results.map((row) =>
          row.reachable ? (
            <Button key={row.port} size="small" type="dashed" onClick={() => onPick(row.port)}>
              {`${row.port} · ${row.latencyMs} ms`}
            </Button>
          ) : (
            <Tag key={row.port} color="red">{`${row.port}: ${row.error}`}</Tag>
          ),
        )}
      </Space>
    </div>
  );
}
