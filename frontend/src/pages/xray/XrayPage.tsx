import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useLocation, useNavigate } from 'react-router';
import { message } from 'antd';

import { useMediaQuery } from '@/hooks/useMediaQuery';
import { useXraySetting } from '@/hooks/useXraySetting';
import type { XraySettingsValue } from '@/hooks/useXraySetting';
import { JsonEditor } from '@/components/form';
import { setMessageInstance } from '@/utils/messageBus';
import { injectedOutboundTags } from '@/lib/xray/injected-outbound-tags';
import { useTorExits } from '@/api/queries/useTor';

import { BasicsTab } from './basics';
import { propagateOutboundTagRename } from './basics/helpers';
import { RoutingTab } from './routing';
import { OutboundsTab } from './outbounds';
import { BalancersTab } from './balancers';
import {
  cleanupOrphanedBalancerLoopbacks,
  ensureMissingBalancerLoopbacks,
  detectBalancerCycles,
} from './balancers/balancer-loopback';
import { DnsTab } from './dns';
import { WarpModal, NordModal, PiaModal, TorModal } from './overrides';
import './XrayPage.css';
import { FileManager } from '@/utils';
import { IlloOffline } from '@/components/dl/Illustrations';
import Icon from '@/components/dl/Icon';
import { Modal as DlModal } from '@/components/dl/overlays';
import { Btn, Empty as DlEmpty, Seg, Sk, TileHead } from '@/components/dl/primitives';

const SECTION_SLUGS = ['basic', 'routing', 'outbound', 'balancer', 'dns', 'advanced'];

type AdvKey = 'xraySetting' | 'inboundSettings' | 'outboundSettings' | 'routingRuleSettings';

