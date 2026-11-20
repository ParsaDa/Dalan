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
import {
  CopyOutlined,
  DeleteOutlined,
  EditOutlined,
  ImportOutlined,
  PlusOutlined,
} from '@ant-design/icons';

import { useTheme } from '@/hooks/useTheme';
import PortTest from './PortTest';
import TunnelEditModal from './TunnelEditModal';
import TunnelEngineBar from './TunnelEngineBar';
import TunnelOptionFields, { TunnelOptionField, cleanOptions } from './TunnelOptionFields';
import TunnelPools from './TunnelPools';
import { BACKPACK_TRANSPORTS } from './backpackTransports';
import { RAKHSH_CARRIERS, isRawCarrier, needsPeerIp } from './rakhshCarriers';
import { useNodesQuery } from '@/api/queries/useNodesQuery';
import { useInboundOptions } from '@/api/queries/useInboundOptions';
import {
  fetchSetupLink,
  useTunnelMutations,
  useTunnelOptions,
  useTunnelStatus,
  useTunnelsQuery,
  type NewPortalValues,
  type ServerTunnel,
  type TunnelEditValues,
  type TunnelMode,
  type TunnelOptionValues,
} from '@/api/queries/useTunnels';
import AppSidebar from '@/layouts/AppSidebar';
import { selectTagRender } from '@/components/form/SelectTag';
import { ClipboardManager, SizeFormatter } from '@/utils';
import { setMessageInstance } from '@/utils/messageBus';

const NAME_PATTERN = /^[a-z0-9][a-z0-9-]{0,30}$/;

const RAKHSH_FORWARD_PLACEHOLDER = '443=127.0.0.1:2096';

interface PortalFormValues {
  name: string;
  address: string;
  port: number;
  engine: 'backpack' | 'rakhsh';
  routeInbounds?: string[];
  nodeId?: number;
  transport: string;
  fallbackTransports?: string[];
  forwardPorts?: string[];
  acceptUdp?: boolean;
  mode: TunnelMode;
  options?: TunnelOptionValues;
}

