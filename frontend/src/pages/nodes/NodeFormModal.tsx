import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Col, Form, Input, InputNumber, Row, Select, Switch, message } from 'antd';
import { FormProvider, useForm, useWatch } from 'react-hook-form';
import { selectTagRender } from '@/components/form';
import type { NodeRecord } from '@/api/queries/useNodesQuery';
import type { RemoteInboundOption } from '@/api/queries/useNodeMutations';
import type { Msg } from '@/utils';
import { NodeFormSchema, type NodeFormValues, type ProbeResult } from '@/schemas/node';
import { FormField, rhfZodValidate } from '@/components/form/rhf';
import { useOutboundTagGroups } from '@/api/queries/useOutboundTags';
import { AccentPicker, CountrySelect } from './NodeIdentityFields';
import Icon from '@/components/dl/Icon';
import { Modal as DlModal } from '@/components/dl/overlays';
import { Btn, FlagDot, Pill } from '@/components/dl/primitives';

type Mode = 'add' | 'edit';

interface NodeFormModalProps {
  open: boolean;
  mode: Mode;
  node: NodeRecord | null;
  testConnection: (payload: Partial<NodeRecord>) => Promise<Msg<ProbeResult>>;
  fetchFingerprint: (payload: Partial<NodeRecord>) => Promise<Msg<string>>;
  fetchInbounds: (payload: Partial<NodeRecord>) => Promise<Msg<RemoteInboundOption[]>>;
  save: (payload: Partial<NodeRecord>) => Promise<Msg<unknown>>;
  onOpenChange: (open: boolean) => void;
  onRemove?: () => void;
}

function defaultValues(): NodeFormValues {
  return {
    id: 0,
    name: '',
    remark: '',
    scheme: 'https',
    address: '',
    publicAddress: '',
    port: 2053,
    basePath: '/',
    apiToken: '',
    hasStoredToken: false,
    enable: true,
    allowPrivateAddress: false,
    tlsVerifyMode: 'verify',
    pinnedCertSha256: '',
    inboundSyncMode: 'all',
    inboundTags: [],
    outboundTag: '',
    country: '',
    accent: '',
  };
}

