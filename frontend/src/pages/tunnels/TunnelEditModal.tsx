import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Alert, Form, Input, InputNumber, Select, Switch } from 'antd';

import Icon from '@/components/dl/Icon';
import { FormModal } from '@/components/dl/overlays';
import { Badge, Btn } from '@/components/dl/primitives';

import { selectTagRender } from '@/components/form/SelectTag';
import type {
  ServerTunnel,
  TunnelEditValues,
  TunnelMode,
  TunnelOption,
} from '@/api/queries/useTunnels';
import TunnelOptionFields, { cleanOptions } from './TunnelOptionFields';
import { isForwarder, requiredSettings } from './forwardingEngines';
import { isRawCarrier, needsPeerIp } from './rakhshCarriers';

interface Props {
  tunnel: ServerTunnel;
  options: TunnelOption[];
  transports: { value: string; label: string }[];
  inboundTagOptions: { value: string; label: string }[];
  onSubmit: (id: number, values: TunnelEditValues) => Promise<boolean>;
  onClose: () => void;
}

// Mount it only while editing (keyed by tunnel), so the form always starts from the stored values.
// The entry end owns the shared settings; the exit end only owns where its traffic leaves and its own options.
export default function TunnelEditModal({
  tunnel,
  options,
  transports,
  inboundTagOptions,
  onSubmit,
  onClose,
}: Props) {
  const { t } = useTranslation();
  const [form] = Form.useForm<TunnelEditValues>();
  const [saving, setSaving] = useState(false);

  const portal = tunnel.role === 'portal';
  const backpack = tunnel.engine === 'backpack';
  const rakhsh = tunnel.engine === 'rakhsh';
  const forwarder = isForwarder(tunnel.engine);
  const transport = Form.useWatch('transport', form) ?? tunnel.transport;
  const links = Form.useWatch(['options', 'links'], form) as { carrier?: string }[] | undefined;
  const rakhshNeedsPeerIp = needsPeerIp([transport, ...(links ?? []).map((l) => l.carrier)]);
  const mode: TunnelMode = tunnel.mode === 'direct' ? 'direct' : 'reverse';

  const submit = async () => {
    const values = await form.validateFields().catch(() => null);
    if (!values) return;
    setSaving(true);
    try {
      const body: TunnelEditValues = portal
        ? {
            address: (values.address ?? '').trim(),
            port: values.port,
            routeInbounds: values.routeInbounds ?? [],
            ...(rakhsh || forwarder
              ? {
                  transport: values.transport,
                  forwardPorts: values.forwardPorts ?? [],
                  options: cleanOptions(values.options),
                }
              : {
                  transport: mode === 'direct' ? '' : values.transport,
                  fallbackTransports: values.fallbackTransports ?? [],
                  forwardPorts: values.forwardPorts ?? [],
                  acceptUdp: values.acceptUdp ?? false,
                  options: cleanOptions(values.options),
                }),
          }
        : {
            exitOutbound: (values.exitOutbound ?? '').trim(),
            options: cleanOptions(values.options),
          };
      if (await onSubmit(tunnel.id, body)) onClose();
    } finally {
      setSaving(false);
    }
  };

  return (
    <FormModal
      open
      title={`${t('pages.tunnels.editTunnel')} · ${tunnel.name}`}
      subtitle={
        <Badge acc={mode !== 'direct'}>
          {mode === 'direct' ? t('pages.tunnels.modeDirect') : t('pages.tunnels.modeReverse')}
        </Badge>
      }
      art={
        <span className="ico acc">
          <Icon name="tunnel" />
        </span>
      }
      width={640}
      onClose={onClose}
      footer={
        <>
          <Btn kind="ghost" onClick={onClose}>
            {t('cancel')}
          </Btn>
          <Btn kind="pri" icon="check" disabled={saving} onClick={submit}>
            {t('save')}
          </Btn>
        </>
      }
    >
      {portal && tunnel.nodeId > 0 && (
        <Alert
          type="info"
          showIcon
          title={t('pages.tunnels.editReachesNode')}
          style={{ marginBottom: 12 }}
        />
      )}
      <Form form={form} layout="vertical" initialValues={tunnel}>
        {portal ? (
          <>
            <Form.Item
              name="address"
              label={t('pages.tunnels.address')}
              extra={t('pages.tunnels.addressHint')}
              rules={[{ required: true, whitespace: true }]}
            >
              <Input autoComplete="off" />
            </Form.Item>
            <Form.Item
              name="port"
              label={t('pages.tunnels.port')}
              rules={[{ required: true, type: 'number', min: 1, max: 65535 }]}
            >
              <InputNumber min={1} max={65535} style={{ width: '100%' }} />
            </Form.Item>
            {rakhsh || forwarder ? (
              <>
                <Form.Item name="transport" label={t('pages.tunnels.transport')}>
                  <Select options={transports} />
                </Form.Item>
                {rakhsh && isRawCarrier(transport) && (
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
                  />
                </Form.Item>
              </>
            ) : (
              <>
                {mode === 'reverse' && (
                  <>
                    <Form.Item name="transport" label={t('pages.tunnels.transport')}>
                      <Select options={transports} />
                    </Form.Item>
                    <Form.Item
                      name="fallbackTransports"
                      label={t('pages.tunnels.fallbackTransports')}
                      extra={t('pages.tunnels.fallbackTransportsHint')}
                    >
                      <Select
                        mode="multiple"
                        options={transports}
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
              </>
            )}
            {(mode === 'reverse' || rakhsh || forwarder) && (
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
          </>
        ) : (
          <Form.Item
            name="exitOutbound"
            label={t('pages.tunnels.exitOutbound')}
            extra={t('pages.tunnels.exitOutboundHint')}
          >
            <Input placeholder="direct" autoComplete="off" />
          </Form.Item>
        )}
        {backpack && (
          <TunnelOptionFields
            options={options}
            mode={mode}
            skip={mode === 'direct' ? ['local_ip', 'peer_ip'] : []}
          />
        )}
        {forwarder && portal && (
          <TunnelOptionFields options={options} mode={mode} pinned={requiredSettings(options)} />
        )}
        {rakhsh && (
          <TunnelOptionFields
            options={options}
            mode={mode}
            forRole={portal ? 'portal' : 'bridge'}
            pinned={rakhshNeedsPeerIp ? ['peer_ip'] : []}
          />
        )}
      </Form>
    </FormModal>
  );
}