function TunnelStatusCell({ tunnel }: { tunnel: ServerTunnel }) {
  const { t } = useTranslation();
  // Management riders are tested through their node's status; every other tunnel reports its own state.
  const testable = !tunnel.managePort;
  const { data, isLoading, error } = useTunnelStatus(tunnel.id, testable && tunnel.enable);

  if (!testable) {
    return <Typography.Text type="secondary">—</Typography.Text>;
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
  const lanes = data.targetLanes
    ? t('pages.tunnels.statusLanes', { up: data.lanesUp ?? 0, total: data.targetLanes })
    : '';
  const loss = data.lossPct ? t('pages.tunnels.statusLoss', { pct: data.lossPct }) : '';
  const connected = [
    t('pages.tunnels.statusConnected'),
    data.latencyMs > 0 ? `${data.latencyMs} ms` : '',
    lanes,
    loss,
  ];
  return (
    <Space orientation="vertical" size={0}>
      <Tag color="success">{connected.filter(Boolean).join(' · ')}</Tag>
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
  const { create, importLink, update, setEnable, remove } = useTunnelMutations();
  const [editing, setEditing] = useState<ServerTunnel | null>(null);
  const { data: inboundOptions = [] } = useInboundOptions();
  const { nodes } = useNodesQuery();

  const [engineNode, setEngineNode] = useState<number>();
  const [portalOpen, setPortalOpen] = useState(false);
  const [importOpen, setImportOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [portalForm] = Form.useForm<PortalFormValues>();
  const [importForm] = Form.useForm<{ link: string; exitOutbound?: string }>();
  const engine = Form.useWatch('engine', portalForm);
  const mode = Form.useWatch('mode', portalForm) ?? 'reverse';
  const chosenNode = Form.useWatch('nodeId', portalForm);
  const chosenAddress = Form.useWatch('address', portalForm);
  const chosenTransport = Form.useWatch('transport', portalForm);
  const chosenLinks = Form.useWatch(['options', 'links'], portalForm) as
    | { carrier?: string }[]
    | undefined;
  const usesBackpack = tunnels.some((tunnel) => tunnel.engine === 'backpack');
  const usesRakhsh = tunnels.some((tunnel) => tunnel.engine === 'rakhsh');
  const { data: backpackCatalog = [] } = useTunnelOptions('backpack');
  const { data: rakhshCatalog = [] } = useTunnelOptions(
    'rakhsh',
    usesRakhsh || engine === 'rakhsh',
  );
  const direct = engine === 'backpack' && mode === 'direct';
  // A direct tunnel's entry end dials the exit end, which is where the node (if any) is.
  const dialsExit = mode === 'direct' && (engine === 'backpack' || engine === 'rakhsh');
  const carrierOption = backpackCatalog.find((o) => o.mode === 'direct' && o.key === 'carrier');
  const rakhshNeedsPeerIp = needsPeerIp([
    chosenTransport,
    ...(chosenLinks ?? []).map((l) => l.carrier),
  ]);

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

  const engineNodePicker = (
    <Select
      size="small"
      allowClear
      aria-label={t('pages.tunnels.engineViaNode')}
      placeholder={t('pages.tunnels.engineViaNode')}
      style={{ minWidth: 200 }}
      value={engineNode}
      onChange={setEngineNode}
      options={nodes
        .filter((node) => node.enable)
        .map((node) => ({ value: node.id, label: node.name }))}
      tagRender={selectTagRender}
    />
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
    const values = await portalForm.validateFields().catch(() => null);
    if (!values) return;
    setSaving(true);
    try {
      const common = {
        name: values.name,
        address: (values.address ?? '').trim(),
        port: values.port,
        ...(values.nodeId ? { nodeId: values.nodeId } : {}),
      };
      const request: NewPortalValues =
        values.engine === 'rakhsh'
          ? {
              ...common,
              engine: 'rakhsh',
              transport: values.transport,
              forwardPorts: values.forwardPorts ?? [],
              routeInbounds: values.routeInbounds ?? [],
              mode: values.mode,
              options: cleanOptions(values.options),
            }
          : {
              ...common,
              engine: 'backpack',
              transport: direct ? '' : values.transport,
              fallbackTransports: direct ? [] : (values.fallbackTransports ?? []),
              forwardPorts: values.forwardPorts ?? [],
              acceptUdp: values.acceptUdp ?? false,
              routeInbounds: direct ? [] : (values.routeInbounds ?? []),
              mode: values.mode,
              options: cleanOptions(values.options),
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

  const saveEdit = async (id: number, values: TunnelEditValues) =>
    report(await update({ id, values }));

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
          {row.engine === 'rakhsh' && <Tag>{t('pages.tunnels.engineRakhsh')}</Tag>}
          {!row.managePort && (
            <Tag color={row.mode === 'direct' ? 'purple' : 'cyan'}>
              {row.mode === 'direct'
                ? t('pages.tunnels.modeDirect')
                : t('pages.tunnels.modeReverse')}
            </Tag>
          )}
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
      title: t('pages.tunnels.traffic'),
      render: (_, row) =>
        row.role === 'portal' &&
        !row.managePort &&
        (row.mode !== 'direct' || row.engine === 'rakhsh')
          ? `↑ ${SizeFormatter.sizeFormat(row.up)} · ↓ ${SizeFormatter.sizeFormat(row.down)}`
          : '',
    },
    {
      title: t('pages.tunnels.address'),
      render: (_, row) => <Typography.Text code>{`${row.address}:${row.port}`}</Typography.Text>,
    },
    {
      title: t('pages.tunnels.transport'),
      dataIndex: 'transport',
      render: (_, row) => <Tag>{row.transport}</Tag>,
    },
    {
      title: t('pages.tunnels.routeInbounds'),
      render: (_, row) =>
        row.role !== 'portal'
          ? row.exitOutbound || 'direct'
          : [
              ...(row.engine === 'backpack' || row.engine === 'rakhsh' ? row.forwardPorts : []),
              ...row.routeInbounds,
            ].map((item) => <Tag key={item}>{item}</Tag>),
    },
    {
      width: 150,
      render: (_, row) => (
        <Space>
          {!row.managePort && (
            <Button
              type="text"
              icon={<EditOutlined aria-hidden />}
              aria-label={`${t('edit')} ${row.name}`}
              title={t('edit')}
              onClick={() => setEditing(row)}
            />
          )}
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
                  {(usesBackpack || usesRakhsh) && (
                    <div style={{ marginBottom: 12 }}>{engineNodePicker}</div>
                  )}
                  {usesBackpack && (
                    <TunnelEngineBar
                      engine="backpack"
                      label={t('pages.tunnels.engineBackpack')}
                      nodeId={engineNode}
                      report={report}
                    />
                  )}
                  {usesRakhsh && (
                    <TunnelEngineBar
                      engine="rakhsh"
                      label={t('pages.tunnels.engineRakhsh')}
                      nodeId={engineNode}
                      report={report}
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
          onValuesChange={(changed: Partial<PortalFormValues>) => {
            // Each engine reads its own settings, so the ones typed for the other engine must not travel along.
            if (!changed.engine) return;
            // setFieldsValue merges objects, so only setFieldValue can empty the options.
            portalForm.setFieldValue('options', {});
            portalForm.setFieldValue('transport', changed.engine === 'rakhsh' ? 'tcp' : 'tcpmux');
          }}
          initialValues={{
            engine: 'backpack',
            mode: 'reverse',
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
            extra={
              dialsExit ? t('pages.tunnels.directAddressHint') : t('pages.tunnels.addressHint')
            }
            rules={[{ required: !(dialsExit && chosenNode), whitespace: true }]}
          >
            <Input placeholder="203.0.113.10" autoComplete="off" />
          </Form.Item>
          <Form.Item
            name="port"
            label={t('pages.tunnels.port')}
            extra={dialsExit ? t('pages.tunnels.directPortHint') : undefined}
            rules={[{ required: true, type: 'number', min: 1, max: 65535 }]}
          >
            <InputNumber min={1} max={65535} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="engine" label={t('pages.tunnels.engine')}>
            <Radio.Group
              options={[
                { value: 'backpack', label: t('pages.tunnels.engineBackpack') },
                { value: 'rakhsh', label: t('pages.tunnels.engineRakhsh') },
              ]}
            />
          </Form.Item>
          {(engine === 'backpack' || engine === 'rakhsh') && (
            <Form.Item
              name="mode"
              label={t('pages.tunnels.mode')}
              extra={
                mode === 'direct'
                  ? engine === 'rakhsh'
                    ? t('pages.tunnels.modeDirectHintRakhsh')
                    : t('pages.tunnels.modeDirectHint')
                  : t('pages.tunnels.modeReverseHint')
              }
            >
              <Radio.Group
                options={[
                  { value: 'reverse', label: t('pages.tunnels.modeReverse') },
                  { value: 'direct', label: t('pages.tunnels.modeDirect') },
                ]}
              />
            </Form.Item>
          )}
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
          {chosenNode && !dialsExit ? (
            <PortTest
              nodeId={chosenNode}
              address={chosenAddress ?? ''}
              onPick={(port) => portalForm.setFieldValue('port', port)}
            />
          ) : null}
          {engine === 'rakhsh' ? (
            <>
              <Form.Item name="transport" label={t('pages.tunnels.transport')}>
                <Select options={RAKHSH_CARRIERS} />
              </Form.Item>
              {isRawCarrier(chosenTransport) && (
                <Alert
                  type="warning"
                  showIcon
                  title={t('pages.tunnels.rakhshRawHint')}
                  style={{ marginBottom: 12 }}
                />
              )}
              <Form.Item
                name="forwardPorts"
                label={t('pages.tunnels.forwardPorts')}
                extra={t('pages.tunnels.rakhshForwardPortsHint')}
              >
                <Select
                  mode="tags"
                  open={false}
                  tagRender={selectTagRender}
                  tokenSeparators={[' ', ',']}
                  placeholder={RAKHSH_FORWARD_PLACEHOLDER}
                />
              </Form.Item>
              <TunnelOptionFields
                options={rakhshCatalog}
                mode={mode}
                forRole="portal"
                pinned={rakhshNeedsPeerIp ? ['peer_ip'] : []}
              />
            </>
          ) : (
            <>
              {direct ? (
                carrierOption && <TunnelOptionField option={carrierOption} required />
              ) : (
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
                </>
              )}
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
              <TunnelOptionFields
                options={backpackCatalog}
                mode={mode}
                skip={direct ? ['carrier', 'local_ip', 'peer_ip'] : []}
              />
            </>
          )}
          {!direct && (
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
          )}
        </Form>
      </Modal>

      {editing && (
        <TunnelEditModal
          key={editing.id}
          tunnel={editing}
          options={editing.engine === 'rakhsh' ? rakhshCatalog : backpackCatalog}
          transports={editing.engine === 'rakhsh' ? RAKHSH_CARRIERS : BACKPACK_TRANSPORTS}
          inboundTagOptions={inboundTagOptions}
          onSubmit={saveEdit}
          onClose={() => setEditing(null)}
        />
      )}

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
