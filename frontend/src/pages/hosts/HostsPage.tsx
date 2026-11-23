import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { message } from 'antd';

import Icon from '@/components/dl/Icon';
import { IlloEmpty, IlloOffline } from '@/components/dl/Illustrations';
import { Modal as DlModal } from '@/components/dl/overlays';
import { Btn, Chip, Empty as DlEmpty, Sk } from '@/components/dl/primitives';
import { fmtInt } from '@/components/dl/numbers';
import { useMediaQuery } from '@/hooks/useMediaQuery';
import { useHostsQuery, type HostRecord } from '@/api/queries/useHostsQuery';
import { useHostMutations } from '@/api/queries/useHostMutations';
import { useInboundOptions } from '@/api/queries/useInboundOptions';
import { setMessageInstance } from '@/utils/messageBus';
import type { BulkAddHostValues } from '@/schemas/api/host';
import HostList, { sortHosts } from './HostList';
import HostFormModal from './HostFormModal';

export default function HostsPage() {
  const { t } = useTranslation();
  const { isMobile } = useMediaQuery();
  const [messageApi, messageContextHolder] = message.useMessage();
  useEffect(() => {
    setMessageInstance(messageApi);
  }, [messageApi]);

  const { hosts, loading, fetched, fetchError, refetch } = useHostsQuery();
  const { bulkCreate, update, remove, setEnable, reorder, bulkSetEnable, bulkDel } =
    useHostMutations();
  const { data: inboundOptions = [] } = useInboundOptions();

  const [formOpen, setFormOpen] = useState(false);
  const [formMode, setFormMode] = useState<'add' | 'edit'>('add');
  const [formHost, setFormHost] = useState<HostRecord | null>(null);
  const [selectedGroupIds, setSelectedGroupIds] = useState<string[]>([]);
  const [pendingOrder, setPendingOrder] = useState<string[] | null>(null);
  const savingOrder = useRef(false);
  const visibleHosts = useMemo(
    () =>
      pendingOrder
        ? hosts.map((h) => ({ ...h, sortOrder: pendingOrder.indexOf(h.groupId) }))
        : hosts,
    [hosts, pendingOrder],
  );

  const onReorder = useCallback(
    async (groupIds: string[]) => {
      const current = sortHosts(hosts).map((h) => h.groupId);
      if (savingOrder.current || current.every((id, i) => id === groupIds[i])) return;
      savingOrder.current = true;
      setPendingOrder(groupIds);
      try {
        const msg = await reorder(groupIds);
        if (msg?.success) messageApi.success(t('pages.hosts.toasts.update'));
        else messageApi.error(msg?.msg || t('somethingWentWrong'));
      } catch {
        messageApi.error(t('somethingWentWrong'));
      } finally {
        savingOrder.current = false;
        setPendingOrder(null);
      }
    },
    [hosts, reorder, messageApi, t],
  );

  const onAdd = useCallback(() => {
    setFormMode('add');
    setFormHost(null);
    setFormOpen(true);
  }, []);

  const onEdit = useCallback((host: HostRecord) => {
    setFormMode('edit');
    setFormHost({ ...host });
    setFormOpen(true);
  }, []);

  const onSave = useCallback(
    async (payload: BulkAddHostValues) => {
      if (formMode === 'edit' && formHost?.groupId) {
        return update(formHost.groupId, payload);
      }
      return bulkCreate(payload);
    },
    [formMode, formHost, update, bulkCreate],
  );

  const [confirm, setConfirm] = useState<{ title: string; run: () => Promise<void> } | null>(null);
  const onDelete = useCallback(
    (host: HostRecord) =>
      setConfirm({
        title: t('pages.hosts.deleteConfirmTitle', { name: host.remark }),
        run: async () => {
          const msg = await remove(host.groupId);
          if (msg?.success) messageApi.success(t('pages.hosts.toasts.delete'));
        },
      }),
    [t, remove, messageApi],
  );

  const onToggleEnable = useCallback(
    async (host: HostRecord, next: boolean) => {
      await setEnable(host.groupId, next);
    },
    [setEnable],
  );

  const onMove = useCallback(
    async (host: HostRecord, dir: 'up' | 'down') => {
      const sorted = sortHosts(hosts);
      const idx = sorted.findIndex((h) => h.groupId === host.groupId);
      const swapWith = dir === 'up' ? idx - 1 : idx + 1;
      if (idx < 0 || swapWith < 0 || swapWith >= sorted.length) return;
      const groupIds = sorted.map((h) => h.groupId);
      [groupIds[idx], groupIds[swapWith]] = [groupIds[swapWith], groupIds[idx]];
      await onReorder(groupIds);
    },
    [hosts, onReorder],
  );

  const onBulkEnable = useCallback(
    async (enable: boolean) => {
      if (selectedGroupIds.length === 0) return;
      const msg = await bulkSetEnable(selectedGroupIds, enable);
      if (msg?.success) setSelectedGroupIds([]);
    },
    [selectedGroupIds, bulkSetEnable],
  );

  const onBulkDelete = useCallback(() => {
    if (selectedGroupIds.length === 0) return;
    setConfirm({
      title: t('pages.hosts.bulkDeleteConfirm', { count: selectedGroupIds.length }),
      run: async () => {
        const msg = await bulkDel(selectedGroupIds);
        if (msg?.success) {
          messageApi.success(t('pages.hosts.toasts.delete'));
          setSelectedGroupIds([]);
        }
      },
    });
  }, [selectedGroupIds, t, bulkDel, messageApi]);

  const [filter, setFilter] = useState<'all' | 'on' | 'off'>('all');
  const counts = {
    all: hosts.length,
    on: hosts.filter((h) => !h.isDisabled).length,
    off: hosts.filter((h) => h.isDisabled).length,
  };
  const shownHosts = visibleHosts.filter((h) =>
    filter === 'all' ? true : filter === 'on' ? !h.isDisabled : h.isDisabled,
  );

  return (
    <div className="col stag" style={{ gap: 18 }}>
      {messageContextHolder}
      <div className="pagehead">
        <div>
          <h2 className="h2" style={{ fontSize: 28 }}>
            {t('menu.hosts')}
          </h2>
          <p>{t('hosts.subtitle')}</p>
        </div>
        <div className="row wrap">
          <Btn kind="pri" icon="plus" onClick={onAdd}>
            {t('pages.hosts.addHost')}
          </Btn>
        </div>
      </div>
      <section className="tile" style={{ display: 'flex', gap: 14, alignItems: 'flex-start' }}>
        <span className="ico acc">
          <Icon name="info" />
        </span>
        <div>
          <div className="h3">{t('hosts.tipTitle')}</div>
          <p className="mut" style={{ margin: '4px 0 0' }}>
            {t('hosts.tipBody')}
          </p>
        </div>
      </section>
      {!fetched ? (
        <section className="tile">
          <Sk h={140} />
        </section>
      ) : fetchError ? (
        <section className="tile edge st-bad">
          <DlEmpty
            art={<IlloOffline />}
            title={t('somethingWentWrong')}
            actions={
              <Btn kind="pri" icon="refresh" disabled={loading} onClick={() => refetch()}>
                {t('refresh')}
              </Btn>
            }
          >
            {fetchError}
          </DlEmpty>
        </section>
      ) : hosts.length === 0 ? (
        <section className="tile">
          <DlEmpty
            art={<IlloEmpty size={130} />}
            title={t('hosts.empty')}
            actions={
              <Btn kind="pri" icon="plus" onClick={onAdd}>
                {t('pages.hosts.addHost')}
              </Btn>
            }
          />
        </section>
      ) : (
        <>
          <div className="chips" role="group" aria-label={t('hosts.filters')}>
            {(['all', 'on', 'off'] as const).map((f) => (
              <Chip
                key={f}
                on={filter === f}
                count={fmtInt(counts[f])}
                onClick={() => setFilter(f)}
              >
                {t(`hosts.filter.${f}`)}
              </Chip>
            ))}
          </div>
          <HostList
            hosts={shownHosts}
            inboundOptions={inboundOptions}
            loading={loading}
            isMobile={isMobile}
            selectedGroupIds={selectedGroupIds}
            onSelectionChange={setSelectedGroupIds}
            onAdd={onAdd}
            onEdit={onEdit}
            onDelete={onDelete}
            onToggleEnable={onToggleEnable}
            onMove={onMove}
            onReorder={onReorder}
            reordering={pendingOrder !== null}
            onBulkEnable={onBulkEnable}
            onBulkDelete={onBulkDelete}
          />
        </>
      )}
      <DlModal
        open={confirm !== null}
        onClose={() => setConfirm(null)}
        label={confirm?.title ?? ''}
        width={440}
      >
        <div className="row" style={{ gap: 12 }}>
          <span className="ico bad">
            <Icon name="trash" />
          </span>
          <h2 className="h3 f1">{confirm?.title}</h2>
        </div>
        <div className="mfoot">
          <Btn kind="ghost" onClick={() => setConfirm(null)}>
            {t('cancel')}
          </Btn>
          <Btn
            kind="dng"
            icon="trash"
            onClick={async () => {
              const run = confirm?.run;
              setConfirm(null);
              await run?.();
            }}
          >
            {t('delete')}
          </Btn>
        </div>
      </DlModal>
      <HostFormModal
        open={formOpen}
        mode={formMode}
        host={formHost}
        inboundOptions={inboundOptions}
        existingHosts={hosts}
        save={onSave}
        onOpenChange={setFormOpen}
      />
    </div>
  );
}
