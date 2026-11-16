import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Card,
  Empty,
  Form,
  Input,
  Modal,
  Popconfirm,
  Select,
  Table,
  Tag,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { DeleteOutlined, PlusOutlined } from '@ant-design/icons';

import { useInboundOptions } from '@/api/queries/useInboundOptions';
import {
  useTunnelPoolMutations,
  useTunnelPoolsQuery,
  type NewPoolValues,
  type TunnelPool,
} from '@/api/queries/useTunnelPools';
import type { ServerTunnel } from '@/api/queries/useTunnels';
import { selectTagRender } from '@/components/form/SelectTag';

interface Props {
  tunnels: ServerTunnel[];
}

export default function TunnelPools({ tunnels }: Props) {
  const { t } = useTranslation();
  const [messageApi, messageContextHolder] = message.useMessage();
  const { pools, loading } = useTunnelPoolsQuery();
  const { create, remove } = useTunnelPoolMutations();
  const { data: inboundOptions = [] } = useInboundOptions();
  const [open, setOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [form] = Form.useForm<NewPoolValues>();

  const entryTunnels = useMemo(
    () => tunnels.filter((row) => row.role === 'portal' && !row.managePort),
    [tunnels],
  );
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

  const submit = async () => {
    const values = await form.validateFields();
    setSaving(true);
    try {
      const ok = report(
        await create({
          name: values.name.trim(),
          members: values.members,
          routeInbounds: values.routeInbounds ?? [],
        }),
      );
      if (ok) setOpen(false);
    } finally {
      setSaving(false);
    }
  };

  const columns: ColumnsType<TunnelPool> = [
    { title: t('pages.tunnels.name'), dataIndex: 'name' },
    {
      title: t('pages.tunnels.poolMembers'),
      render: (_, row) =>
        row.members.map((member) =>
          member === row.active ? (
            <Tag key={member} color="green">{`${member} · ${t('pages.tunnels.poolInUse')}`}</Tag>
          ) : (
            <Tag key={member}>{member}</Tag>
          ),
        ),
    },
    {
      title: t('pages.tunnels.routeInbounds'),
      render: (_, row) => row.routeInbounds.map((tag) => <Tag key={tag}>{tag}</Tag>),
    },
    {
      width: 60,
      render: (_, row) => (
        <Popconfirm
          title={t('pages.tunnels.poolDeleteConfirm')}
          onConfirm={async () => report(await remove(row.id))}
        >
          <Button
            type="text"
            danger
            icon={<DeleteOutlined aria-hidden />}
            aria-label={`${t('delete')} ${row.name}`}
          />
        </Popconfirm>
      ),
    },
  ];

  return (
    <>
      {messageContextHolder}
      <Card
        title={t('pages.tunnels.pools')}
        extra={
          <Button
            icon={<PlusOutlined aria-hidden />}
            disabled={entryTunnels.length < 2}
            onClick={() => {
              form.resetFields();
              setOpen(true);
            }}
          >
            {t('pages.tunnels.newPool')}
          </Button>
        }
      >
        <Typography.Paragraph type="secondary">{t('pages.tunnels.poolsHint')}</Typography.Paragraph>
        <Table<TunnelPool>
          rowKey="id"
          size="small"
          loading={loading}
          columns={columns}
          dataSource={pools}
          pagination={false}
          scroll={{ x: 'max-content' }}
          locale={{ emptyText: <Empty description={t('pages.tunnels.poolsEmpty')} /> }}
        />
      </Card>
      <Modal
        title={t('pages.tunnels.newPool')}
        open={open}
        confirmLoading={saving}
        onOk={submit}
        onCancel={() => setOpen(false)}
        destroyOnHidden
      >
        <Form form={form} layout="vertical" initialValues={{ members: [], routeInbounds: [] }}>
          <Form.Item
            name="name"
            label={t('pages.tunnels.name')}
            rules={[{ required: true, message: t('pages.enrolments.required') }]}
          >
            <Input placeholder="main" autoComplete="off" />
          </Form.Item>
          <Form.Item
            name="members"
            label={t('pages.tunnels.poolTunnels')}
            extra={t('pages.tunnels.poolTunnelsHint')}
            rules={[
              { required: true, type: 'array', min: 2, message: t('pages.tunnels.poolNeedsTwo') },
            ]}
          >
            <Select
              mode="multiple"
              tagRender={selectTagRender}
              options={entryTunnels.map((row) => ({ value: row.name, label: row.name }))}
            />
          </Form.Item>
          <Form.Item name="routeInbounds" label={t('pages.tunnels.routeInbounds')}>
            <Select mode="multiple" tagRender={selectTagRender} options={inboundTagOptions} />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
