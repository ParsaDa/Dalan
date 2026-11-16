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
  Radio,
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
import PortTest from './PortTest';
import TunnelPools from './TunnelPools';
import { useNodesQuery } from '@/api/queries/useNodesQuery';
import { useInboundOptions } from '@/api/queries/useInboundOptions';
import {
  fetchSetupLink,
  useTunnelEngine,
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
const BACKPACK_TRANSPORTS = [
  'tcp',
  'tcpmux',
  'stealth',
  'ws',
  'wss',
  'wsmux',
  'wssmux',
  'kcp',
  'quic',
  'udp',
  'xdi',
  'pck',
].map((value) => ({ value, label: value }));

interface PortalFormValues {
  name: string;
  address: string;
  port: number;
  engine: 'xray' | 'backpack';
  security: 'reality' | 'none';
  serverName?: string;
  routeInbounds?: string[];
  nodeId?: number;
  transport: string;
  fallbackTransports?: string[];
  forwardPorts?: string[];
  acceptUdp?: boolean;
}

function TunnelStatusCell({ tunnel }: { tunnel: ServerTunnel }) {
  const { t } = useTranslation();
  // Xray tunnels are tested from the entry end; a BackPack engine reports its own state on both ends.
  const testable = tunnel.role === 'portal' || tunnel.engine === 'backpack';
  const { data, isLoading, error } = useTunnelStatus(tunnel.id, testable && tunnel.enable);

  if (!testable) {
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
      <Tag color="success">
        {data.latencyMs > 0
          ? `${t('pages.tunnels.statusConnected')} · ${data.latencyMs} ms`
          : t('pages.tunnels.statusConnected')}
      </Tag>
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
  const [modal, modalContextHolder] = Modal.useModal();
  useEffect(() => {
    setMessageInstance(messageApi);
  }, [messageApi]);

  const { tunnels, loading, fetched, fetchError, refetch } = useTunnelsQuery();
  const { create, importLink, setEnable, remove, installEngine, installingEngine } =
    useTunnelMutations();
  const { data: engineInfo } = useTunnelEngine();
  const { data: inboundOptions = [] } = useInboundOptions();
  const { nodes } = useNodesQuery();

  const [portalOpen, setPortalOpen] = useState(false);
  const [importOpen, setImportOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [portalForm] = Form.useForm<PortalFormValues>();
  const [importForm] = Form.useForm<{ link: string; exitOutbound?: string }>();
  const security = Form.useWatch('security', portalForm);
  const engine = Form.useWatch('engine', portalForm);
  const chosenNode = Form.useWatch('nodeId', portalForm);
  const chosenAddress = Form.useWatch('address', portalForm);
  const usesBackpack = tunnels.some((tunnel) => tunnel.engine === 'backpack');

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

  const removeTunnel = async (row: ServerTunnel) => {
    const msg = await remove({ id: row.id });
    if (msg?.success || !row.nodeId) {
      report(msg);
      return;
    }
    modal.confirm({
      title: msg?.msg || t('somethingWentWrong'),
      content: t('pages.tunnels.removeHereHint'),
      okText: t('pages.tunnels.removeHere'),
      okType: 'danger',
      cancelText: t('cancel'),
      onOk: async () => report(await remove({ id: row.id, here: true })),
    });
  };

  const submitPortal = async () => {
    const values = await portalForm.validateFields();
    setSaving(true);
    try {
      const common = {
        name: values.name,
        address: values.address.trim(),
        port: values.port,
        ...(values.nodeId ? { nodeId: values.nodeId } : {}),
      };
      const request: NewPortalValues =
        values.engine === 'backpack'
          ? {
              ...common,
              engine: 'backpack',
              transport: values.transport,
              fallbackTransports: values.fallbackTransports ?? [],
              forwardPorts: values.forwardPorts ?? [],
              acceptUdp: values.acceptUdp ?? false,
              routeInbounds: values.routeInbounds ?? [],
            }
          : {
              ...common,
              engine: 'xray',
              security: values.security,
              serverName: values.security === 'reality' ? (values.serverName ?? '').trim() : '',
              routeInbounds: values.routeInbounds ?? [],
            };
      const ok = report(await create(request));
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
      render: (name: string, row) => (
        <Space>
          <Typography.Text strong>{name}</Typography.Text>
          {row.engine === 'backpack' && <Tag>{t('pages.tunnels.engineBackpack')}</Tag>}
        </Space>
      ),
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
      render: (value: string, row) =>
        row.engine === 'backpack' ? (
          <Tag>{row.transport}</Tag>
        ) : value === 'reality' ? (
          <Tag>{t('pages.tunnels.securityReality')}</Tag>
        ) : (
          <Tag color="warning">{t('pages.tunnels.securityNone')}</Tag>
        ),
    },
    {
      title: t('pages.tunnels.routeInbounds'),
      render: (_, row) =>
        row.role !== 'portal'
          ? row.exitOutbound || 'direct'
          : [...(row.engine === 'backpack' ? row.forwardPorts : []), ...row.routeInbounds].map(
              (item) => <Tag key={item}>{item}</Tag>,
            ),
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
          <Popconfirm title={t('pages.tunnels.deleteConfirm')} onConfirm={() => removeTunnel(row)}>
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
      {modalContextHolder}
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
                  {usesBackpack && engineInfo && !engineInfo.installed && (
                    <Alert
                      type="warning"
                      showIcon
                      title={t('pages.tunnels.engineMissing')}
                      description={t('pages.tunnels.engineMissingHint')}
                      action={
                        <Button
                          size="small"
                          loading={installingEngine}
                          onClick={async () => report(await installEngine())}
                        >
                          {t('pages.tunnels.installEngine')}
                        </Button>
                      }
                      style={{ marginBottom: 12 }}
                    />
                  )}
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
              {fetched && !fetchError && (
                <div style={{ marginTop: 16 }}>
                  <TunnelPools tunnels={tunnels} />
                </div>
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
          initialValues={{
            engine: 'xray',
            security: 'reality',
            port: 8443,
            routeInbounds: [],
            transport: 'tcpmux',
            fallbackTransports: [],
            forwardPorts: [],
            acceptUdp: false,
          }}
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
          <Form.Item name="engine" label={t('pages.tunnels.engine')}>
            <Radio.Group
              options={[
                { value: 'xray', label: t('pages.tunnels.engineXray') },
                { value: 'backpack', label: t('pages.tunnels.engineBackpack') },
              ]}
            />
          </Form.Item>
          {nodes.length > 0 && (
            <Form.Item
              name="nodeId"
              label={t('pages.tunnels.nodeEnd')}
              extra={t('pages.tunnels.nodeEndHint')}
            >
              <Select
                allowClear
                options={nodes
                  .filter((node) => node.enable)
                  .map((node) => ({ value: node.id, label: node.name }))}
              />
            </Form.Item>
          )}
          {chosenNode ? (
            <PortTest
              nodeId={chosenNode}
              address={chosenAddress ?? ''}
              onPick={(port) => portalForm.setFieldValue('port', port)}
            />
          ) : null}
          {engine === 'backpack' ? (
            <>
              <Form.Item name="transport" label={t('pages.tunnels.transport')}>
                <Select options={BACKPACK_TRANSPORTS} />
              </Form.Item>
              <Form.Item
                name="fallbackTransports"
                label={t('pages.tunnels.fallbackTransports')}
                extra={t('pages.tunnels.fallbackTransportsHint')}
              >
                <Select
                  mode="multiple"
                  options={BACKPACK_TRANSPORTS}
                  tagRender={selectTagRender}
                  allowClear
                />
              </Form.Item>
              <Form.Item
                name="forwardPorts"
                label={t('pages.tunnels.forwardPorts')}
                extra={t('pages.tunnels.forwardPortsHint')}
              >
                <Select
                  mode="tags"
                  open={false}
                  tagRender={selectTagRender}
                  tokenSeparators={[' ', ',']}
                />
              </Form.Item>
              <Form.Item
                name="acceptUdp"
                label={t('pages.tunnels.acceptUdp')}
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
            </>
          ) : (
            <>
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
            </>
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
