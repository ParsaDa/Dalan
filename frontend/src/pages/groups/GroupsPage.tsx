import { lazy, useCallback, useEffect, useId, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { message } from 'antd';

import Icon from '@/components/dl/Icon';
import { IlloEmpty, IlloOffline } from '@/components/dl/Illustrations';
import type { MenuAction } from '@/components/dl/MoreMenu';
import { FormModal, Modal as DlModal } from '@/components/dl/overlays';
import { Btn, Empty as DlEmpty, Sk } from '@/components/dl/primitives';
import GroupCard from './GroupCard';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { usePageTitle } from '@/hooks/usePageTitle';
import { useClients } from '@/hooks/useClients';
import { HttpUtil } from '@/utils';
import { setMessageInstance } from '@/utils/messageBus';
import { LazyMount } from '@/components/utility';
import { keys } from '@/api/queryKeys';
import {
  ClientRecordSchema,
  GroupSummaryListSchema,
  type ClientRecord,
  type GroupSummary,
} from '@/schemas/client';
import { parseMsg } from '@/utils/zodValidate';

const ClientRecordListSchema = z
  .array(ClientRecordSchema)
  .nullable()
  .transform((v) => v ?? []);

const SubLinksModal = lazy(() => import('../clients/SubLinksModal'));
const ClientBulkAdjustModal = lazy(() => import('../clients/ClientBulkAdjustModal'));
const GroupAddClientsModal = lazy(() => import('./GroupAddClientsModal'));
const GroupRemoveClientsModal = lazy(() => import('./GroupRemoveClientsModal'));

const JSON_HEADERS = { headers: { 'Content-Type': 'application/json' } } as const;

async function fetchGroups(): Promise<GroupSummary[]> {
  const msg = await HttpUtil.get('/panel/api/clients/groups', undefined, { silent: true });
  if (!msg?.success) throw new Error(msg?.msg || 'Failed to load groups');
  const validated = parseMsg(msg, GroupSummaryListSchema, 'clients/groups');
  return validated.obj ?? [];
}

async function fetchEmailsForGroup(name: string): Promise<string[]> {
  const msg = await HttpUtil.get<string[]>(
    `/panel/api/clients/groups/${encodeURIComponent(name)}/emails`,
    undefined,
    { silent: true },
  );
  if (!msg?.success || !Array.isArray(msg.obj)) return [];
  return msg.obj;
}

export default function GroupsPage() {
  usePageTitle();
  const { t } = useTranslation();
  const uid = useId();
  const [messageApi, messageContextHolder] = message.useMessage();
  useEffect(() => {
    setMessageInstance(messageApi);
  }, [messageApi]);
  const queryClient = useQueryClient();

  const { subSettings, bulkAdjust, bulkAddToGroup, bulkRemoveFromGroup, bulkDelete } = useClients({
    list: false,
  });

  const groupsQuery = useQuery({
    queryKey: keys.clients.groups(),
    queryFn: fetchGroups,
  });
  const groups = useMemo(() => groupsQuery.data ?? [], [groupsQuery.data]);
  const loading = groupsQuery.isFetching;
  const fetched = groupsQuery.data !== undefined || groupsQuery.isError;
  const fetchError = groupsQuery.error ? (groupsQuery.error as Error).message : '';

  const invalidate = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: keys.clients.root() });
  }, [queryClient]);

  const createMut = useMutation({
    mutationFn: (body: { name: string }) =>
      HttpUtil.post('/panel/api/clients/groups/create', body, JSON_HEADERS),
    onSuccess: (msg) => {
      if (msg?.success) invalidate();
    },
  });

  const renameMut = useMutation({
    mutationFn: (body: { oldName: string; newName: string }) =>
      HttpUtil.post('/panel/api/clients/groups/rename', body, JSON_HEADERS),
    onSuccess: (msg) => {
      if (msg?.success) invalidate();
    },
  });

  const deleteMut = useMutation({
    mutationFn: (body: { name: string }) =>
      HttpUtil.post('/panel/api/clients/groups/delete', body, JSON_HEADERS),
    onSuccess: (msg) => {
      if (msg?.success) invalidate();
    },
  });

  const groupResetMut = useMutation({
    mutationFn: (body: { name: string }) =>
      HttpUtil.post('/panel/api/clients/groups/resetTraffic', body, JSON_HEADERS),
    onSuccess: (msg) => {
      if (msg?.success) invalidate();
    },
  });

  const [confirm, setConfirm] = useState<{
    title: string;
    content: string;
    danger: string;
    run: () => Promise<void>;
  } | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [createName, setCreateName] = useState('');

  const [renameOpen, setRenameOpen] = useState(false);
  const [renameTarget, setRenameTarget] = useState<GroupSummary | null>(null);
  const [renameValue, setRenameValue] = useState('');

  const [subLinksOpen, setSubLinksOpen] = useState(false);
  const [adjustOpen, setAdjustOpen] = useState(false);
  const [addClientsOpen, setAddClientsOpen] = useState(false);
  const [removeClientsOpen, setRemoveClientsOpen] = useState(false);
  const [groupEmails, setGroupEmails] = useState<string[]>([]);
  const [groupForAction, setGroupForAction] = useState<GroupSummary | null>(null);

  const allClientsQuery = useQuery<ClientRecord[]>({
    queryKey: keys.clients.all(),
    queryFn: async () => {
      const msg = await HttpUtil.get('/panel/api/clients/list', undefined, { silent: true });
      if (!msg?.success) throw new Error(msg?.msg || 'Failed to load clients');
      const validated = parseMsg(msg, ClientRecordListSchema, 'clients/list');
      return validated.obj ?? [];
    },
    enabled: addClientsOpen || removeClientsOpen || subLinksOpen,
    staleTime: 30_000,
  });
  const allClients = allClientsQuery.data ?? [];

  const totalTraffic = useMemo(
    () => groups.reduce((acc, g) => acc + (g.trafficUsed || 0), 0),
    [groups],
  );

  function openCreate() {
    setCreateName('');
    setCreateOpen(true);
  }

  async function confirmCreate() {
    const name = createName.trim();
    if (!name) return;
    if (groups.some((g) => g.name.toLowerCase() === name.toLowerCase())) {
      messageApi.error(t('pages.groups.renameCollision', { name }));
      return;
    }
    const msg = await createMut.mutateAsync({ name });
    if (msg?.success) {
      messageApi.success(t('pages.groups.createSuccess', { name }));
      setCreateOpen(false);
    }
  }

  function openRename(g: GroupSummary) {
    setRenameTarget(g);
    setRenameValue(g.name);
    setRenameOpen(true);
  }

  async function confirmRename() {
    if (!renameTarget) return;
    const next = renameValue.trim();
    if (!next || next === renameTarget.name) {
      setRenameOpen(false);
      return;
    }
    if (
      groups.some(
        (g) => g.name.toLowerCase() === next.toLowerCase() && g.name !== renameTarget.name,
      )
    ) {
      messageApi.error(t('pages.groups.renameCollision', { name: next }));
      return;
    }
    const msg = await renameMut.mutateAsync({ oldName: renameTarget.name, newName: next });
    if (msg?.success) {
      const affected = (msg.obj as { affected?: number } | undefined)?.affected ?? 0;
      messageApi.success(t('pages.groups.renameSuccess', { count: affected }));
      setRenameOpen(false);
    }
  }

  function onDelete(g: GroupSummary) {
    setConfirm({
      title: t('pages.groups.deleteConfirmTitle', { name: g.name }),
      content: t('pages.groups.deleteConfirmContent', { count: g.clientCount }),
      danger: t('delete'),
      run: async () => {
        const msg = await deleteMut.mutateAsync({ name: g.name });
        if (msg?.success) {
          const affected = (msg.obj as { affected?: number } | undefined)?.affected ?? 0;
          messageApi.success(t('pages.groups.deleteSuccess', { count: affected }));
        }
      },
    });
  }

  async function openSubLinksFor(g: GroupSummary) {
    if (!g.clientCount) {
      messageApi.info(t('pages.groups.emptyForAction'));
      return;
    }
    const emails = await fetchEmailsForGroup(g.name);
    if (emails.length === 0) {
      messageApi.info(t('pages.groups.emptyForAction'));
      return;
    }
    setGroupForAction(g);
    setGroupEmails(emails);
    setSubLinksOpen(true);
  }

  async function openAdjustFor(g: GroupSummary) {
    if (!g.clientCount) {
      messageApi.info(t('pages.groups.emptyForAction'));
      return;
    }
    const emails = await fetchEmailsForGroup(g.name);
    if (emails.length === 0) {
      messageApi.info(t('pages.groups.emptyForAction'));
      return;
    }
    setGroupForAction(g);
    setGroupEmails(emails);
    setAdjustOpen(true);
  }

  function openAddClientsFor(g: GroupSummary) {
    setGroupForAction(g);
    setAddClientsOpen(true);
  }

  function openRemoveClientsFor(g: GroupSummary) {
    if (!g.clientCount) {
      messageApi.info(t('pages.groups.emptyForAction'));
      return;
    }
    setGroupForAction(g);
    setRemoveClientsOpen(true);
  }

  function onDeleteClients(g: GroupSummary) {
    if (!g.clientCount) {
      messageApi.info(t('pages.groups.emptyForAction'));
      return;
    }
    setConfirm({
      title: t('pages.groups.deleteClientsConfirmTitle', { name: g.name }),
      content: t('pages.groups.deleteClientsConfirmContent', { count: g.clientCount }),
      danger: t('delete'),
      run: async () => {
        const emails = await fetchEmailsForGroup(g.name);
        if (emails.length === 0) return;
        const msg = await bulkDelete(emails);
        if (msg?.success) {
          const ok = msg.obj?.deleted ?? 0;
          const skipped = msg.obj?.skipped ?? [];
          const failed = skipped.length;
          if (failed === 0) {
            messageApi.success(t('pages.groups.deleteClientsSuccess', { count: ok }));
          } else {
            const firstError = skipped[0]?.reason ?? msg?.msg ?? '';
            messageApi.warning(
              firstError
                ? `${t('pages.groups.deleteClientsMixed', { ok, failed })} — ${firstError}`
                : t('pages.groups.deleteClientsMixed', { ok, failed }),
            );
          }
        }
      },
    });
  }

  function onResetTraffic(g: GroupSummary) {
    if (!g.clientCount) {
      messageApi.info(t('pages.groups.emptyForAction'));
      return;
    }
    setConfirm({
      title: t('pages.groups.resetConfirmTitle', { name: g.name }),
      content: t('pages.groups.resetConfirmContent'),
      danger: t('reset'),
      run: async () => {
        const msg = await groupResetMut.mutateAsync({ name: g.name });
        if (msg?.success) {
          messageApi.success(t('pages.groups.resetSuccess', { name: g.name }));
        }
      },
    });
  }

  function rowActions(row: GroupSummary): MenuAction[] {
    const none = !row.clientCount;
    return [
      {
        key: 'subLinks',
        label: t('pages.clients.subLinksSelected', { count: row.clientCount || 0 }),
        icon: 'link',
        disabled: none,
        onSelect: () => openSubLinksFor(row),
      },
      {
        key: 'adjust',
        label: t('pages.clients.adjustSelected', { count: row.clientCount || 0 }),
        icon: 'clock',
        disabled: none,
        onSelect: () => openAdjustFor(row),
      },
      {
        key: 'reset',
        label: t('pages.groups.resetTraffic'),
        icon: 'refresh',
        disabled: none,
        onSelect: () => onResetTraffic(row),
      },
      {
        key: 'addClients',
        label: t('pages.groups.addToGroup'),
        icon: 'plus',
        onSelect: () => openAddClientsFor(row),
      },
      {
        key: 'removeClients',
        label: t('pages.groups.removeFromGroup'),
        danger: true,
        disabled: none,
        onSelect: () => openRemoveClientsFor(row),
      },
      {
        key: 'deleteClients',
        label: t('pages.groups.deleteClients'),
        icon: 'trash',
        danger: true,
        disabled: none,
        onSelect: () => onDeleteClients(row),
      },
      {
        key: 'delete',
        label: t('pages.groups.deleteGroupOnly'),
        icon: 'trash',
        danger: true,
        onSelect: () => onDelete(row),
      },
    ];
  }

  return (
    <div className="col stag" style={{ gap: 18 }}>
      {messageContextHolder}
      <div className="pagehead">
        <div>
          <h2 className="h2" style={{ fontSize: 28 }}>
            {t('menu.groups')}
          </h2>
          <p>{t('groups.subtitle')}</p>
        </div>
        <div className="row wrap">
          <Btn kind="pri" icon="plus" onClick={openCreate}>
            {t('groups.new')}
          </Btn>
        </div>
      </div>
      {!fetched ? (
        <div className="grid">
          <section className="tile s6">
            <Sk h={150} />
          </section>
          <section className="tile s6">
            <Sk h={150} />
          </section>
        </div>
      ) : fetchError ? (
        <section className="tile edge st-bad">
          <DlEmpty
            art={<IlloOffline />}
            title={t('somethingWentWrong')}
            actions={
              <Btn
                kind="pri"
                icon="refresh"
                disabled={loading}
                onClick={() => groupsQuery.refetch()}
              >
                {t('refresh')}
              </Btn>
            }
          >
            {fetchError}
          </DlEmpty>
        </section>
      ) : groups.length === 0 ? (
        <section className="tile">
          <DlEmpty
            art={<IlloEmpty size={130} />}
            title={t('groups.empty')}
            actions={
              <Btn kind="pri" icon="plus" onClick={openCreate}>
                {t('groups.new')}
              </Btn>
            }
          >
            {t('groups.emptyHint')}
          </DlEmpty>
        </section>
      ) : (
        <div className="grid">
          {groups.map((g) => (
            <GroupCard
              key={g.name}
              group={g}
              share={totalTraffic ? (g.trafficUsed || 0) / totalTraffic : 0}
              onRename={() => openRename(g)}
              actions={rowActions(g)}
            />
          ))}
        </div>
      )}

      <DlModal
        open={confirm !== null}
        onClose={() => setConfirm(null)}
        label={confirm?.title ?? ''}
        width={460}
      >
        <div className="row" style={{ gap: 12, alignItems: 'flex-start' }}>
          <span className="ico bad">
            <Icon name="trash" />
          </span>
          <div className="f1">
            <h2 className="h3">{confirm?.title}</h2>
            <p className="mut" style={{ margin: '4px 0 0' }}>
              {confirm?.content}
            </p>
          </div>
        </div>
        <div className="mfoot">
          <Btn kind="ghost" onClick={() => setConfirm(null)}>
            {t('cancel')}
          </Btn>
          <Btn
            kind="dng"
            onClick={async () => {
              const run = confirm?.run;
              setConfirm(null);
              await run?.();
            }}
          >
            {confirm?.danger}
          </Btn>
        </div>
      </DlModal>

      <FormModal
        open={createOpen}
        title={t('pages.groups.addGroup')}
        art={
          <span className="ico acc">
            <Icon name="groups" />
          </span>
        }
        width={460}
        onClose={() => setCreateOpen(false)}
        footer={
          <>
            <Btn kind="ghost" onClick={() => setCreateOpen(false)}>
              {t('cancel')}
            </Btn>
            <Btn kind="pri" icon="check" disabled={createMut.isPending} onClick={confirmCreate}>
              {t('create')}
            </Btn>
          </>
        }
      >
        <label className="field">
          <span>{t('pages.groups.name')}</span>
          <input
            id={`${uid}-create-name`}
            className="inp"
            value={createName}
            onChange={(e) => setCreateName(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') confirmCreate();
            }}
            placeholder={t('pages.clients.groupPlaceholder')}
            autoFocus
          />
        </label>
      </FormModal>

      <FormModal
        open={renameOpen}
        title={renameTarget ? t('pages.groups.renameTitle', { name: renameTarget.name }) : ''}
        art={
          <span className="ico acc">
            <Icon name="groups" />
          </span>
        }
        width={460}
        onClose={() => setRenameOpen(false)}
        footer={
          <>
            <Btn kind="ghost" onClick={() => setRenameOpen(false)}>
              {t('cancel')}
            </Btn>
            <Btn kind="pri" icon="check" disabled={renameMut.isPending} onClick={confirmRename}>
              {t('save')}
            </Btn>
          </>
        }
      >
        <label className="field">
          <span>{t('pages.groups.name')}</span>
          <input
            id={`${uid}-rename-name`}
            className="inp"
            value={renameValue}
            onChange={(e) => setRenameValue(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') confirmRename();
            }}
            placeholder={t('pages.clients.groupPlaceholder')}
            autoFocus
          />
        </label>
      </FormModal>

      <LazyMount when={subLinksOpen}>
        <SubLinksModal
          open={subLinksOpen}
          emails={groupEmails}
          clients={allClients}
          subSettings={subSettings}
          onOpenChange={setSubLinksOpen}
        />
      </LazyMount>

      <LazyMount when={adjustOpen}>
        <ClientBulkAdjustModal
          open={adjustOpen}
          count={groupEmails.length}
          onOpenChange={setAdjustOpen}
          onSubmit={async (addDays, addBytes, flow, limitHwid, adTag) => {
            const msg = await bulkAdjust(groupEmails, addDays, addBytes, flow, limitHwid, adTag);
            if (msg?.success) {
              const obj = msg.obj ?? { adjusted: 0 };
              messageApi.success(
                t('pages.groups.adjustSuccess', {
                  count: obj.adjusted ?? 0,
                  name: groupForAction?.name ?? '',
                }),
              );
              return obj;
            }
            return null;
          }}
        />
      </LazyMount>

      <LazyMount when={addClientsOpen}>
        <GroupAddClientsModal
          open={addClientsOpen}
          groupName={groupForAction?.name ?? null}
          candidates={allClients.filter((c) => c.group !== groupForAction?.name)}
          onClose={() => setAddClientsOpen(false)}
          onSubmit={async (emails) => {
            const msg = await bulkAddToGroup(emails, groupForAction?.name ?? '');
            if (msg?.success) {
              return (msg.obj as { affected?: number } | undefined) ?? { affected: 0 };
            }
            return null;
          }}
        />
      </LazyMount>

      <LazyMount when={removeClientsOpen}>
        <GroupRemoveClientsModal
          open={removeClientsOpen}
          groupName={groupForAction?.name ?? null}
          members={allClients.filter((c) => c.group === groupForAction?.name)}
          onClose={() => setRemoveClientsOpen(false)}
          onSubmit={async (emails) => {
            const msg = await bulkRemoveFromGroup(emails);
            if (msg?.success) {
              return (msg.obj as { affected?: number } | undefined) ?? { affected: 0 };
            }
            return null;
          }}
        />
      </LazyMount>
    </div>
  );
}
