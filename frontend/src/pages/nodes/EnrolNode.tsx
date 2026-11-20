import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { FormProvider, useForm, useWatch } from 'react-hook-form';
import {
  Alert,
  Button,
  Form,
  Input,
  InputNumber,
  List,
  Modal,
  Popconfirm,
  Radio,
  Select,
  Typography,
  message,
} from 'antd';
import { ApiOutlined, CopyOutlined, DeleteOutlined } from '@ant-design/icons';

import { withBasePath } from '@/api/http-init';
import {
  useEnrolmentMutations,
  useWaitingEnrolments,
  type Invitation,
} from '@/api/queries/useEnrolments';
import { useTunnelsQuery } from '@/api/queries/useTunnels';
import { FormField, rhfZodValidate } from '@/components/form/rhf';
import { EnrolmentFormSchema, type EnrolmentFormValues } from '@/schemas/enrolment';
import { ClipboardManager } from '@/utils';
import { BACKPACK_TRANSPORTS } from '../tunnels/backpackTransports';
import { RAKHSH_STREAM_CARRIERS } from '../tunnels/rakhshCarriers';

const DEFAULT_TRANSPORT = { backpack: 'wssmux', rakhsh: 'wss' } as const;

function defaultValues(name = ''): EnrolmentFormValues {
  return {
    name,
    panelUrl: window.location.origin + withBasePath('/'),
    address: window.location.hostname,
    port: 8443,
    engine: 'backpack',
    transport: DEFAULT_TRANSPORT.backpack,
    tunnelId: 0,
  };
}

export default function EnrolNode() {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button icon={<ApiOutlined aria-hidden />} onClick={() => setOpen(true)}>
        {t('pages.enrolments.title')}
      </Button>
      <EnrolNodeDialog open={open} onClose={() => setOpen(false)} />
    </>
  );
}

interface EnrolNodeDialogProps {
  open: boolean;
  onClose: () => void;
  initialName?: string;
}

export function EnrolNodeDialog({ open, onClose, initialName }: EnrolNodeDialogProps) {
  return open ? <EnrolNodeModal onClose={onClose} initialName={initialName} /> : null;
}