export default function NodeFormModal({
  open,
  mode,
  node,
  testConnection,
  fetchFingerprint,
  fetchInbounds,
  save,
  onOpenChange,
  onRemove,
}: NodeFormModalProps) {
  const { t } = useTranslation();
  const methods = useForm<NodeFormValues>({ defaultValues: defaultValues() });
  const [messageApi, messageContextHolder] = message.useMessage();

  const [submitting, setSubmitting] = useState(false);
  const [testing, setTesting] = useState(false);
  const [fetchingPin, setFetchingPin] = useState(false);
  const [fetchingInbounds, setFetchingInbounds] = useState(false);
  const [inboundOptions, setInboundOptions] = useState<RemoteInboundOption[]>([]);
  const [testResult, setTestResult] = useState<ProbeResult | null>(null);
  const scheme = useWatch({ control: methods.control, name: 'scheme' }) ?? 'https';
  const tlsVerifyMode = useWatch({ control: methods.control, name: 'tlsVerifyMode' }) ?? 'verify';
  const country = useWatch({ control: methods.control, name: 'country' }) ?? '';
  const inboundSyncMode = useWatch({ control: methods.control, name: 'inboundSyncMode' }) ?? 'all';
  const { data: outboundGroups } = useOutboundTagGroups({ excludeBlackhole: true });

  // Outbounds and balancers share one picker (like the panel-outbound selector);
  // when balancers exist they get a labeled group so it's clear the selection
  // routes through a balancer. Empty falls back to the placeholder ("Direct
  // connection") rather than a synthetic option, so it can't read as a second
  // "direct" next to a real freedom outbound.
  const outboundOptions = useMemo<
    (
      | { label: string; value: string }
      | { label: string; options: { label: string; value: string }[] }
    )[]
  >(() => {
    const outOpts = (outboundGroups?.outbounds ?? []).map((tag) => ({ label: tag, value: tag }));
    if (!outboundGroups?.balancers.length) return outOpts;
    return [
      { label: t('pages.xray.Outbounds'), options: outOpts },
      {
        label: t('pages.xray.Balancers'),
        options: outboundGroups.balancers.map((tag) => ({ label: tag, value: tag })),
      },
    ];
  }, [outboundGroups, t]);

  // Reset during render, not in an effect, so the first frame is already clean.
  const [synced, setSynced] = useState<{ mode: string; node: NodeRecord | null } | null>(null);
  if (!open) {
    if (synced) setSynced(null);
  } else if (!synced || synced.mode !== mode || synced.node !== (node ?? null)) {
    setSynced({ mode, node: node ?? null });
    const base = defaultValues();
    const next: NodeFormValues =
      mode === 'edit' && node
        ? {
            ...base,
            ...(node as unknown as Partial<NodeFormValues>),
            id: node.id,
            scheme: (node.scheme as 'http' | 'https') || base.scheme,
            inboundSyncMode: (node.inboundSyncMode as 'all' | 'selected') || base.inboundSyncMode,
            inboundTags: node.inboundTags ?? [],
            apiToken: '',
            hasStoredToken: node.hasApiToken ?? false,
          }
        : base;
    if (next.scheme === 'http') next.tlsVerifyMode = 'skip';
    methods.reset(next);
    setInboundOptions((next.inboundTags || []).map((tag) => ({ tag })));
    setTestResult(null);
  }

  const title = useMemo(
    () => (mode === 'edit' ? t('pages.nodes.editNode') : t('pages.nodes.addNode')),
    [mode, t],
  );

  const editingWithToken = mode === 'edit' && Boolean(node?.hasApiToken);

  function buildPayload(values: NodeFormValues): Partial<NodeRecord> {
    const token = values.apiToken.trim();
    const payload: Partial<NodeRecord> = {
      id: values.id || 0,
      name: values.name.trim(),
      remark: values.remark?.trim() || '',
      scheme: values.scheme,
      address: values.address.trim(),
      publicAddress: values.publicAddress.trim(),
      port: values.port,
      basePath: values.basePath.trim() || '/',
      enable: values.enable,
      allowPrivateAddress: values.allowPrivateAddress,
      tlsVerifyMode: values.tlsVerifyMode,
      pinnedCertSha256: values.tlsVerifyMode === 'pin' ? values.pinnedCertSha256.trim() : '',
      inboundSyncMode: values.inboundSyncMode,
      inboundTags: values.inboundSyncMode === 'selected' ? values.inboundTags : [],
      outboundTag: values.outboundTag || '',
      country: values.country,
      accent: values.accent,
    };
    if (token) payload.apiToken = token;
    return payload;
  }

  async function onTest() {
    if (!(await methods.trigger(['name', 'address', 'port']))) return;
    setTesting(true);
    setTestResult(null);
    try {
      const payload = buildPayload(methods.getValues());
      const msg = await testConnection(payload);
      if (msg?.success && msg.obj) {
        setTestResult(msg.obj);
      } else {
        setTestResult({ status: 'offline', error: msg?.msg || 'unknown error' });
      }
    } finally {
      setTesting(false);
    }
  }

  async function onFetchPin() {
    if (!(await methods.trigger(['name', 'address', 'port']))) return;
    setFetchingPin(true);
    try {
      const payload = buildPayload(methods.getValues());
      const msg = await fetchFingerprint(payload);
      if (msg?.success && msg.obj) {
        methods.setValue('pinnedCertSha256', msg.obj);
        messageApi.success(t('pages.nodes.pinFetched'));
      } else {
        messageApi.error(msg?.msg || t('pages.nodes.pinFetchFailed'));
      }
    } finally {
      setFetchingPin(false);
    }
  }

  async function onFetchInbounds() {
    if (!(await methods.trigger(['name', 'address', 'port', 'apiToken']))) return;
    setFetchingInbounds(true);
    try {
      const msg = await fetchInbounds(buildPayload(methods.getValues()));
      if (msg?.success && Array.isArray(msg.obj)) {
        setInboundOptions(msg.obj);
        messageApi.success(t('pages.nodes.inboundsLoaded', { count: msg.obj.length }));
      } else {
        messageApi.error(msg?.msg || t('pages.nodes.inboundsLoadFailed'));
      }
    } finally {
      setFetchingInbounds(false);
    }
  }

  async function onFinish(values: NodeFormValues) {
    const result = NodeFormSchema.safeParse(values);
    if (!result.success) {
      messageApi.error(t(result.error.issues[0]?.message ?? 'pages.nodes.toasts.fillRequired'));
      return;
    }
    setSubmitting(true);
    try {
      const payload = buildPayload(result.data);
      const test = await testConnection(payload);
      const probe = test?.success ? test.obj : null;
      if (!probe || probe.status !== 'online') {
        setTestResult(
          probe ?? { status: 'offline', error: test?.msg || t('pages.nodes.connectionFailed') },
        );
        return;
      }
      setTestResult(probe);
      const msg = await save(payload);
      if (msg?.success) {
        onOpenChange(false);
      }
    } finally {
      setSubmitting(false);
    }
  }

  function close() {
    if (!submitting) onOpenChange(false);
  }

  const note = (tone: 'warn' | 'info', text: string) => (
    <div
      className="row"
      style={{
        gap: 8,
        fontSize: 13,
        fontWeight: 650,
        color: tone === 'warn' ? 'var(--warn-t)' : 'var(--mut)',
        marginBottom: 16,
      }}
    >
      <Icon name={tone === 'warn' ? 'alert' : 'info'} size={16} />
      <span className="f1">{text}</span>
    </div>
  );

  return (
    <>
      {messageContextHolder}
      <DlModal open={open} onClose={close} label={title} width={640}>
        <div className="row sp">
          <div className="row">
            <FlagDot country={country || undefined} size={40} />
            <div>
              <h2 className="h2">{title}</h2>
              {mode === 'edit' && node && <div className="mut">{node.name}</div>}
            </div>
          </div>
          <button type="button" className="ib sm" aria-label={t('close')} onClick={close}>
            <Icon name="x" />
          </button>
        </div>
        <FormProvider {...methods}>
          <Form
            layout="vertical"
            className="dl-form"
            style={{
              maxHeight: 'calc(100vh - 260px)',
              overflowY: 'auto',
              overflowX: 'hidden',
              paddingInlineEnd: 4,
            }}
          >
            <Row gutter={14}>
              <Col xs={24} md={12}>
                <FormField
                  label={t('pages.nodes.name')}
                  name="name"
                  rules={{ validate: rhfZodValidate(NodeFormSchema.shape.name) }}
                >
                  <Input placeholder={t('pages.nodes.namePlaceholder')} />
                </FormField>
              </Col>
              <Col xs={24} md={12}>
                <FormField
                  label={t('pages.nodes.country')}
                  name="country"
                  extra={t('nodes.countryHint')}
                >
                  <CountrySelect />
                </FormField>
              </Col>
            </Row>
            <FormField label={t('nodes.colour')} name="accent" extra={t('nodes.colourHint')}>
              <AccentPicker />
            </FormField>
            <Row gutter={14}>
              <Col xs={24} md={6}>
                <FormField
                  label={t('pages.nodes.scheme')}
                  name="scheme"
                  onAfterChange={(value) => {
                    if (value === 'http') methods.setValue('tlsVerifyMode', 'skip');
                  }}
                >
                  <Select
                    options={[
                      { value: 'https', label: 'https' },
                      { value: 'http', label: 'http' },
                    ]}
                  />
                </FormField>
              </Col>
              <Col xs={24} md={12}>
                <FormField
                  label={t('pages.nodes.address')}
                  name="address"
                  rules={{ validate: rhfZodValidate(NodeFormSchema.shape.address) }}
                >
                  <Input className="mono" placeholder={t('pages.nodes.addressPlaceholder')} />
                </FormField>
              </Col>
              <Col xs={24} md={6}>
                <FormField
                  label={t('nodes.panelPort')}
                  name="port"
                  rules={{ validate: rhfZodValidate(NodeFormSchema.shape.port) }}
                >
                  <InputNumber min={1} max={65535} style={{ width: '100%' }} />
                </FormField>
              </Col>
            </Row>
            <FormField
              label={t('pages.nodes.apiToken')}
              name="apiToken"
              rules={{ validate: rhfZodValidate(NodeFormSchema.shape.apiToken) }}
              tooltip={t('pages.nodes.apiTokenHint')}
              extra={editingWithToken ? t('pages.nodes.apiTokenKeepHint') : undefined}
            >
              <Input.Password
                placeholder={
                  editingWithToken
                    ? t('pages.nodes.apiTokenKeepHint')
                    : t('pages.nodes.apiTokenPlaceholder')
                }
              />
            </FormField>
            <div className="tile" style={{ padding: 14, marginBottom: 16 }}>
              <div className="row" style={{ marginBottom: 10 }}>
                <span className="ico acc">
                  <Icon name="tunnel" />
                </span>
                <div className="f1">
                  <b>{t('nodes.manageThrough')}</b>
                  <div className="mut" style={{ fontSize: 12.5 }}>
                    {t('pages.nodes.outboundTagHint')}
                  </div>
                </div>
              </div>
              <FormField
                label={t('pages.nodes.outboundTag')}
                name="outboundTag"
                transform={{ input: (v) => (v as string) || undefined }}
                style={{ marginBottom: 0 }}
              >
                <Select
                  allowClear
                  showSearch
                  placeholder={t('pages.nodes.outboundTagPlaceholder')}
                  options={outboundOptions}
                />
              </FormField>
            </div>
            <details className="more-opts">
              <summary>
                <Icon name="chevd" />
                {t('nodes.moreOptions')}
              </summary>
              <div style={{ paddingTop: 12 }}>
                <FormField label={t('pages.nodes.remark')} name="remark">
                  <Input />
                </FormField>
                <FormField
                  label={t('pages.nodes.publicAddress')}
                  name="publicAddress"
                  tooltip={t('pages.nodes.publicAddressHint')}
                >
                  <Input placeholder={t('pages.nodes.addressPlaceholder')} />
                </FormField>
                <Row gutter={14}>
                  <Col xs={24} md={12}>
                    <FormField label={t('pages.nodes.basePath')} name="basePath">
                      <Input placeholder="/" />
                    </FormField>
                  </Col>
                  <Col xs={12} md={6}>
                    <FormField label={t('pages.nodes.enable')} name="enable" valueProp="checked">
                      <Switch />
                    </FormField>
                  </Col>
                  <Col xs={12} md={6}>
                    <FormField
                      label={t('pages.nodes.allowPrivateAddress')}
                      name="allowPrivateAddress"
                      valueProp="checked"
                      tooltip={t('pages.nodes.allowPrivateAddressHint')}
                    >
                      <Switch />
                    </FormField>
                  </Col>
                </Row>
                <FormField
                  label={t('pages.nodes.tlsVerifyMode')}
                  name="tlsVerifyMode"
                  tooltip={t('pages.nodes.tlsVerifyModeHint')}
                >
                  <Select
                    disabled={scheme === 'http'}
                    options={[
                      { value: 'verify', label: t('pages.nodes.tlsVerify') },
                      { value: 'pin', label: t('pages.nodes.tlsPin') },
                      { value: 'skip', label: t('pages.nodes.tlsSkip') },
                      { value: 'mtls', label: t('pages.nodes.tlsMtls') },
                    ]}
                  />
                </FormField>
                {tlsVerifyMode === 'skip' && note('warn', t('pages.nodes.tlsSkipWarning'))}
                {tlsVerifyMode === 'mtls' && note('info', t('pages.nodes.mtlsFormHint'))}
                {tlsVerifyMode === 'pin' && (
                  <FormField
                    label={t('pages.nodes.pinnedCert')}
                    name="pinnedCertSha256"
                    tooltip={t('pages.nodes.pinnedCertHint')}
                  >
                    <Input.Search
                      placeholder={t('pages.nodes.pinnedCertPlaceholder')}
                      enterButton={t('pages.nodes.fetchPin')}
                      loading={fetchingPin}
                      onSearch={onFetchPin}
                    />
                  </FormField>
                )}
                <FormField
                  label={t('pages.nodes.inboundSyncMode')}
                  name="inboundSyncMode"
                  tooltip={t('pages.nodes.inboundSyncModeHint')}
                >
                  <Select
                    options={[
                      { value: 'all', label: t('pages.nodes.allInbounds') },
                      { value: 'selected', label: t('pages.nodes.selectedInbounds') },
                    ]}
                  />
                </FormField>
                {inboundSyncMode === 'selected' && (
                  <FormField
                    label={t('pages.nodes.inboundTags')}
                    name="inboundTags"
                    tooltip={t('pages.nodes.inboundTagsHint')}
                  >
                    <Select
                      mode="multiple"
                      tagRender={selectTagRender}
                      allowClear
                      loading={fetchingInbounds}
                      placeholder={t('pages.nodes.inboundTagsPlaceholder')}
                      popupRender={(menu) => (
                        <>
                          <Button
                            type="text"
                            block
                            loading={fetchingInbounds}
                            onClick={onFetchInbounds}
                          >
                            {t('pages.nodes.loadInbounds')}
                          </Button>
                          {menu}
                        </>
                      )}
                      options={inboundOptions.map((inbound) => ({
                        value: inbound.tag,
                        label: `${inbound.remark || inbound.tag}${inbound.protocol ? ` (${inbound.protocol}:${inbound.port || 0})` : ''}`,
                      }))}
                    />
                  </FormField>
                )}
              </div>
            </details>
          </Form>
        </FormProvider>
        <div className="row wrap" style={{ gap: 10 }}>
          <Btn size="sm" icon="refresh" disabled={testing} onClick={onTest}>
            {t('pages.nodes.testConnection')}
          </Btn>
          {testResult &&
            (testResult.status === 'online' ? (
              <Pill status="ok">
                {t('pages.nodes.connectionOk', { ms: testResult.latencyMs })}
                {testResult.xrayVersion ? ` · Xray ${testResult.xrayVersion}` : ''}
              </Pill>
            ) : (
              <Pill status="bad">{testResult.error || t('pages.nodes.connectionFailed')}</Pill>
            ))}
        </div>
        <div className="mfoot">
          {onRemove && (
            <Btn
              kind="ghost"
              className="dng"
              icon="trash"
              style={{ marginInlineEnd: 'auto' }}
              onClick={onRemove}
            >
              {t('nodes.remove')}
            </Btn>
          )}
          <Btn kind="ghost" onClick={close}>
            {t('cancel')}
          </Btn>
          <Btn
            kind="pri"
            icon="check"
            disabled={submitting}
            onClick={methods.handleSubmit(onFinish)}
          >
            {submitting ? t('nodes.saving') : t('save')}
          </Btn>
        </div>
      </DlModal>
    </>
  );
}
