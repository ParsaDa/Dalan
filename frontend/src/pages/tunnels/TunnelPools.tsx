import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Form, Input, Select, message } from 'antd';
import Icon from '@/components/dl/Icon';
import { FormModal, Modal as DlModal } from '@/components/dl/overlays';
import { Btn, IconBtn, Sk, TileHead } from '@/components/dl/primitives';

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

  const [deleting, setDeleting] = useState<TunnelPool | null>(null);

  return (
    <>
      {messageContextHolder}
      <section className="tile">
        <TileHead
          icon="layers"
          title={t('pages.tunnels.pools')}
          right={
            <Btn
              size="sm"
              icon="plus"
              disabled={entryTunnels.length < 2}
              onClick={() => {
                form.resetFields();
                setOpen(true);
              }}
            >
              {t('pages.tunnels.newPool')}
            </Btn>
          }
        />
        <p className="mut" style={{ margin: '-6px 0 10px', fontSize: 12.5 }}>
          {t('pages.tunnels.poolsHint')}
        </p>
        {loading ? (
          <Sk h={56} />
        ) : pools.length === 0 ? (
          <div className="mut" style={{ fontSize: 13, padding: '10px 0' }}>
            {t('pages.tunnels.poolsEmpty')}
          </div>
        ) : (
          <div className="list">
            {pools.map((pool) => (
              <div className="li" key={pool.id}>
                <span className="ico acc">
                  <Icon name="layers" />
                </span>
                <div className="f1" style={{ minWidth: 0 }}>
                  <b>{pool.name}</b>
                  <div className="row wrap" style={{ gap: 4, marginTop: 6 }}>
                    {pool.members.map((member) =>
                      member === pool.active ? (
                        <span key={member} className="badge acc">
                          <bdi>{member}</bdi>
                          {` · ${t('pages.tunnels.poolInUse')}`}
                        </span>
                      ) : (
                        <span key={member} className="badge">
                          {member}
                        </span>
                      ),
                    )}
                    {pool.routeInbounds.map((tag) => (
                      <span key={tag} className="badge mono">
                        {tag}
                      </span>
                    ))}
                  </div>
                </div>
                <IconBtn
                  small
                  icon="trash"
                  label={`${t('delete')} ${pool.name}`}
                  onClick={() => setDeleting(pool)}
                />
              </div>
            ))}
          </div>
        )}
      </section>
      <DlModal
        open={deleting !== null}
        onClose={() => setDeleting(null)}
        label={t('pages.tunnels.poolDeleteConfirm')}
        width={440}
      >
        <div className="row" style={{ gap: 12 }}>
          <span className="ico bad">
            <Icon name="trash" />
          </span>
          <h2 className="h3 f1">{t('pages.tunnels.poolDeleteConfirm')}</h2>
        </div>
        <div className="mfoot">
          <Btn kind="ghost" onClick={() => setDeleting(null)}>
            {t('cancel')}
          </Btn>
          <Btn
            kind="dng"
            icon="trash"
            onClick={async () => {
              const pool = deleting;
              setDeleting(null);
              if (pool) report(await remove(pool.id));
            }}
          >
            {t('delete')}
          </Btn>
        </div>
      </DlModal>
      <FormModal
        open={open}
        title={t('pages.tunnels.newPool')}
        art={
          <span className="ico acc">
            <Icon name="layers" />
          </span>
        }
        onClose={() => setOpen(false)}
        footer={
          <>
            <Btn kind="ghost" onClick={() => setOpen(false)}>
              {t('cancel')}
            </Btn>
            <Btn kind="pri" icon="check" disabled={saving} onClick={submit}>
              {t('save')}
            </Btn>
          </>
        }
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
      </FormModal>
    </>
  );
}