export default function XrayPage() {
  const { t } = useTranslation();
  const { isMobile } = useMediaQuery();
  const [messageApi, messageContextHolder] = message.useMessage();
  useEffect(() => {
    setMessageInstance(messageApi);
  }, [messageApi]);
  const xs = useXraySetting();
  const {
    fetched,
    spinning,
    saveDisabled,
    fetchError,
    xraySetting,
    setXraySetting,
    templateSettings,
    setTemplateSettings,
    outboundTestUrl,
    setOutboundTestUrl,
    inboundTags,
    clientReverseTags,
    subscriptionOutbounds,
    subscriptionOutboundTags,
    tunnelOutbounds,
    torOutbounds,
    outboundsTraffic,
    outboundTestStates,
    subscriptionTestStates,
    testingAll,
    fetchAll,
    resetOutboundsTraffic,
    testOutbound,
    testSubscriptionOutbound,
    testAllOutbounds,
    saveAll,
    resetToDefault,
  } = xs;

  const injectedTags = useMemo(
    () => injectedOutboundTags({ subscriptionOutboundTags, tunnelOutbounds, torOutbounds }),
    [subscriptionOutboundTags, tunnelOutbounds, torOutbounds],
  );
  const { exits: torExits } = useTorExits();
  const torStatus = useMemo(
    () => Object.fromEntries(torExits.map((e) => [e.tag, e.status])),
    [torExits],
  );
  const [warpOpen, setWarpOpen] = useState(false);
  const [nordOpen, setNordOpen] = useState(false);
  const [piaOpen, setPiaOpen] = useState(false);
  const [torOpen, setTorOpen] = useState(false);
  const [advSettings, setAdvSettings] = useState<AdvKey>('xraySetting');
  const [resetOpen, setResetOpen] = useState(false);
  const location = useLocation();
  const navigate = useNavigate();
  // The Tor and PIA page links here with ?open=pia to open that dialog over the outbounds.
  const openParam = new URLSearchParams(location.search).get('open');
  const [openedFor, setOpenedFor] = useState('');
  if (openParam && openedFor !== location.key) {
    setOpenedFor(location.key);
    if (openParam === 'pia') setPiaOpen(true);
    if (openParam === 'tor') setTorOpen(true);
  }
  const pathSection =
    location.pathname === '/outbound'
      ? 'outbound'
      : location.pathname === '/routing'
        ? 'routing'
        : '';
  const sectionSlug = pathSection || location.hash.replace(/^#/, '');
  const activeSection = SECTION_SLUGS.includes(sectionSlug) ? sectionSlug : 'advanced';

  const mutate = useCallback(
    (mutator: (next: XraySettingsValue) => void) => {
      setTemplateSettings((prev) => {
        if (!prev) return prev;
        const clone = JSON.parse(JSON.stringify(prev)) as XraySettingsValue;
        mutator(clone);
        return clone;
      });
    },
    [setTemplateSettings],
  );

  async function onTestOutbound(idx: number, mode: string) {
    const outbound = templateSettings?.outbounds?.[idx];
    if (outbound) await testOutbound(idx, outbound, mode);
  }

  async function onTestSubscription(outbound: Record<string, unknown>, mode: string) {
    const tag = typeof outbound?.tag === 'string' ? outbound.tag : '';
    if (tag) await testSubscriptionOutbound(tag, outbound, mode);
  }

  function onAddOutbound(outbound: Record<string, unknown>) {
    mutate((tt) => {
      if (!Array.isArray(tt.outbounds)) tt.outbounds = [];
      tt.outbounds.push(outbound as never);
    });
  }
  function onResetOutbound(payload: {
    index: number;
    outbound: Record<string, unknown>;
    oldTag?: string;
    newTag?: string;
  }) {
    mutate((tt) => {
      if (!tt.outbounds || payload.index < 0) return;
      tt.outbounds[payload.index] = payload.outbound as never;
      if (payload.oldTag && payload.newTag) {
        propagateOutboundTagRename(tt, payload.oldTag, payload.newTag);
      }
    });
  }
  function onRemoveOutboundByTag(tag: string) {
    mutate((tt) => {
      if (!tt.outbounds) return;
      const idx = tt.outbounds.findIndex((o) => o?.tag === tag);
      if (idx >= 0) tt.outbounds.splice(idx, 1);
    });
  }
  const advancedText = useMemo(() => {
    if (advSettings === 'xraySetting') return xraySetting;
    const tpl = templateSettings;
    if (!tpl) return '';
    try {
      switch (advSettings) {
        case 'inboundSettings':
          return JSON.stringify(tpl.inbounds || [], null, 2);
        case 'outboundSettings':
          return JSON.stringify(tpl.outbounds || [], null, 2);
        case 'routingRuleSettings':
          return JSON.stringify(tpl.routing?.rules || [], null, 2);
        default:
          return '';
      }
    } catch {
      return '';
    }
  }, [advSettings, xraySetting, templateSettings]);

  const jsonError = useMemo(() => {
    try {
      JSON.parse(xraySetting);
      return '';
    } catch (e) {
      return (e as Error).message;
    }
  }, [xraySetting]);
  // Tunnel, Tor and subscription outbounds are written by the panel; edits to them here are replaced.
  const managedInJson = useMemo(
    () => injectedTags.filter((tag) => xraySetting.includes(`"${tag}"`)),
    [injectedTags, xraySetting],
  );

  function onAdvancedTextChange(next: string) {
    if (advSettings === 'xraySetting') {
      setXraySetting(next);
      return;
    }
    let parsed;
    try {
      parsed = JSON.parse(next);
    } catch {
      return;
    }
    mutate((tt) => {
      switch (advSettings) {
        case 'inboundSettings':
          tt.inbounds = parsed;
          break;
        case 'outboundSettings':
          tt.outbounds = parsed;
          break;
        case 'routingRuleSettings':
          if (!tt.routing) tt.routing = {};
          tt.routing.rules = parsed;
          break;
      }
    });
  }

  function onSaveAll() {
    try {
      JSON.parse(xraySetting);
    } catch (e) {
      messageApi.error(`Advanced JSON: ${(e as Error).message}`);
      navigate('/xray#advanced');
      return;
    }
    if (templateSettings) {
      const clone = JSON.parse(JSON.stringify(templateSettings));
      ensureMissingBalancerLoopbacks(clone);
      cleanupOrphanedBalancerLoopbacks(clone);
      const cycles = detectBalancerCycles(clone);
      if (cycles.length > 0) {
        const names = cycles.map((c) => c.join(' → ')).join(', ');
        messageApi.error(t('pages.xray.balancer.balancerFallbackCycle') + ' (' + names + ')');
        return;
      }
      const serialized = JSON.stringify(clone, null, 2);
      setXraySetting(serialized);
      setTemplateSettings(clone);
    }
    saveAll();
  }

  const sectionBody = (() => {
    switch (activeSection) {
      case 'routing':
        return (
          <RoutingTab
            variant="page"
            templateSettings={templateSettings}
            setTemplateSettings={setTemplateSettings}
            inboundTags={inboundTags}
            clientReverseTags={clientReverseTags}
            subscriptionOutboundTags={injectedTags}
            isMobile={isMobile}
          />
        );
      case 'outbound':
        return (
          <OutboundsTab
            templateSettings={templateSettings}
            setTemplateSettings={setTemplateSettings}
            outboundsTraffic={outboundsTraffic}
            outboundTestStates={outboundTestStates}
            subscriptionTestStates={subscriptionTestStates}
            testingAll={testingAll}
            inboundTags={inboundTags}
            subscriptionOutbounds={subscriptionOutbounds}
            tunnelOutbounds={tunnelOutbounds}
            torOutbounds={torOutbounds}
            torStatus={torStatus}
            subscriptionOutboundTags={injectedTags}
            isMobile={isMobile}
            onResetTraffic={resetOutboundsTraffic}
            onTest={onTestOutbound}
            onTestSubscription={onTestSubscription}
            onTestAll={testAllOutbounds}
            onShowWarp={() => setWarpOpen(true)}
            onShowNord={() => setNordOpen(true)}
            onShowPia={() => setPiaOpen(true)}
            onShowTor={() => setTorOpen(true)}
            onRefreshXrayData={fetchAll}
            onEditJson={() => navigate('/xray#advanced')}
            rulesTile={
              <RoutingTab
                variant="tile"
                templateSettings={templateSettings}
                setTemplateSettings={setTemplateSettings}
                inboundTags={inboundTags}
                clientReverseTags={clientReverseTags}
                subscriptionOutboundTags={injectedTags}
                isMobile={isMobile}
              />
            }
          />
        );
      case 'balancer':
        return (
          <BalancersTab
            templateSettings={templateSettings}
            setTemplateSettings={setTemplateSettings}
            clientReverseTags={clientReverseTags}
            subscriptionOutboundTags={injectedTags}
            isMobile={isMobile}
          />
        );
      case 'dns':
        return (
          <DnsTab templateSettings={templateSettings} setTemplateSettings={setTemplateSettings} />
        );
      case 'advanced':
        return (
          <div className="grid">
            <section className="tile s8 dense" style={{ padding: 0 }}>
              <div
                className="row sp wrap"
                style={{ padding: '12px 16px', borderBottom: '1px solid var(--glass-b)', gap: 8 }}
              >
                <Seg
                  label={t('pages.xray.Template')}
                  value={advSettings}
                  onChange={setAdvSettings}
                  options={[
                    { value: 'xraySetting', label: t('pages.xray.completeTemplate') },
                    { value: 'inboundSettings', label: t('pages.xray.Inbounds') },
                    { value: 'outboundSettings', label: t('pages.xray.Outbounds') },
                    { value: 'routingRuleSettings', label: t('pages.xray.Routings') },
                  ]}
                />
                <span className="mut num" style={{ fontSize: 12, fontWeight: 650 }}>
                  {t('xrayPage.lines', { n: advancedText.split('\n').length })}
                </span>
              </div>
              <div style={{ padding: 12 }}>
                <JsonEditor
                  value={advancedText}
                  onChange={onAdvancedTextChange}
                  minHeight="420px"
                  maxHeight="720px"
                />
              </div>
            </section>
            <section className="tile s4 dense" style={{ alignSelf: 'start' }}>
              <TileHead icon="info" title={t('xrayPage.check')} />
              <div className="list">
                {jsonError ? (
                  <div className="li">
                    <span className="ico bad">
                      <Icon name="alert" />
                    </span>
                    <div className="f1">
                      <b>{t('xrayPage.jsonInvalid')}</b>
                      <div className="mut mono" style={{ fontSize: 12 }}>
                        {jsonError}
                      </div>
                    </div>
                  </div>
                ) : (
                  <div className="li">
                    <span className="ico ok">
                      <Icon name="okc" />
                    </span>
                    <div className="f1">
                      <b>{t('xrayPage.jsonValid')}</b>
                    </div>
                  </div>
                )}
                {managedInJson.map((tag) => (
                  <div className="li" key={tag}>
                    <span className="ico warn">
                      <Icon name="alert" />
                    </span>
                    <div className="f1">
                      <b className="mono">{tag}</b>
                      <div className="mut" style={{ fontSize: 12.5 }}>
                        {t('xrayPage.managed')}
                      </div>
                    </div>
                  </div>
                ))}
                <div className="li">
                  <span className="ico">
                    <Icon name="info" />
                  </span>
                  <div className="f1 mut" style={{ fontSize: 12.5 }}>
                    {t('pages.xray.TemplateDesc')}
                  </div>
                </div>
              </div>
            </section>
          </div>
        );
      default:
        return (
          <BasicsTab
            templateSettings={templateSettings}
            setTemplateSettings={setTemplateSettings}
            outboundTestUrl={outboundTestUrl}
            onChangeOutboundTestUrl={setOutboundTestUrl}
            onResetDefault={resetToDefault}
          />
        );
    }
  })();

  const saveBar = !saveDisabled && (
    <div
      className="tile edge st-warn row sp wrap"
      role="status"
      style={{ position: 'sticky', bottom: 16, zIndex: 5, padding: '12px 16px', gap: 10 }}
    >
      <span style={{ fontWeight: 650 }}>{t('outbounds.unsaved')}</span>
      <Btn kind="pri" icon="check" disabled={spinning} onClick={onSaveAll}>
        {t('pages.xray.save')}
      </Btn>
    </div>
  );
  const overrideModals = (
    <>
      <WarpModal
        open={warpOpen}
        templateSettings={templateSettings}
        onClose={() => setWarpOpen(false)}
        onAddOutbound={onAddOutbound}
        onResetOutbound={onResetOutbound}
        onRemoveOutbound={onRemoveOutboundByTag}
      />
      <NordModal
        open={nordOpen}
        templateSettings={templateSettings}
        onClose={() => setNordOpen(false)}
        onAddOutbound={onAddOutbound}
        onResetOutbound={onResetOutbound}
      />
      <TorModal open={torOpen} onClose={() => setTorOpen(false)} onChanged={fetchAll} />
      <PiaModal
        open={piaOpen}
        templateSettings={templateSettings}
        onClose={() => setPiaOpen(false)}
        onAddOutbound={onAddOutbound}
        onResetOutbound={onResetOutbound}
      />
    </>
  );

  {
    return (
      <div className="col stag" style={{ gap: 18 }}>
        {messageContextHolder}
        {!fetched ? (
          <section className="tile">
            <Sk h={160} />
          </section>
        ) : fetchError ? (
          <section className="tile edge st-bad">
            <DlEmpty
              art={<IlloOffline />}
              title={t('somethingWentWrong')}
              actions={
                <Btn kind="pri" icon="refresh" onClick={fetchAll}>
                  {t('check')}
                </Btn>
              }
            >
              {fetchError}
            </DlEmpty>
          </section>
        ) : activeSection === 'routing' ? (
          <>
            <div className="pagehead">
              <div>
                <h2 className="h2" style={{ fontSize: 28 }}>
                  {t('menu.routing')}
                </h2>
                <p>{t('routing.subtitle')}</p>
              </div>
            </div>
            {sectionBody}
          </>
        ) : activeSection === 'outbound' ? (
          sectionBody
        ) : (
          <>
            <div className="pagehead">
              <div>
                <h2 className="h2" style={{ fontSize: 28 }}>
                  {t('menu.xray')}
                </h2>
                <p>{t('xrayPage.subtitle')}</p>
              </div>
              <div className="row wrap">
                <Btn
                  icon="down"
                  onClick={() =>
                    FileManager.downloadTextFile(xraySetting, 'config.json', {
                      type: 'application/json',
                    })
                  }
                >
                  {t('download')}
                </Btn>
                <Btn icon="refresh" onClick={() => setResetOpen(true)}>
                  {t('xrayPage.reset')}
                </Btn>
                <Btn
                  kind="pri"
                  icon="check"
                  disabled={saveDisabled || spinning}
                  onClick={onSaveAll}
                >
                  {t('pages.xray.save')}
                </Btn>
              </div>
            </div>
            <div>
              <Seg
                label={t('xrayPage.sections')}
                value={activeSection}
                onChange={(v) => navigate(`/xray#${v}`)}
                options={[
                  { value: 'advanced', label: t('xrayPage.json') },
                  { value: 'basic', label: t('pages.xray.basicTemplate') },
                  { value: 'balancer', label: t('xrayPage.balancers') },
                  { value: 'dns', label: 'DNS' },
                ]}
              />
            </div>
            {activeSection === 'advanced' ? (
              sectionBody
            ) : (
              <section className="tile">{sectionBody}</section>
            )}
          </>
        )}
        {saveBar}
        {overrideModals}
        <DlModal
          open={resetOpen}
          onClose={() => setResetOpen(false)}
          label={t('pages.settings.resetDefaultConfig')}
          width={460}
        >
          <h2 className="h3">{t('pages.settings.resetDefaultConfig')}</h2>
          <div className="mfoot">
            <Btn kind="ghost" onClick={() => setResetOpen(false)}>
              {t('cancel')}
            </Btn>
            <Btn
              kind="dng"
              icon="refresh"
              onClick={() => {
                setResetOpen(false);
                resetToDefault();
              }}
            >
              {t('reset')}
            </Btn>
          </div>
        </DlModal>
      </div>
    );
  }
}
