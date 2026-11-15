import { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Alert,
  Button,
  Card,
  ConfigProvider,
  Empty,
  Form,
  Input,
  InputNumber,
  Layout,
  Modal,
  Popconfirm,
  Result,
  Select,
  Space,
  Spin,
  Switch,
  Table,
  Tag,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { CopyOutlined, DeleteOutlined, ImportOutlined, PlusOutlined } from '@ant-design/icons';

import { useTheme } from '@/hooks/useTheme';
import { useInboundOptions } from '@/api/queries/useInboundOptions';
import {
  fetchSetupLink,
  useTunnelMutations,
  useTunnelStatus,
  useTunnelsQuery,
  type NewPortalValues,
  type ServerTunnel,
} from '@/api/queries/useTunnels';
import AppSidebar from '@/layouts/AppSidebar';
import { selectTagRender } from '@/components/form/SelectTag';
import { ClipboardManager } from '@/utils';
import { setMessageInstance } from '@/utils/messageBus';

const NAME_PATTERN = /^[a-z0-9][a-z0-9-]{0,30}$/;

function TunnelStatusCell({ tunnel }: { tunnel: ServerTunnel }) {
  const { t } = useTranslation();
  const isEntry = tunnel.role === 'portal';
  const { data, isLoading, error } = useTunnelStatus(tunnel.id, isEntry && tunnel.enable);

  if (!isEntry) {
    return <Typography.Text type="secondary">{t('pages.tunnels.statusExitOnly')}</Typography.Text>;
  }
  if (!tunnel.enable) return <Typography.Text type="secondary">—</Typography.Text>;
  if (isLoading) return <Spin size="small" />;
  if (!data?.connected) {
    return (
      <Tag color="error" title={data?.error || (error as Error | null)?.message}>
        {t('pages.tunnels.statusDown')}
      </Tag>
    );
  }
  const exit = [data.exitLoc, data.exitIp].filter(Boolean).join(' · ');
  return (
    <Space orientation="vertical" size={0}>
      <Tag color="success">{`${t('pages.tunnels.statusConnected')} · ${data.latencyMs} ms`}</Tag>
      {exit && (
        <Typography.Text type="secondary">{`${t('pages.tunnels.statusExit')} ${exit}`}</Typography.Text>
      )}
    </Space>
  );
}

export default function TunnelsPage() {
  const { t } = useTranslation();
  const { isDark, isUltra, antdThemeConfig } = useTheme();
  const [messageApi, messageContextHolder] = message.useMessage();
  useEffect(() => {
    setMessageInstance(messageApi);
  }, [messageApi]);

  const { tunnels, loading, fetched, fetchError, refetch } = useTunnelsQuery();
  const { create, importLink, setEnable, remove } = useTunnelMutations();
  const { data: inboundOptions = [] } = useInboundOptions();

  const [portalOpen, setPortalOpen] = useState(false);
  const [importOpen, setImportOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [portalForm] = Form.useForm<NewPortalValues>();
  const [importForm] = Form.useForm<{ link: string; exitOutbound?: string }>();
  const security = Form.useWatch('security', portalForm);

  const inboundTagOptions = useMemo(
    () =>
      inboundOptions
        .filter((inbound) => inbound.tag)
        .map((inbound) => ({
          value: inbound.tag as string,
          label: inbound.remark ? `${inbound.remark} (${inbound.tag})` : (inbound.tag as string),
        })),
    [inboundOptions],
  );

  const report = (msg: { success?: boolean; msg?: string } | null | undefined) => {
    if (msg?.success) return true;
    messageApi.error(msg?.msg || t('somethingWentWrong'));
    return false;
  };

  const submitPortal = async () => {
    const values = await portalForm.validateFields();
    setSaving(true);
    try {
      const ok = report(
        await create({
          name: values.name,
          address: values.address.trim(),
          port: values.port,
          security: values.security,
          serverName: values.security === 'reality' ? (values.serverName ?? '').trim() : '',
          routeInbounds: values.routeInbounds ?? [],
        }),
      );
      if (ok) {
        setPortalOpen(false);
        portalForm.resetFields();
      }
    } finally {
      setSaving(false);
    }
  };

  const submitImport = async () => {
    const values = await importForm.validateFields();
    setSaving(true);
    try {
      const ok = report(
        await importLink({
          link: values.link.trim(),
          exitOutbound: (values.exitOutbound ?? '').trim(),
        }),
      );
      if (ok) {
        setImportOpen(false);
        importForm.resetFields();
      }
    } finally {
      setSaving(false);
    }
  };

  const copyLink = async (id: number) => {
    try {
      const link = await fetchSetupLink(id);
      if (await ClipboardManager.copyText(link)) messageApi.success(t('pages.tunnels.linkCopied'));
    } catch {
      messageApi.error(t('somethingWentWrong'));
    }
  };

  const columns: ColumnsType<ServerTunnel> = [
    {
      title: t('pages.tunnels.enabled'),
      dataIndex: 'enable',
      width: 90,
      render: (enable: boolean, row) => (
        <Switch
          size="small"
          checked={enable}
          aria-label={`${t('pages.tunnels.enabled')} ${row.name}`}
          onChange={async (checked) => report(await setEnable({ id: row.id, enable: checked }))}
        />
      ),
    },
    {
      title: t('pages.tunnels.name'),
      dataIndex: 'name',
      render: (name: string) => <Typography.Text strong>{name}</Typography.Text>,
    },
    {
      title: t('pages.tunnels.role'),
      dataIndex: 'role',
      render: (role: string) =>
        role === 'portal' ? (
          <Tag color="blue">{t('pages.tunnels.rolePortal')}</Tag>
        ) : (
          <Tag color="green">{t('pages.tunnels.roleBridge')}</Tag>
        ),
    },
    {
      title: t('pages.tunnels.status'),
      render: (_, row) => <TunnelStatusCell tunnel={row} />,
    },
    {
      title: t('pages.tunnels.address'),
      render: (_, row) => <Typography.Text code>{`${row.address}:${row.port}`}</Typography.Text>,
    },
    {
      title: t('pages.tunnels.security'),
      dataIndex: 'security',
      render: (value: string) =>
        value === 'reality' ? (
          <Tag>{t('pages.tunnels.securityReality')}</Tag>
        ) : (
          <Tag color="warning">{t('pages.tunnels.securityNone')}</Tag>
        ),
    },
    {
      title: t('pages.tunnels.routeInbounds'),
      render: (_, row) =>
        row.role === 'portal'
          ? (row.routeInbounds ?? []).map((tag) => <Tag key={tag}>{tag}</Tag>)
          : row.exitOutbound || 'direct',
    },
    {
      width: 110,
      render: (_, row) => (
        <Space>
          {row.role === 'portal' && (
            <Button
              type="text"
              icon={<CopyOutlined aria-hidden />}
              aria-label={t('pages.tunnels.copyLink')}
              title={t('pages.tunnels.copyLink')}
              onClick={() => copyLink(row.id)}
            />
          )}
          <Popconfirm
            title={t('pages.tunnels.deleteConfirm')}
            onConfirm={async () => report(await remove(row.id))}
          >
            <Button
              type="text"
              danger
              icon={<DeleteOutlined aria-hidden />}
              aria-label={`${t('delete')} ${row.name}`}
            />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const pageClass = ['tunnels-page', isDark ? 'is-dark' : '', isUltra ? 'is-ultra' : '']
    .filter(Boolean)
    .join(' ');

  return (
    <ConfigProvider theme={antdThemeConfig}>
      {messageContextHolder}
      <Layout className={pageClass}>
        <AppSidebar />
        <Layout className="content-shell">
          <Layout.Content id="content-layout" className="content-area">
            <Spin aria-live="off" spinning={!fetched} delay={200} size="large">
              {!fetched ? (
                <div className="loading-spacer" />
              ) : fetchError ? (
                <Result
                  status="error"
                  title={t('somethingWentWrong')}
                  subTitle={fetchError}
                  extra={
                    <Button type="primary" loading={loading} onClick={() => refetch()}>
                      {t('refresh')}
                    </Button>
                  }
                />
              ) : (
                <Card
                  title={t('pages.tunnels.title')}
                  extra={
                    <Space wrap>
                      <Button
                        icon={<ImportOutlined aria-hidden />}
                        onClick={() => setImportOpen(true)}
                      >
                        {t('pages.tunnels.importLink')}
                      </Button>
                      <Button
                        type="primary"
                        icon={<PlusOutlined aria-hidden />}
                        onClick={() => setPortalOpen(true)}
                      >
                        {t('pages.tunnels.newPortal')}
                      </Button>
                    </Space>
                  }
                >
                  <Typography.Paragraph type="secondary">
                    {t('pages.tunnels.subtitle')}
                  </Typography.Paragraph>
                  {tunnels.some((tunnel) => tunnel.role === 'portal') && (
                    <Alert
                      type="info"
                      showIcon
                      title={t('pages.tunnels.linkHint')}
                      style={{ marginBottom: 12 }}
                    />
                  )}
                  <Table<ServerTunnel>
                    rowKey="id"
                    size="small"
                    loading={loading}
                    columns={columns}
                    dataSource={tunnels}
                    pagination={false}
                    scroll={{ x: 'max-content' }}
                    locale={{ emptyText: <Empty description={t('pages.tunnels.empty')} /> }}
                  />
                </Card>
              )}
            </Spin>
          </Layout.Content>
        </Layout>
      </Layout>

      <Modal
        title={t('pages.tunnels.newPortal')}
        open={portalOpen}
        confirmLoading={saving}
        onOk={submitPortal}
        onCancel={() => setPortalOpen(false)}
        destroyOnHidden
      >
        <Form
          form={portalForm}
          layout="vertical"
          initialValues={{ security: 'reality', port: 8443, routeInbounds: [] }}
        >
          <Form.Item
            name="name"
            label={t('pages.tunnels.name')}
            rules={[{ required: true, pattern: NAME_PATTERN }]}
          >
            <Input placeholder="de1" autoComplete="off" />
          </Form.Item>
          <Form.Item
            name="address"
            label={t('pages.tunnels.address')}
            extra={t('pages.tunnels.addressHint')}
            rules={[{ required: true, whitespace: true }]}
          >
            <Input placeholder="203.0.113.10" autoComplete="off" />
          </Form.Item>
          <Form.Item
            name="port"
            label={t('pages.tunnels.port')}
            rules={[{ required: true, type: 'number', min: 1, max: 65535 }]}
          >
            <InputNumber min={1} max={65535} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="security" label={t('pages.tunnels.security')}>
            <Select
              options={[
                { value: 'reality', label: t('pages.tunnels.securityReality') },
                { value: 'none', label: t('pages.tunnels.securityNone') },
              ]}
            />
          </Form.Item>
          {security !== 'none' && (
            <Form.Item
              name="serverName"
              label={t('pages.tunnels.serverName')}
              extra={t('pages.tunnels.serverNameHint')}
              rules={[{ required: true, whitespace: true }]}
            >
              <Input placeholder="www.example.com" autoComplete="off" />
            </Form.Item>
          )}
          <Form.Item
            name="routeInbounds"
            label={t('pages.tunnels.routeInbounds')}
            extra={t('pages.tunnels.routeInboundsHint')}
          >
            <Select
              mode="multiple"
              options={inboundTagOptions}
              tagRender={selectTagRender}
              allowClear
            />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={t('pages.tunnels.importLink')}
        open={importOpen}
        confirmLoading={saving}
        onOk={submitImport}
        onCancel={() => setImportOpen(false)}
        destroyOnHidden
      >
        <Form form={importForm} layout="vertical">
          <Form.Item
            name="link"
            label={t('pages.tunnels.link')}
            rules={[{ required: true, whitespace: true }]}
          >
            <Input.TextArea
              rows={4}
              placeholder={t('pages.tunnels.pasteLink')}
              autoComplete="off"
            />
          </Form.Item>
          <Form.Item
            name="exitOutbound"
            label={t('pages.tunnels.exitOutbound')}
            extra={t('pages.tunnels.exitOutboundHint')}
          >
            <Input placeholder="direct" autoComplete="off" />
          </Form.Item>
        </Form>
      </Modal>
    </ConfigProvider>
  );
}