function EnrolNodeModal({ onClose, initialName }: Omit<EnrolNodeDialogProps, 'open'>) {
  const { t } = useTranslation();
  const [invitation, setInvitation] = useState<Invitation | null>(null);
  const [messageApi, messageContextHolder] = message.useMessage();
  const methods = useForm<EnrolmentFormValues>({ defaultValues: defaultValues(initialName) });
  const engine = useWatch({ control: methods.control, name: 'engine' });
  const tunnelId = useWatch({ control: methods.control, name: 'tunnelId' });
  const { tunnels } = useTunnelsQuery();
  const carriers = tunnels.filter(
    (row) =>
      row.role === 'portal' &&
      !row.managePort &&
      !row.parentId &&
      row.enable &&
      (row.engine === 'backpack' || row.engine === 'rakhsh'),
  );
  const waiting = useWaitingEnrolments(true);
  const { create, creating, cancel } = useEnrolmentMutations();

  useEffect(() => {
    methods.setValue('transport', DEFAULT_TRANSPORT[engine]);
  }, [engine, methods]);

  const submit = methods.handleSubmit(async (values) => {
    const parsed = EnrolmentFormSchema.parse(values);
    const created = await create(
      parsed.tunnelId > 0
        ? { name: parsed.name, panelUrl: parsed.panelUrl, tunnelId: parsed.tunnelId }
        : {
            name: parsed.name,
            panelUrl: parsed.panelUrl,
            address: parsed.address,
            port: parsed.port,
            engine: parsed.engine,
            transport: parsed.transport,
          },
    );
    if (created) setInvitation(created);
  });

  const copyInstallCommand = async () => {
    if (invitation && (await ClipboardManager.copyText(invitation.installCommand)))
      messageApi.success(t('pages.enrolments.copied'));
  };

  const copyCommand = async () => {
    if (invitation && (await ClipboardManager.copyText(invitation.command))) {
      messageApi.success(t('pages.enrolments.copied'));
    }
  };

  return (
    <>
      {messageContextHolder}
      <Modal
        title={t('pages.enrolments.title')}
        open
        onCancel={onClose}
        footer={
          invitation ? (
            <Button type="primary" onClick={onClose}>
              {t('close')}
            </Button>
          ) : (
            <Button type="primary" loading={creating} onClick={submit}>
              {t('pages.enrolments.create')}
            </Button>
          )
        }
      >
        {invitation ? (
          <>
            <Alert
              type="info"
              showIcon
              title={t('pages.enrolments.commandTitle')}
              description={t('pages.enrolments.commandHint')}
              style={{ marginBottom: 12 }}
            />
            <Typography.Paragraph code style={{ direction: 'ltr', wordBreak: 'break-all' }}>
              {invitation.command}
            </Typography.Paragraph>
            <Button icon={<CopyOutlined aria-hidden />} onClick={copyCommand}>
              {t('pages.enrolments.copy')}
            </Button>
            <Typography.Paragraph type="secondary" style={{ marginTop: 16 }}>
              {t('pages.enrolments.installTitle')}
            </Typography.Paragraph>
            <Typography.Paragraph code style={{ direction: 'ltr', wordBreak: 'break-all' }}>
              {invitation.installCommand}
            </Typography.Paragraph>
            <Button icon={<CopyOutlined aria-hidden />} onClick={copyInstallCommand}>
              {t('pages.enrolments.copyInstall')}
            </Button>
          </>
        ) : (
          <FormProvider {...methods}>
            <Typography.Paragraph type="secondary">
              {t('pages.enrolments.intro')}
            </Typography.Paragraph>
            <Form layout="vertical">
              <FormField
                name="name"
                label={t('pages.enrolments.name')}
                rules={{ validate: rhfZodValidate(EnrolmentFormSchema.shape.name) }}
              >
                <Input placeholder="de1" autoComplete="off" />
              </FormField>
              <FormField
                name="panelUrl"
                label={t('pages.enrolments.panelUrl')}
                extra={t('pages.enrolments.panelUrlHint')}
                rules={{ validate: rhfZodValidate(EnrolmentFormSchema.shape.panelUrl) }}
              >
                <Input autoComplete="off" style={{ direction: 'ltr' }} />
              </FormField>
              {carriers.length > 0 && (
                <FormField
                  name="tunnelId"
                  label={t('pages.enrolments.tunnel')}
                  extra={t('pages.enrolments.tunnelHint')}
                >
                  <Select
                    options={[
                      { value: 0, label: t('pages.enrolments.tunnelNew') },
                      ...carriers.map((row) => ({
                        value: row.id,
                        label:
                          `${row.name} · ${row.address}:${row.port}` +
                          ((row.routeInbounds ?? []).length > 0
                            ? ` · ${t('pages.enrolments.tunnelCarriesUsers')}`
                            : ''),
                      })),
                    ]}
                  />
                </FormField>
              )}
              {tunnelId === 0 && (
                <>
                  <FormField name="engine" label={t('pages.tunnels.engine')}>
                    <Radio.Group
                      options={[
                        { value: 'backpack', label: t('pages.tunnels.engineBackpack') },
                        { value: 'rakhsh', label: t('pages.tunnels.engineRakhsh') },
                      ]}
                    />
                  </FormField>
                  <FormField
                    name="address"
                    label={t('pages.tunnels.address')}
                    extra={t('pages.tunnels.addressHint')}
                    rules={{ validate: rhfZodValidate(EnrolmentFormSchema.shape.address) }}
                  >
                    <Input autoComplete="off" style={{ direction: 'ltr' }} />
                  </FormField>
                  <FormField
                    name="port"
                    label={t('pages.tunnels.port')}
                    rules={{ validate: rhfZodValidate(EnrolmentFormSchema.shape.port) }}
                  >
                    <InputNumber min={1} max={65535} style={{ width: '100%' }} />
                  </FormField>
                  <FormField name="transport" label={t('pages.tunnels.transport')}>
                    <Select
                      options={engine === 'rakhsh' ? RAKHSH_STREAM_CARRIERS : BACKPACK_TRANSPORTS}
                    />
                  </FormField>
                </>
              )}
            </Form>
            {waiting.length > 0 && (
              <List
                size="small"
                header={t('pages.enrolments.waiting')}
                dataSource={waiting}
                renderItem={(row) => (
                  <List.Item
                    actions={[
                      <Popconfirm
                        key="cancel"
                        title={t('pages.enrolments.cancelConfirm')}
                        onConfirm={() => cancel(row.id)}
                      >
                        <Button
                          type="text"
                          danger
                          icon={<DeleteOutlined aria-hidden />}
                          aria-label={t('pages.enrolments.cancel')}
                        />
                      </Popconfirm>,
                    ]}
                  >
                    <List.Item.Meta
                      title={row.name}
                      description={`${t('pages.enrolments.expires')}: ${new Date(row.expiresAt * 1000).toLocaleTimeString()}`}
                    />
                  </List.Item>
                )}
              />
            )}
          </FormProvider>
        )}
      </Modal>
    </>
  );
}
