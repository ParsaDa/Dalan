import { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Alert, Form, Input, InputNumber, Radio, Select, Switch, message } from 'antd';
import { useQueryClient } from '@tanstack/react-query';

import { useMediaQuery } from '@/hooks/useMediaQuery';

import Icon from '@/components/dl/Icon';
import { IlloEmpty, IlloOffline } from '@/components/dl/Illustrations';
import { FormModal, Modal as DlModal } from '@/components/dl/overlays';
import { Btn, Chip, Empty as DlEmpty, FlagDot, Sk, TileHead } from '@/components/dl/primitives';
import { fmtInt } from '@/components/dl/numbers';
import { EngineCard, FlowArrow, TunnelCard, TunnelRow } from './TunnelParts';
import PortTest from './PortTest';
import { TunnelBenchmark, type BenchmarkPick } from './TunnelBenchmark';
import { TunnelSpeedtest } from './TunnelSpeedtest';
import TunnelEditModal from './TunnelEditModal';
import TunnelOptionFields, { TunnelOptionField, cleanOptions } from './TunnelOptionFields';
import TunnelPools from './TunnelPools';
import { BACKPACK_TRANSPORTS } from './backpackTransports';
import {
  FORWARDER_ENGINES,
  FORWARDER_LABELS,
  isForwarder,
  modeAfterEngineChange,
  modesOf,
  settingsAsOptions,
  transportAfterChange,
  transportsFor,
} from './forwardingEngines';
import { RAKHSH_CARRIERS, isRawCarrier, needsPeerIp } from './rakhshCarriers';
import { useNodesQuery } from '@/api/queries/useNodesQuery';
import { useInboundOptions } from '@/api/queries/useInboundOptions';
import {
  fetchSetupLink,
  useForwardingEngines,
  useTunnelMutations,
  useTunnelOptions,
  TUNNELS_KEY,
  useTunnelStatuses,
  useTunnelsQuery,
  type NewPortalValues,
  type ServerTunnel,
  type TunnelEditValues,
  type TunnelEngine,
  type TunnelMode,
  type TunnelOptionValues,
} from '@/api/queries/useTunnels';
import { selectTagRender } from '@/components/form/SelectTag';
import { ClipboardManager } from '@/utils';
import { setMessageInstance } from '@/utils/messageBus';

const NAME_PATTERN = /^[a-z0-9][a-z0-9-]{0,30}$/;

const RAKHSH_FORWARD_PLACEHOLDER = '443=127.0.0.1:2096';

interface PortalFormValues {
  name: string;
  address: string;
  port: number;
  engine: TunnelEngine;
  routeInbounds?: string[];
  nodeId?: number;
  transport: string;
  fallbackTransports?: string[];
  forwardPorts?: string[];
  acceptUdp?: boolean;
  mode: TunnelMode;
  options?: TunnelOptionValues;
}

