import { lazy, useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router';
import { Button, ConfigProvider, Modal, message } from 'antd';
import { CopyOutlined, CloudDownloadOutlined } from '@ant-design/icons';

import { HttpUtil, ClipboardManager, FileManager } from '@/utils';
import { useTheme } from '@/hooks/useTheme';
import { useStatusQuery } from '@/api/queries/useStatusQuery';
import { useMediaQuery } from '@/hooks/useMediaQuery';
import { LazyMount } from '@/components/utility';
import { setMessageInstance } from '@/utils/messageBus';
import type { PanelUpdateInfo } from './PanelUpdateModal';
import DashboardPage from './dashboard/DashboardPage';
import { ToolsTile } from './dashboard/ToolsTile';
import { Btn, Empty } from '@/components/dl/primitives';
import { Modal as DlModal } from '@/components/dl/overlays';
import { IlloOffline } from '@/components/dl/Illustrations';
import Mark from '@/components/dl/Mark';
const JsonEditor = lazy(() => import('@/components/form/JsonEditor'));
const PanelUpdateModal = lazy(() => import('./PanelUpdateModal'));
const BackupModal = lazy(() => import('./BackupModal'));
const SystemHistoryModal = lazy(() => import('./SystemHistoryModal'));
const XrayMetricsModal = lazy(() => import('./XrayMetricsModal'));
const AmneziaWGLogModal = lazy(() => import('./AmneziaWGLogModal'));
const VersionModal = lazy(() => import('./VersionModal'));

export default function IndexPage() {
  const { t } = useTranslation();
  const { antdThemeConfig } = useTheme();
  const { status, fetched, fetchError, refresh } = useStatusQuery();
  const { isMobile } = useMediaQuery();
  const [messageApi, messageContextHolder] = message.useMessage();
  useEffect(() => {
    setMessageInstance(messageApi);
  }, [messageApi]);

  const [devChannelEnable, setDevChannelEnable] = useState(false);
  const [panelUpdateInfo, setPanelUpdateInfo] = useState<PanelUpdateInfo>({
    currentVersion: '',
    latestVersion: '',
    updateAvailable: false,
  });

  const basePath = window.X_UI_BASE_PATH || '';

  const [backupOpen, setBackupOpen] = useState(false);
  const [panelUpdateOpen, setPanelUpdateOpen] = useState(false);
  const [sysHistoryOpen, setSysHistoryOpen] = useState(false);
  const [xrayMetricsOpen, setXrayMetricsOpen] = useState(false);
  const [amneziawgLogsOpen, setAmneziawgLogsOpen] = useState(false);
  const [versionOpen, setVersionOpen] = useState(false);
  const [configTextOpen, setConfigTextOpen] = useState(false);
  const [configText, setConfigText] = useState('');
  const [loading, setLoading] = useState(false);
  const [loadingTip, setLoadingTip] = useState(t('loading'));
  const navigate = useNavigate();

  useEffect(() => {
    HttpUtil.post<{ devChannelEnable?: boolean }>('/panel/api/setting/defaultSettings').then(
      (msg) => {
        if (msg?.success && msg.obj) {
          setDevChannelEnable(!!msg.obj.devChannelEnable);
        }
      },
    );
    HttpUtil.get<PanelUpdateInfo>('/panel/api/server/getPanelUpdateInfo').then((msg) => {
      if (msg?.success && msg.obj) setPanelUpdateInfo(msg.obj);
    });
  }, []);

  const displayVersion = useMemo(
    () => window.X_UI_CUR_VER || panelUpdateInfo.currentVersion || '?',
    [panelUpdateInfo.currentVersion],
  );

  const setBusy = useCallback(({ busy, tip }: { busy: boolean; tip?: string }) => {
    setLoading(busy);
    if (tip) setLoadingTip(tip);
  }, []);

  async function handleChannelChange(dev: boolean) {
    const res = await HttpUtil.post('/panel/api/server/setUpdateChannel', { dev });
    if (!res?.success) return;
    setDevChannelEnable(dev);
    const msg = await HttpUtil.get<PanelUpdateInfo>('/panel/api/server/getPanelUpdateInfo');
    if (msg?.success && msg.obj) setPanelUpdateInfo(msg.obj);
  }

  async function openConfig() {
    setLoading(true);
    try {
      const msg = await HttpUtil.get('/panel/api/server/getConfigJson');
      if (!msg?.success) return;
      setConfigText(JSON.stringify(msg.obj, null, 2));
      setConfigTextOpen(true);
    } finally {
      setLoading(false);
    }
  }

  async function copyConfig() {
    const ok = await ClipboardManager.copyText(configText || '');
    if (ok) messageApi.success(t('copied'));
  }

  function downloadConfig() {
    FileManager.downloadTextFile(configText, 'config.json');
  }

  return (
    <ConfigProvider theme={antdThemeConfig}>
      {messageContextHolder}
      <DlModal open={loading} onClose={() => {}} label={loadingTip} width={420}>
        <div className="row" aria-live="polite">
          <Mark size={40} motion="loop" />
          <b>{loadingTip}</b>
        </div>
      </DlModal>
      {!fetched ? (
        <div className="grid" aria-busy="true" aria-label={t('loading')}>
          {[0, 1, 2].map((i) => (
            <section key={i} className="tile s4 col" style={{ gap: 14 }}>
              <div className="row sp">
                <div className="sk" style={{ height: 16, width: '40%' }} />
                <div className="sk" style={{ height: 22, width: 70, borderRadius: 99 }} />
              </div>
              <div className="sk" style={{ height: 40, width: '55%' }} />
              <div className="sk" style={{ height: 56 }} />
              <div className="sk" style={{ height: 8, borderRadius: 99 }} />
            </section>
          ))}
        </div>
      ) : fetchError ? (
        <section className="tile edge st-bad">
          <Empty
            art={<IlloOffline />}
            title={t('somethingWentWrong')}
            actions={
              <Btn kind="pri" icon="refresh" onClick={refresh}>
                {t('refresh')}
              </Btn>
            }
          >
            {fetchError}
          </Empty>
        </section>
      ) : (
        <DashboardPage
          tools={
            <ToolsTile
              panelVersion={displayVersion}
              latestVersion={panelUpdateInfo.latestVersion}
              updateAvailable={panelUpdateInfo.updateAvailable}
              xrayVersion={status.xray.version}
              actions={{
                onLogs: () => navigate('/logs'),
                onAmneziaWGLogs: status.amneziawg.configured
                  ? () => setAmneziawgLogsOpen(true)
                  : undefined,
                onConfig: openConfig,
                onBackup: () => setBackupOpen(true),
                onHistory: () => setSysHistoryOpen(true),
                onMetrics: () => setXrayMetricsOpen(true),
                onUpdate: () => setPanelUpdateOpen(true),
                onVersion: () => setVersionOpen(true),
              }}
            />
          }
        />
      )}

      <LazyMount when={panelUpdateOpen}>
        <PanelUpdateModal
          open={panelUpdateOpen}
          info={panelUpdateInfo}
          devChannelEnable={devChannelEnable}
          onChannelChange={handleChannelChange}
          onClose={() => setPanelUpdateOpen(false)}
          onBusy={setBusy}
        />
      </LazyMount>
      <LazyMount when={backupOpen}>
        <BackupModal
          open={backupOpen}
          basePath={basePath}
          onClose={() => setBackupOpen(false)}
          onBusy={setBusy}
        />
      </LazyMount>
      <LazyMount when={sysHistoryOpen}>
        <SystemHistoryModal
          open={sysHistoryOpen}
          status={status}
          onClose={() => setSysHistoryOpen(false)}
        />
      </LazyMount>
      <LazyMount when={xrayMetricsOpen}>
        <XrayMetricsModal open={xrayMetricsOpen} onClose={() => setXrayMetricsOpen(false)} />
      </LazyMount>
      <LazyMount when={amneziawgLogsOpen}>
        <AmneziaWGLogModal open={amneziawgLogsOpen} onClose={() => setAmneziawgLogsOpen(false)} />
      </LazyMount>
      <LazyMount when={versionOpen}>
        <VersionModal
          open={versionOpen}
          status={status}
          onClose={() => setVersionOpen(false)}
          onBusy={setBusy}
        />
      </LazyMount>

      <LazyMount when={configTextOpen}>
        <Modal
          open={configTextOpen}
          title={t('pages.index.config')}
          width={isMobile ? '100%' : 900}
          style={isMobile ? { top: 20, maxWidth: 'calc(100vw - 16px)' } : { top: 20 }}
          onCancel={() => setConfigTextOpen(false)}
          footer={[
            <Button
              key="download"
              onClick={downloadConfig}
              size={isMobile ? 'small' : 'middle'}
              icon={<CloudDownloadOutlined aria-hidden />}
            >
              {isMobile ? 'Download' : 'config.json'}
            </Button>,
            <Button
              key="copy"
              type="primary"
              onClick={copyConfig}
              size={isMobile ? 'small' : 'middle'}
              icon={<CopyOutlined aria-hidden />}
            >
              Copy
            </Button>,
          ]}
        >
          <JsonEditor
            value={configText}
            onChange={setConfigText}
            minHeight={isMobile ? '300px' : 'calc(100vh - 220px)'}
            maxHeight={isMobile ? '70vh' : 'calc(100vh - 220px)'}
            readOnly
          />
        </Modal>
      </LazyMount>
    </ConfigProvider>
  );
}