export default function TunnelsPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { isMobile: narrow } = useMediaQuery(700);
  const [messageApi, messageContextHolder] = message.useMessage();
  useEffect(() => {
    setMessageInstance(messageApi);
  }, [messageApi]);

  const { tunnels, fetched, fetchError, refetch } = useTunnelsQuery();
  const { create, importLink, update, setEnable, remove } = useTunnelMutations();
  const [editing, setEditing] = useState<ServerTunnel | null>(null);
  const { data: inboundOptions = [] } = useInboundOptions();
  const { nodes } = useNodesQuery();
  const nodeRefs = useMemo(
    () =>
      nodes.map((n) => ({
        id: n.id,
        name: n.name ?? '',
        country: n.country,
        enable: n.enable,
        transitive: n.transitive,
      })),
    [nodes],
  );

  const [portalOpen, setPortalOpen] = useState(false);
  const [importOpen, setImportOpen] = useState(false);
  const [benchmarkOpen, setBenchmarkOpen] = useState(false);
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
  const usesRakhsh = tunnels.some((tunnel) => tunnel.engine === 'rakhsh');
  const { data: backpackCatalog = [] } = useTunnelOptions('backpack');
  const { data: rakhshCatalog = [] } = useTunnelOptions(
    'rakhsh',
    usesRakhsh || engine === 'rakhsh',
  );
  const { data: forwardingEngines = [] } = useForwardingEngines();
  const forwarder = isForwarder(engine);
  const forwarderDef = forwardingEngines.find((e) => e.id === engine);
  const editingEngine = forwardingEngines.find((e) => e.id === editing?.engine);
  const forwarderOptions = useMemo(() => settingsAsOptions(forwarderDef), [forwarderDef]);
  // The mode and transport must be ones the chosen engine runs, also once its list has arrived.
  useEffect(() => {
    if (!forwarderDef) return;
    const nextMode = modeAfterEngineChange(forwarderDef, mode);
    if (nextMode !== mode) portalForm.setFieldValue('mode', nextMode);
    const nextTransport = transportAfterChange(forwarderDef, nextMode, chosenTransport);
    if (nextTransport !== chosenTransport) portalForm.setFieldValue('transport', nextTransport);
  }, [forwarderDef, mode, chosenTransport, portalForm]);
  const direct = engine === 'backpack' && mode === 'direct';
  // A direct tunnel's entry end dials the exit end, which is where the node (if any) is.
  const dialsExit = mode === 'direct';
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

  const report = (msg: { success?: boolean; msg?: string } | null | undefined) => {
    if (msg?.success) return true;
    messageApi.error(msg?.msg || t('somethingWentWrong'));
    return false;
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
      const request: NewPortalValues = isForwarder(values.engine)
        ? {
            ...common,
            engine: values.engine,
            transport: values.transport,
            forwardPorts: values.forwardPorts ?? [],
            routeInbounds: values.routeInbounds ?? [],
            mode: values.mode,
            options: cleanOptions(values.options),
          }
        : values.engine === 'rakhsh'
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

  const applyBenchmarkMethod = ({ candidate, nodeId, address }: BenchmarkPick) => {
    setBenchmarkOpen(false);
    setPortalOpen(true);
    portalForm.setFieldsValue({
      engine: candidate.engine as TunnelEngine,
      mode: candidate.mode as TunnelMode,
      transport: candidate.transport,
      nodeId,
      address,
    });
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

  const statuses = useTunnelStatuses(tunnels);
  const statusOf = (id: number) => statuses[tunnels.findIndex((tu) => tu.id === id)]?.data;
  const stateOf = (tu: ServerTunnel): 'ok' | 'bad' | 'off' | 'none' =>
    !tu.enable
      ? 'off'
      : tu.managePort
        ? 'none'
        : statusOf(tu.id)?.connected
          ? 'ok'
          : statusOf(tu.id)
            ? 'bad'
            : 'none';
  const [filter, setFilter] = useState<
    'all' | 'ok' | 'bad' | 'off' | 'backpack' | 'rakhsh' | 'forwarding'
  >('all');
  const shown = tunnels.filter((tu) =>
    filter === 'all'
      ? true
      : filter === 'forwarding'
        ? isForwarder(tu.engine)
        : filter === 'backpack' || filter === 'rakhsh'
          ? tu.engine === filter
          : stateOf(tu) === filter,
  );
  const [deleting, setDeleting] = useState<ServerTunnel | null>(null);
  const [speedtesting, setSpeedtesting] = useState<ServerTunnel | null>(null);
  const [removeHere, setRemoveHere] = useState<{ row: ServerTunnel; reason: string } | null>(null);
  const testAll = () => queryClient.invalidateQueries({ queryKey: [...TUNNELS_KEY, 'status'] });
  const flowTunnel =
    tunnels.find((tu) => tu.enable && tu.role === 'portal' && tu.mode === 'reverse' && tu.nodeId) ??
    tunnels.find((tu) => tu.enable && tu.nodeId);
  const flowNode = flowTunnel ? nodes.find((n) => n.id === flowTunnel.nodeId) : undefined;
  const count = (f: typeof filter) =>
    tunnels.filter((tu) =>
      f === 'forwarding'
        ? isForwarder(tu.engine)
        : f === 'backpack' || f === 'rakhsh'
          ? tu.engine === f
          : stateOf(tu) === f,
    ).length;

  return (
    <div className="col stag" style={{ gap: 18 }}>
      {messageContextHolder}
      <div className="pagehead">
        <div>
          <h2 className="h2" style={{ fontSize: 28 }}>
            {t('pages.tunnels.title')}
          </h2>
          <p>{t('tunnels.subtitle')}</p>
        </div>
        <div className="row wrap">
          <Btn icon="refresh" onClick={testAll}>
            {t('tunnels.testAll')}
          </Btn>
          {nodes.length > 0 && (
            <Btn icon="bolt" onClick={() => setBenchmarkOpen(true)}>
              {t('tunnels.benchmark.open')}
            </Btn>
          )}
          <Btn icon="down" onClick={() => setImportOpen(true)}>
            {t('pages.tunnels.importLink')}
          </Btn>
          <Btn kind="pri" icon="plus" onClick={() => setPortalOpen(true)}>
            {t('pages.tunnels.newPortal')}
          </Btn>
        </div>
      </div>
      {!fetched ? (
        <section className="tile">
          <Sk h={120} />
        </section>
      ) : fetchError ? (
        <section className="tile edge st-bad">
          <DlEmpty
            art={<IlloOffline />}
            title={t('somethingWentWrong')}
            actions={
              <Btn kind="pri" icon="refresh" onClick={() => refetch()}>
                {t('refresh')}
              </Btn>
            }
          >
            {fetchError}
          </DlEmpty>
        </section>
      ) : (
        <>
          <div className="grid">
            <EngineCard
              engine="backpack"
              label={t('pages.tunnels.engineBackpack')}
              blurb={t('tunnels.backpackBlurb')}
              count={tunnels.filter((tu) => tu.engine === 'backpack').length}
              nodes={nodeRefs}
              report={report}
            />
            <EngineCard
              engine="rakhsh"
              label={t('pages.tunnels.engineRakhsh')}
              blurb={t('tunnels.rakhshBlurb')}
              count={tunnels.filter((tu) => tu.engine === 'rakhsh').length}
              nodes={nodeRefs}
              report={report}
            />
            {FORWARDER_ENGINES.map((id) => (
              <EngineCard
                key={id}
                engine={id}
                label={FORWARDER_LABELS[id]}
                blurb={t(`tunnels.blurb.${id}`)}
                count={tunnels.filter((tu) => tu.engine === id).length}
                nodes={nodeRefs}
                report={report}
              />
            ))}
            <section
              className="tile s4"
              style={{
                borderStyle: 'dashed',
                display: 'flex',
                flexDirection: 'column',
                justifyContent: 'center',
                alignItems: 'center',
                textAlign: 'center',
                gap: 8,
                minHeight: 170,
              }}
            >
              <span className="ico">
                <Icon name="plus" />
              </span>
              <div className="h3">{t('tunnels.moreEngines')}</div>
              <div className="mut" style={{ fontSize: 12.5, maxWidth: 260 }}>
                {t('tunnels.moreEnginesHint')}
              </div>
            </section>
            <section className="tile s12" aria-label={t('tunnels.flowTitle')}>
              <TileHead
                icon="tunnel"
                title={t('tunnels.flowTitle')}
                right={
                  <span className="mut" style={{ fontSize: 12.5, fontWeight: 650 }}>
                    {t('tunnels.flowHint')}
                  </span>
                }
              />
              <div className="row wrap" style={{ gap: 14, alignItems: 'center' }}>
                <div
                  className="tile"
                  style={{
                    padding: '14px 16px',
                    display: 'flex',
                    gap: 10,
                    alignItems: 'center',
                  }}
                >
                  <span className="ico">
                    <Icon name="users" />
                  </span>
                  <div>
                    <b>{t('tunnels.flowUsers')}</b>
                    <div className="mut" style={{ fontSize: 12 }}>
                      {t('tunnels.flowUsersSub')}
                    </div>
                  </div>
                </div>
                <FlowArrow />
                <div
                  className="tile edge"
                  style={{
                    padding: '14px 16px',
                    display: 'flex',
                    gap: 10,
                    alignItems: 'center',
                  }}
                >
                  <FlagDot size={28} />
                  <div>
                    <b>{t('pages.index.thisServer')}</b>
                    <div className="mut" style={{ fontSize: 12 }}>
                      {t('tunnels.flowEntry')}
                    </div>
                  </div>
                </div>
                <FlowArrow reverse={flowTunnel?.mode !== 'direct'} />
                <div
                  className="tile"
                  style={{
                    padding: '14px 16px',
                    display: 'flex',
                    gap: 10,
                    alignItems: 'center',
                  }}
                >
                  <FlagDot country={flowNode?.country} size={28} />
                  <div>
                    <b>{flowNode?.name ?? t('tunnels.flowExitNode')}</b>
                    <div className="mut" style={{ fontSize: 12 }}>
                      {t(flowTunnel?.mode === 'direct' ? 'tunnels.flowDials' : 'tunnels.flowOpens')}
                    </div>
                  </div>
                </div>
                <span className="mut" style={{ fontWeight: 650, fontSize: 13 }}>
                  {t('tunnels.flowInternet')}
                </span>
              </div>
            </section>
          </div>
          {tunnels.some((tunnel) => tunnel.role === 'portal') && (
            <div className="row mut" style={{ gap: 8, fontSize: 13, fontWeight: 650 }}>
              <Icon name="info" size={16} />
              <span className="f1">{t('pages.tunnels.linkHint')}</span>
            </div>
          )}
          <div className="chips" role="group" aria-label={t('tunnels.filters')}>
            {(['all', 'ok', 'bad', 'off', 'backpack', 'rakhsh', 'forwarding'] as const).map((f) => (
              <Chip
                key={f}
                on={filter === f}
                count={fmtInt(f === 'all' ? tunnels.length : count(f))}
                onClick={() => setFilter(f)}
              >
                {t(`tunnels.filter.${f}`)}
              </Chip>
            ))}
          </div>
          {tunnels.length === 0 ? (
            <section className="tile">
              <DlEmpty
                art={<IlloEmpty size={130} />}
                title={t('pages.tunnels.empty')}
                actions={
                  <Btn kind="pri" icon="plus" onClick={() => setPortalOpen(true)}>
                    {t('pages.tunnels.newPortal')}
                  </Btn>
                }
              />
            </section>
          ) : (
            <section className="tile dense" style={{ padding: 20 }}>
              {narrow ? (
                <div className="list">
                  {shown.map((row) => (
                    <TunnelCard
                      key={row.id}
                      row={row}
                      nodes={nodeRefs}
                      status={statusOf(row.id)}
                      state={stateOf(row)}
                      onToggle={async (on) => report(await setEnable({ id: row.id, enable: on }))}
                      onEdit={() => setEditing(row)}
                      onCopy={() => copyLink(row.id)}
                      onDelete={() => setDeleting(row)}
                      onSpeedtest={() => setSpeedtesting(row)}
                    />
                  ))}
                </div>
              ) : (
                <div className="tblbox">
                  <table className="tbl" style={{ minWidth: 1040 }}>
                    <thead>
                      <tr>
                        <th>{t('tunnels.colTunnel')}</th>
                        <th>{t('tunnels.colEngine')}</th>
                        <th>{t('tunnels.colMode')}</th>
                        <th>{t('tunnels.colLatency')}</th>
                        <th>{t('tunnels.colLastHour')}</th>
                        <th>{t('tunnels.colState')}</th>
                        <th>{t('pages.tunnels.traffic')}</th>
                        <th>{t('tunnels.colRoutes')}</th>
                        <th>
                          <span
                            style={{
                              position: 'absolute',
                              width: 1,
                              height: 1,
                              overflow: 'hidden',
                            }}
                          >
                            {t('pages.nodes.actions')}
                          </span>
                        </th>
                      </tr>
                    </thead>
                    <tbody>
                      {shown.map((row) => (
                        <TunnelRow
                          key={row.id}
                          row={row}
                          nodes={nodeRefs}
                          status={statusOf(row.id)}
                          state={stateOf(row)}
                          onToggle={async (on) =>
                            report(await setEnable({ id: row.id, enable: on }))
                          }
                          onEdit={() => setEditing(row)}
                          onCopy={() => copyLink(row.id)}
                          onDelete={() => setDeleting(row)}
                          onSpeedtest={() => setSpeedtesting(row)}
                        />
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </section>
          )}
          <TunnelPools tunnels={tunnels} />
        </>
      )}

      <TunnelSpeedtest tunnel={speedtesting} onClose={() => setSpeedtesting(null)} />
      <TunnelBenchmark
        open={benchmarkOpen}
        nodes={nodes.map((n) => ({ id: n.id, name: n.name ?? '', enable: n.enable ?? false }))}
        catalog={forwardingEngines}
        onUse={applyBenchmarkMethod}
        onClose={() => setBenchmarkOpen(false)}
      />
      <DlModal
        open={deleting !== null}
        onClose={() => setDeleting(null)}
        label={t('pages.tunnels.deleteConfirm')}
        width={440}
      >
        <div className="row" style={{ gap: 12 }}>
          <span className="ico bad">
            <Icon name="trash" />
          </span>
          <h2 className="h3 f1">{t('pages.tunnels.deleteConfirm')}</h2>
        </div>
        <div className="mfoot">
          <Btn kind="ghost" onClick={() => setDeleting(null)}>
            {t('cancel')}
          </Btn>
          <Btn
            kind="dng"
            icon="trash"
            onClick={async () => {
              const row = deleting;
              setDeleting(null);
              if (!row) return;
              const msg = await remove({ id: row.id });
              if (msg?.success || !row.nodeId) report(msg);
              else
                setRemoveHere({
                  row,
                  reason: msg?.msg || t('somethingWentWrong'),
                });
            }}
          >
            {t('delete')}
          </Btn>
        </div>
      </DlModal>
      <DlModal
        open={removeHere !== null}
        onClose={() => setRemoveHere(null)}
        label={removeHere?.reason ?? ''}
        width={480}
      >
        <h2 className="h3">{removeHere?.reason}</h2>
        <p className="mut" style={{ margin: 0 }}>
          {t('pages.tunnels.removeHereHint')}
        </p>
        <div className="mfoot">
          <Btn kind="ghost" onClick={() => setRemoveHere(null)}>
            {t('cancel')}
          </Btn>
          <Btn
            kind="dng"
            icon="trash"
            onClick={async () => {
              const row = removeHere?.row;
              setRemoveHere(null);
              if (row) report(await remove({ id: row.id, here: true }));
            }}
          >
            {t('pages.tunnels.removeHere')}
          </Btn>
        </div>
      </DlModal>
      <FormModal
        open={portalOpen}
        title={t('pages.tunnels.newPortal')}
        art={
          <span className="ico acc">
            <Icon name="tunnel" />
          </span>
        }
        onClose={() => setPortalOpen(false)}
        width={640}
        footer={
          <>
            <Btn kind="ghost" onClick={() => setPortalOpen(false)}>
              {t('cancel')}
            </Btn>
            <Btn kind="pri" icon="check" disabled={saving} onClick={submitPortal}>
              {t('tunnels.create')}
            </Btn>
          </>
        }
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
                ...FORWARDER_ENGINES.map((id) => ({ value: id, label: FORWARDER_LABELS[id] })),
              ]}
            />
          </Form.Item>
          {(engine === 'backpack' || engine === 'rakhsh' || forwarderDef) && (
            <Form.Item
              name="mode"
              label={t('pages.tunnels.mode')}
              extra={
                mode === 'direct'
                  ? engine === 'rakhsh'
                    ? t('pages.tunnels.modeDirectHintRakhsh')
                    : forwarder
                      ? t('pages.tunnels.modeDirectHintForwarder')
                      : t('pages.tunnels.modeDirectHint')
                  : t('pages.tunnels.modeReverseHint')
              }
            >
              <Radio.Group
                options={[
                  { value: 'reverse', label: t('pages.tunnels.modeReverse') },
                  { value: 'direct', label: t('pages.tunnels.modeDirect') },
                ].filter(
                  (o) => !forwarderDef || modesOf(forwarderDef).includes(o.value as TunnelMode),
                )}
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
          {engine === 'rakhsh' || forwarder ? (
            <>
              <Form.Item name="transport" label={t('pages.tunnels.transport')}>
                <Select
                  options={forwarderDef ? transportsFor(forwarderDef, mode) : RAKHSH_CARRIERS}
                />
              </Form.Item>
              {engine === 'rakhsh' && isRawCarrier(chosenTransport) && (
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
              {forwarder ? (
                <TunnelOptionFields options={forwarderOptions} mode={mode} />
              ) : (
                <TunnelOptionFields
                  options={rakhshCatalog}
                  mode={mode}
                  forRole="portal"
                  pinned={rakhshNeedsPeerIp ? ['peer_ip'] : []}
                />
              )}
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
      </FormModal>

      {editing && (
        <TunnelEditModal
          key={editing.id}
          tunnel={editing}
          options={
            editingEngine
              ? settingsAsOptions(editingEngine)
              : editing.engine === 'rakhsh'
                ? rakhshCatalog
                : backpackCatalog
          }
          transports={
            editingEngine
              ? transportsFor(editingEngine, editing.mode === 'direct' ? 'direct' : 'reverse')
              : editing.engine === 'rakhsh'
                ? RAKHSH_CARRIERS
                : BACKPACK_TRANSPORTS
          }
          inboundTagOptions={inboundTagOptions}
          onSubmit={saveEdit}
          onClose={() => setEditing(null)}
        />
      )}

      <FormModal
        open={importOpen}
        title={t('pages.tunnels.importLink')}
        art={
          <span className="ico acc">
            <Icon name="link" />
          </span>
        }
        onClose={() => setImportOpen(false)}
        footer={
          <>
            <Btn kind="ghost" onClick={() => setImportOpen(false)}>
              {t('cancel')}
            </Btn>
            <Btn kind="pri" icon="check" disabled={saving} onClick={submitImport}>
              {t('tunnels.import')}
            </Btn>
          </>
        }
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
      </FormModal>
    </div>
  );
}
