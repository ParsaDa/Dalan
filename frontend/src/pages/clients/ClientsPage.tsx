import { lazy, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useLocation, useSearchParams } from 'react-router';
import { useTranslation } from 'react-i18next';
import { Modal, message } from 'antd';

import { formatInboundLabel } from '@/lib/inbounds/label';
import { useMediaQuery } from '@/hooks/useMediaQuery';
import { useWebSocket } from '@/hooks/useWebSocket';
import { useClients } from '@/hooks/useClients';
import { useNodesQuery } from '@/api/queries/useNodesQuery';
import { useHostsQuery } from '@/api/queries/useHostsQuery';
import { useDatepicker } from '@/hooks/useDatepicker';
import type {
  ClientRecord,
  InboundOption,
  ExternalLink,
  ExternalLinkInput,
} from '@/hooks/useClients';
import { IntlUtil } from '@/utils';
import { setMessageInstance } from '@/utils/messageBus';
import { LazyMount } from '@/components/utility';
const ClientFormModal = lazy(() => import('./ClientFormModal'));
const ClientInfoModal = lazy(() => import('./ClientInfoModal'));
const ClientQrModal = lazy(() => import('./ClientQrModal'));
const ClientBulkAddModal = lazy(() => import('./ClientBulkAddModal'));
const ClientBulkAdjustModal = lazy(() => import('./ClientBulkAdjustModal'));
const FilterDrawer = lazy(() => import('./FilterDrawer'));
const SubLinksModal = lazy(() => import('./SubLinksModal'));
const BulkAddToGroupModal = lazy(() => import('./BulkAddToGroupModal'));
const BulkAttachInboundsModal = lazy(() => import('./BulkAttachInboundsModal'));
const BulkDetachInboundsModal = lazy(() => import('./BulkDetachInboundsModal'));
const TextModal = lazy(() => import('@/components/feedback/TextModal'));
const PromptModal = lazy(() => import('@/components/feedback/PromptModal'));
import { emptyFilters, activeFilterCount } from './filters';
import type { ClientFilters } from './filters';
import Icon from '@/components/dl/Icon';
import { ConfirmDelete } from '@/components/dl/overlays';
import { DropChip } from '@/components/dl/DropChip';
import { IlloEmpty, IlloOffline } from '@/components/dl/Illustrations';
import { MoreMenu } from '@/components/dl/MoreMenu';
import { Btn, Chip, Empty as DlEmpty, IconBtn, Sk } from '@/components/dl/primitives';
import { fmtInt } from '@/components/dl/numbers';
import ClientTable from './ClientTable';

const FILTER_STATE_KEY = 'clientsFilterState';
const DISABLED_PAGE_SIZE = 200;
const DEFAULT_TABLE_PAGE_SIZE = 25;

// The server sends exact counters but caps the email arrays behind them, so a
// panel with thousands of depleted clients neither ships nor renders them all.
// The trailing chip reports what the popover left out.

type Bucket = 'active' | 'deactive' | 'depleted' | 'expiring';

interface PersistedFilterState {
  searchKey: string;
  filters: ClientFilters;
  sort: string;
  // The page size resolved on the previous visit. Without it the first list
  // request has to wait for /setting/defaultSettings just to learn how many rows
  // to ask for, which serialises two round trips on every load.
  pageSize: number | null;
}

function readFilterState(): PersistedFilterState {
  try {
    const raw = JSON.parse(localStorage.getItem(FILTER_STATE_KEY) || '{}');
    const fromRaw = (raw.filters ?? {}) as Partial<ClientFilters>;
    return {
      searchKey: typeof raw.searchKey === 'string' ? raw.searchKey : '',
      filters: {
        ...emptyFilters(),
        ...fromRaw,
        buckets: Array.isArray(fromRaw.buckets) ? fromRaw.buckets : [],
        protocols: Array.isArray(fromRaw.protocols) ? fromRaw.protocols : [],
        inboundIds: Array.isArray(fromRaw.inboundIds) ? fromRaw.inboundIds : [],
        nodeIds: Array.isArray(fromRaw.nodeIds) ? fromRaw.nodeIds : [],
        groups: Array.isArray(fromRaw.groups) ? fromRaw.groups : [],
      },
      sort: typeof raw.sort === 'string' ? raw.sort : '',
      pageSize: typeof raw.pageSize === 'number' && raw.pageSize > 0 ? raw.pageSize : null,
    };
  } catch {
    return { searchKey: '', filters: emptyFilters(), sort: '', pageSize: null };
  }
}

function gbToBytes(gb: number | undefined): number {
  if (!gb || gb <= 0) return 0;
  return Math.round(gb * 1024 * 1024 * 1024);
}

const SORT_OPTIONS: {
  value: string;
  column: string;
  order: 'ascend' | 'descend';
  labelKey: string;
}[] = [
  {
    value: 'createdAt:ascend',
    column: 'createdAt',
    order: 'ascend',
    labelKey: 'pages.clients.sortOldest',
  },
  {
    value: 'createdAt:descend',
    column: 'createdAt',
    order: 'descend',
    labelKey: 'pages.clients.sortNewest',
  },
  {
    value: 'updatedAt:descend',
    column: 'updatedAt',
    order: 'descend',
    labelKey: 'pages.clients.sortRecentlyUpdated',
  },
  {
    value: 'lastOnline:descend',
    column: 'lastOnline',
    order: 'descend',
    labelKey: 'pages.clients.sortRecentlyOnline',
  },
  {
    value: 'email:ascend',
    column: 'email',
    order: 'ascend',
    labelKey: 'pages.clients.sortEmailAZ',
  },
  {
    value: 'email:descend',
    column: 'email',
    order: 'descend',
    labelKey: 'pages.clients.sortEmailZA',
  },
  {
    value: 'traffic:descend',
    column: 'traffic',
    order: 'descend',
    labelKey: 'pages.clients.sortMostTraffic',
  },
  {
    value: 'remaining:descend',
    column: 'remaining',
    order: 'descend',
    labelKey: 'pages.clients.sortHighestRemaining',
  },
  {
    value: 'expiryTime:ascend',
    column: 'expiryTime',
    order: 'ascend',
    labelKey: 'pages.clients.sortExpiringSoonest',
  },
];

const DEFAULT_SORT = SORT_OPTIONS[0];

function sortValueFor(column: string | null, order: 'ascend' | 'descend' | null): string {
  if (!column || !order) return DEFAULT_SORT.value;
  return `${column}:${order}`;
}

export default function ClientsPage() {
  const { t } = useTranslation();
  const { datepicker } = useDatepicker();
  const { isMobile } = useMediaQuery();
  const [modal, modalContextHolder] = Modal.useModal();
  const [bulkDel, setBulkDel] = useState<string[] | null>(null);
  const [messageApi, messageContextHolder] = message.useMessage();
  useEffect(() => {
    setMessageInstance(messageApi);
  }, [messageApi]);

  const {
    clients,
    total,
    filtered,
    summary,
    allGroups,
    setQuery,
    inbounds,
    onlines,
    transitioning,
    fetched,
    fetchError,
    subSettings,
    tgBotEnable,
    expireDiff,
    trafficDiff,
    pageSize,
    settingsReady,
    create,
    update,
    remove,
    bulkDelete,
    bulkAdjust,
    bulkEnable,
    bulkDisable,
    bulkAddToGroup,
    bulkRemoveFromGroup,
    attach,
    setExternalLinks,
    bulkAttach,
    detach,
    bulkDetach,
    resetTraffic,
    resetAllTraffics,
    delDepleted,
    delOrphans,
    exportClients,
    importClients,
    setEnable,
    clientSpeed,
    applyTrafficEvent,
    applyClientStatsEvent,
    refresh,
    hydrate,
  } = useClients();

  useWebSocket({
    traffic: applyTrafficEvent,
    client_stats: applyClientStatsEvent,
  });

  // Node list for the Nodes filter; the section only renders when the panel
  // actually manages nodes (#4997).
  const { nodes } = useNodesQuery();
  // Tunnel configs advertise these Hosts, so an empty list must mean "no hosts"
  // and not "not loaded yet" — the page gate waits for it.
  const {
    hosts,
    fetched: hostsFetched,
    fetchError: hostsFetchError,
    refetch: refetchHosts,
  } = useHostsQuery();
  const hostsError = hosts.length > 0 ? '' : hostsFetchError;

  const [togglingEmail, setTogglingEmail] = useState<string | null>(null);
  const [formOpen, setFormOpen] = useState(false);
  const [formMode, setFormMode] = useState<'add' | 'edit'>('add');
  const [editingClient, setEditingClient] = useState<ClientRecord | null>(null);
  const [editingAttachedIds, setEditingAttachedIds] = useState<number[]>([]);
  const [editingExternalLinks, setEditingExternalLinks] = useState<ExternalLink[]>([]);
  const [editingTunnelAllowedIPs, setEditingTunnelAllowedIPs] = useState<Record<number, string>>(
    {},
  );
  const [infoOpen, setInfoOpen] = useState(false);
  const [infoClient, setInfoClient] = useState<ClientRecord | null>(null);
  const [qrOpen, setQrOpen] = useState(false);
  const [qrClient, setQrClient] = useState<ClientRecord | null>(null);
  const [viewingTunnelAllowedIPs, setViewingTunnelAllowedIPs] = useState<Record<number, string>>(
    {},
  );
  const [bulkAddOpen, setBulkAddOpen] = useState(false);
  const [bulkAdjustOpen, setBulkAdjustOpen] = useState(false);
  const [subLinksOpen, setSubLinksOpen] = useState(false);
  const [bulkGroupOpen, setBulkGroupOpen] = useState(false);
  const [bulkAttachOpen, setBulkAttachOpen] = useState(false);
  const [bulkDetachOpen, setBulkDetachOpen] = useState(false);
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([]);

  const [textOpen, setTextOpen] = useState(false);
  const [textTitle, setTextTitle] = useState('');
  const [textContent, setTextContent] = useState('');
  const [textFileName, setTextFileName] = useState('');
  const [promptOpen, setPromptOpen] = useState(false);
  const [promptTitle, setPromptTitle] = useState('');
  const [promptOkText, setPromptOkText] = useState('');
  const [promptInitial, setPromptInitial] = useState('');
  const [promptLoading, setPromptLoading] = useState(false);
  const [promptHandler, setPromptHandler] = useState<
    ((value: string) => Promise<boolean | void> | boolean | void) | null
  >(null);

  const initial = readFilterState();
  const location = useLocation();
  const [searchParams] = useSearchParams();
  const searchParam = searchParams.get('search');
  const [searchKey, setSearchKey] = useState(
    searchParam !== null ? searchParam : initial.searchKey,
  );
  const [filters, setFilters] = useState<ClientFilters>(initial.filters);
  const [filterDrawerOpen, setFilterDrawerOpen] = useState(false);

  const initialSort = SORT_OPTIONS.find((o) => o.value === initial.sort) ?? DEFAULT_SORT;
  const [sortColumn, setSortColumn] = useState<string | null>(initialSort.column);
  const [sortOrder, setSortOrder] = useState<'ascend' | 'descend' | null>(initialSort.order);
  // Derived, not mirrored into state by an effect: an effect lags one render
  // behind the settings arriving, and that lag is what made the page fetch the
  // list once with the placeholder size and again with the real one.
  const [pageSizeChoice, setPageSizeChoice] = useState<number | null>(null);
  const settingsPageSize = settingsReady ? (pageSize > 0 ? pageSize : DISABLED_PAGE_SIZE) : null;
  // Last visit's resolved size stands in until the settings land, so the list
  // request goes out with the page mount instead of queueing behind them. If the
  // admin has since changed the setting the authoritative value replaces it and
  // costs one refetch — only on the load that follows the change. Null means
  // nothing is known yet, which is the one case worth waiting for.
  const resolvedPageSize = pageSizeChoice ?? settingsPageSize ?? initial.pageSize;
  const tablePageSize = resolvedPageSize ?? DEFAULT_TABLE_PAGE_SIZE;
  // debouncedSearch lags behind the input so we don't spam the server on every
  // keystroke; the search box still feels instant locally.
  const [debouncedSearch, setDebouncedSearch] = useState(searchKey);
  const [prevLocationKey, setPrevLocationKey] = useState(location.key);

  if (location.key !== prevLocationKey) {
    setPrevLocationKey(location.key);
    if (searchParam !== null) {
      setSearchKey(searchParam);
      setDebouncedSearch(searchParam);
    }
  }

  useEffect(() => {
    localStorage.setItem(
      FILTER_STATE_KEY,
      JSON.stringify({
        searchKey,
        filters,
        sort: sortValueFor(sortColumn, sortOrder),
        // Only ever persist a size we actually resolved, never the render fallback.
        pageSize: resolvedPageSize,
      }),
    );
  }, [searchKey, filters, sortColumn, sortOrder, resolvedPageSize]);

  useEffect(() => {
    const handle = window.setTimeout(() => setDebouncedSearch(searchKey), 300);
    return () => window.clearTimeout(handle);
  }, [searchKey]);

  // The page belongs to one query: a new search, filter or sort starts again at page 1.
  const querySig = JSON.stringify([debouncedSearch, filters, sortColumn, sortOrder]);
  const [paging, setPaging] = useState({ sig: querySig, page: 1 });
  const currentPage = paging.sig === querySig ? paging.page : 1;
  const setCurrentPage = (page: number) => setPaging({ sig: querySig, page });

  // The node filter maps onto inbound ids client-side (#4997): the paging API
  // already accepts an inbound CSV, so nodes never have to reach the backend.
  // Sentinel 0 = "local panel" (inbounds without a nodeId).
  const effectiveInboundCsv = useMemo(() => {
    if (!filters.nodeIds.length) return filters.inboundIds.join(',');
    const nodeSet = new Set(filters.nodeIds);
    const nodeInboundIds = inbounds.filter((ib) => nodeSet.has(ib.nodeId ?? 0)).map((ib) => ib.id);
    const pool = filters.inboundIds.length
      ? nodeInboundIds.filter((id) => filters.inboundIds.includes(id))
      : nodeInboundIds;
    // Nothing matches the selected nodes: send an impossible id so the filter
    // yields an honest empty result instead of being silently ignored.
    return pool.length ? pool.join(',') : '-1';
  }, [filters.nodeIds, filters.inboundIds, inbounds]);

  useEffect(() => {
    // With no remembered size and no settings yet, any query we build would be a
    // guess, and issuing it costs a full server round trip that is thrown away as
    // soon as the real size arrives.
    if (resolvedPageSize === null) return;
    setQuery({
      page: currentPage,
      pageSize: tablePageSize,
      search: debouncedSearch,
      filter: filters.buckets.join(','),
      protocol: filters.protocols.join(','),
      inbound: effectiveInboundCsv,
      expiryFrom: filters.expiryFrom,
      expiryTo: filters.expiryTo,
      usageFrom: gbToBytes(filters.usageFromGB),
      usageTo: gbToBytes(filters.usageToGB),
      autoRenew: filters.autoRenew || undefined,
      hasTgId: filters.hasTgId || undefined,
      hasComment: filters.hasComment || undefined,
      group: filters.groups.join(',') || undefined,
      sort: sortColumn || undefined,
      order: sortOrder || undefined,
    });
  }, [
    setQuery,
    resolvedPageSize,
    currentPage,
    tablePageSize,
    debouncedSearch,
    filters,
    effectiveInboundCsv,
    sortColumn,
    sortOrder,
  ]);

  const activeCount = activeFilterCount(filters);

  // Row handlers take an email and look the row up here at call time. Keying
  // them on the record object instead would defeat the memoised cells: every
  // traffic push replaces the row object of every client whose counters moved,
  // so the memo would miss on exactly the rows that are busy. Reading through
  // the ref also means a modal opened mid-poll shows current usage.
  const rowsByEmail = useRef(new Map<string, ClientRecord>());
  const rowMap = useMemo(() => {
    const map = new Map<string, ClientRecord>();
    for (const c of clients) map.set(c.email, c);
    return map;
  }, [clients]);
  useEffect(() => {
    rowsByEmail.current = rowMap;
  }, [rowMap]);

  const onlineSet = useMemo(() => new Set(onlines || []), [onlines]);
  const inboundsById = useMemo(() => {
    const out: Record<number, InboundOption> = {};
    for (const ib of inbounds) out[ib.id] = ib;
    return out;
  }, [inbounds]);

  const protocolOptions = useMemo(() => {
    const values = new Set<string>(
      (inbounds || []).map((i) => i.protocol).filter((x): x is string => !!x),
    );
    return [...values].sort();
  }, [inbounds]);

  const groupOptions = useMemo(() => {
    const values = new Set<string>(allGroups);
    for (const g of filters.groups) values.add(g);
    return [...values].sort((a, b) => a.localeCompare(b));
  }, [allGroups, filters.groups]);

  const isOnline = useCallback((email: string) => !!email && onlineSet.has(email), [onlineSet]);

  function inboundLabel(id: number) {
    const ib = inboundsById[id];
    return formatInboundLabel(ib?.tag, ib?.remark);
  }

  const clientBucket = useCallback(
    (row: ClientRecord | null | undefined): Bucket | null => {
      if (!row) return null;
      const traffic = row.traffic || {};
      const used = (traffic.up || 0) + (traffic.down || 0);
      const total = row.totalGB || 0;
      const now = Date.now();
      const expired = (row.expiryTime ?? 0) > 0 && (row.expiryTime ?? 0) <= now;
      const exhausted = total > 0 && used >= total;
      if (expired || exhausted) return 'depleted';
      if (!row.enable) return 'deactive';
      const nearExpiry =
        (row.expiryTime ?? 0) > 0 && (row.expiryTime ?? 0) - now < (expireDiff || 0);
      const nearLimit = total > 0 && total - used < (trafficDiff || 0);
      if (nearExpiry || nearLimit) return 'expiring';
      return 'active';
    },
    [expireDiff, trafficDiff],
  );

  // The list page renders rows the server already sorted, filtered, and
  // paginated. Local filtering is gone — keep the variable name so the rest
  // of the file (table dataSource, mobile cards, select-all) doesn't need
  // a rename.
  const filteredClients = clients;

  // Sort is server-side now; the page already arrives in the requested
  // order, so we just hand it through.
  const sortedClients = filteredClients;

  async function onToggleEnable(row: ClientRecord, next: boolean) {
    setTogglingEmail(row.email);
    try {
      const msg = await setEnable(row, next);
      if (!msg?.success) {
        messageApi.error(msg?.msg || t('somethingWentWrong'));
      }
    } finally {
      setTogglingEmail(null);
    }
  }

  function onAdd() {
    setFormMode('add');
    setEditingClient(null);
    setEditingAttachedIds([]);
    setEditingExternalLinks([]);
    setEditingTunnelAllowedIPs({});
    setFormOpen(true);
  }

  const onEdit = useCallback(
    async (email: string) => {
      const row = rowsByEmail.current.get(email);
      if (!row) return;
      setFormMode('edit');
      // Paged list omits per-client secrets to keep the row payload tiny;
      // edit needs them, so fetch the full record first.
      const full = await hydrate(row.email);
      const merged: ClientRecord = full ? { ...row, ...full.client } : { ...row };
      setEditingClient(merged);
      const ids = full?.inboundIds ?? (Array.isArray(row.inboundIds) ? row.inboundIds : []);
      setEditingAttachedIds([...ids]);
      setEditingExternalLinks(Array.isArray(full?.externalLinks) ? [...full.externalLinks] : []);
      setEditingTunnelAllowedIPs(full?.tunnelAllowedIPs ?? {});
      setFormOpen(true);
    },
    [hydrate],
  );

  const onDelete = useCallback(
    (email: string) => {
      const row = rowsByEmail.current.get(email);
      if (!row) return;
      modal.confirm({
        title: t('pages.clients.deleteConfirmTitle', { email: row.email }),
        content: t('pages.clients.deleteConfirmContent'),
        okText: t('delete'),
        okType: 'danger',
        cancelText: t('cancel'),
        onOk: async () => {
          const msg = await remove(row.email);
          if (msg?.success) messageApi.success(t('pages.clients.toasts.deleted'));
        },
      });
    },
    [modal, t, remove, messageApi],
  );

  const onResetTraffic = useCallback(
    (email: string) => {
      const row = rowsByEmail.current.get(email);
      if (!row?.email) {
        messageApi.warning(t('pages.clients.resetNotPossible'));
        return;
      }
      modal.confirm({
        title: `${t('pages.inbounds.resetTraffic')} — ${row.email}`,
        content: t('pages.inbounds.resetTrafficContent'),
        okText: t('reset'),
        cancelText: t('cancel'),
        onOk: async () => {
          const msg = await resetTraffic(row);
          if (msg?.success) messageApi.success(t('pages.clients.toasts.trafficReset'));
        },
      });
    },
    [modal, t, resetTraffic, messageApi],
  );

  const onShowInfo = useCallback(
    async (email: string) => {
      const row = rowsByEmail.current.get(email);
      if (!row) return;
      const full = await hydrate(row.email);
      setInfoClient(full ? { ...row, ...full.client, inboundIds: full.inboundIds } : row);
      setViewingTunnelAllowedIPs(full?.tunnelAllowedIPs ?? {});
      setInfoOpen(true);
    },
    [hydrate],
  );

  const onShowQr = useCallback(
    async (email: string) => {
      const row = rowsByEmail.current.get(email);
      if (!row) return;
      const full = await hydrate(row.email);
      setQrClient(full ? { ...row, ...full.client, inboundIds: full.inboundIds } : row);
      setViewingTunnelAllowedIPs(full?.tunnelAllowedIPs ?? {});
      setQrOpen(true);
    },
    [hydrate],
  );

  const [refreshing, setRefreshing] = useState(false);
  const onRefreshClick = useCallback(async () => {
    setRefreshing(true);
    try {
      await Promise.all([refresh(), refetchHosts()]);
    } finally {
      setRefreshing(false);
    }
  }, [refresh, refetchHosts]);

  const openText = useCallback((opts: { title: string; content: string; fileName?: string }) => {
    setTextTitle(opts.title);
    setTextContent(opts.content);
    setTextFileName(opts.fileName || '');
    setTextOpen(true);
  }, []);

  const openPrompt = useCallback(
    (opts: {
      title: string;
      okText?: string;
      value?: string;
      confirm: (value: string) => Promise<boolean | void> | boolean | void;
    }) => {
      setPromptTitle(opts.title);
      setPromptOkText(opts.okText || t('confirm'));
      setPromptInitial(opts.value || '');
      setPromptHandler(() => opts.confirm);
      setPromptOpen(true);
    },
    [t],
  );

  const onPromptConfirm = useCallback(
    async (value: string) => {
      if (!promptHandler) {
        setPromptOpen(false);
        return;
      }
      setPromptLoading(true);
      try {
        const ok = await promptHandler(value);
        if (ok !== false) setPromptOpen(false);
      } finally {
        setPromptLoading(false);
      }
    },
    [promptHandler],
  );

  function onResetAllTraffics() {
    modal.confirm({
      title: t('pages.clients.resetAllTrafficsTitle'),
      content: t('pages.clients.resetAllTrafficsContent'),
      okText: t('reset'),
      okType: 'danger',
      cancelText: t('cancel'),
      onOk: async () => {
        const msg = await resetAllTraffics();
        if (msg?.success) messageApi.success(t('pages.clients.toasts.allTrafficsReset'));
      },
    });
  }

  function onDelDepleted() {
    modal.confirm({
      title: t('pages.clients.delDepletedConfirmTitle'),
      content: t('pages.clients.delDepletedConfirmContent'),
      okText: t('delete'),
      okType: 'danger',
      cancelText: t('cancel'),
      onOk: async () => {
        const msg = await delDepleted();
        if (msg?.success) {
          const deleted = msg.obj?.deleted ?? 0;
          messageApi.success(t('pages.clients.toasts.delDepleted', { count: deleted }));
        }
      },
    });
  }

  function onDeleteOrphans() {
    modal.confirm({
      title: t('pages.clients.delOrphansConfirmTitle'),
      content: t('pages.clients.delOrphansConfirmContent'),
      okText: t('delete'),
      okType: 'danger',
      cancelText: t('cancel'),
      onOk: async () => {
        const msg = await delOrphans();
        if (msg?.success) {
          const deleted = msg.obj?.deleted ?? 0;
          messageApi.success(t('pages.clients.toasts.delOrphans', { count: deleted }));
        }
      },
    });
  }

  async function onExportClients() {
    const items = await exportClients();
    if (!items) return;
    openText({
      title: t('pages.clients.exportClients'),
      content: JSON.stringify(items, null, 2),
      fileName: 'clients-export.json',
    });
  }

  function onImportClients() {
    openPrompt({
      title: t('pages.clients.importClients'),
      okText: t('pages.clients.import'),
      value: '',
      confirm: async (value) => {
        const msg = await importClients(value);
        if (!msg?.success) return false;
        const created = msg.obj?.created ?? 0;
        const skipped = msg.obj?.skipped ?? [];
        if (skipped.length === 0) {
          messageApi.success(t('pages.clients.toasts.imported', { count: created }));
        } else {
          const firstError = skipped[0]?.reason ?? '';
          messageApi.warning(
            firstError
              ? `${t('pages.clients.toasts.importedMixed', { ok: created, failed: skipped.length })} — ${firstError}`
              : t('pages.clients.toasts.importedMixed', { ok: created, failed: skipped.length }),
          );
        }
        return true;
      },
    });
  }

  function onBulkUngroup() {
    const emails = [...selectedRowKeys];
    if (emails.length === 0) return;
    modal.confirm({
      title: t('pages.clients.ungroupConfirmTitle', { count: emails.length }),
      content: t('pages.clients.ungroupConfirmContent'),
      okText: t('confirm'),
      okType: 'danger',
      cancelText: t('cancel'),
      onOk: async () => {
        const msg = await bulkRemoveFromGroup(emails);
        if (msg?.success) {
          setSelectedRowKeys([]);
          const affected =
            (msg.obj as { affected?: number } | undefined)?.affected ?? emails.length;
          messageApi.success(t('pages.clients.ungroupSuccessToast', { count: affected }));
        }
      },
    });
  }

  function onBulkSetEnable(enable: boolean) {
    const emails = [...selectedRowKeys];
    if (emails.length === 0) return;
    modal.confirm({
      title: t(
        enable ? 'pages.clients.bulkEnableConfirmTitle' : 'pages.clients.bulkDisableConfirmTitle',
        { count: emails.length },
      ),
      content: t(
        enable
          ? 'pages.clients.bulkEnableConfirmContent'
          : 'pages.clients.bulkDisableConfirmContent',
      ),
      okText: t('confirm'),
      okType: enable ? 'primary' : 'danger',
      cancelText: t('cancel'),
      onOk: async () => {
        const msg = enable ? await bulkEnable(emails) : await bulkDisable(emails);
        setSelectedRowKeys([]);
        const changed = msg?.obj?.changed ?? 0;
        const skipped = msg?.obj?.skipped ?? [];
        const failed = skipped.length;
        const firstError = skipped[0]?.reason ?? msg?.msg ?? '';
        const okKey = enable
          ? 'pages.clients.toasts.bulkEnabled'
          : 'pages.clients.toasts.bulkDisabled';
        const mixedKey = enable
          ? 'pages.clients.toasts.bulkEnabledMixed'
          : 'pages.clients.toasts.bulkDisabledMixed';
        if (failed === 0 && msg?.success) {
          messageApi.success(t(okKey, { count: changed }));
        } else {
          messageApi.warning(
            firstError
              ? `${t(mixedKey, { ok: changed, failed })} — ${firstError}`
              : t(mixedKey, { ok: changed, failed }),
          );
        }
      },
    });
  }

  function onBulkDelete() {
    if (selectedRowKeys.length > 0) setBulkDel([...selectedRowKeys]);
  }

  async function runBulkDelete(emails: string[]) {
    const msg = await bulkDelete(emails);
    setBulkDel(null);
    setSelectedRowKeys([]);
    const ok = msg?.obj?.deleted ?? 0;
    const skipped = msg?.obj?.skipped ?? [];
    const failed = skipped.length;
    const firstError = skipped[0]?.reason ?? msg?.msg ?? '';
    if (failed === 0 && msg?.success) {
      messageApi.success(t('pages.clients.toasts.bulkDeleted', { count: ok }));
    } else {
      messageApi.warning(
        firstError
          ? `${t('pages.clients.toasts.bulkDeletedMixed', { ok, failed })} — ${firstError}`
          : t('pages.clients.toasts.bulkDeletedMixed', { ok, failed }),
      );
    }
  }

  const onSave = useCallback(
    async (
      payload: Record<string, unknown> | { client: Record<string, unknown>; inboundIds: number[] },
      meta:
        | { isEdit: false; email: string; externalLinks: ExternalLinkInput[] }
        | {
            isEdit: true;
            email: string;
            attach: number[];
            detach: number[];
            externalLinks: ExternalLinkInput[];
          },
    ) => {
      if (!meta.isEdit) {
        const createMsg = await create(payload);
        if (!createMsg?.success) return createMsg;
        if (meta.email && meta.externalLinks.length > 0) {
          const r = await setExternalLinks(meta.email, meta.externalLinks);
          if (!r?.success) return r;
        }
        return createMsg;
      }
      const updateMsg = await update(meta.email, payload);
      if (!updateMsg?.success) return updateMsg;
      const rawEmail = (payload as { email?: unknown }).email;
      const emailKey =
        typeof rawEmail === 'string' && rawEmail.trim() ? rawEmail.trim() : meta.email;
      if (Array.isArray(meta.attach) && meta.attach.length > 0) {
        const r = await attach(emailKey, meta.attach);
        if (!r?.success) return r;
      }
      if (Array.isArray(meta.detach) && meta.detach.length > 0) {
        const r = await detach(emailKey, meta.detach);
        if (!r?.success) return r;
      }
      // Always replace the client's external links (an empty set clears them).
      const r = await setExternalLinks(emailKey, meta.externalLinks);
      if (!r?.success) return r;
      return updateMsg;
    },
    [create, update, attach, detach, setExternalLinks],
  );

  function toggleSelect(email: string, checked: boolean) {
    setSelectedRowKeys((prev) => {
      const next = new Set(prev);
      if (checked) next.add(email);
      else next.delete(email);
      return Array.from(next);
    });
  }

  function selectAll(checked: boolean) {
    setSelectedRowKeys(checked ? filteredClients.map((c) => c.email) : []);
  }

  const isOnlyBucket = (bucket: string) =>
    filters.buckets.length === 1 && filters.buckets[0] === bucket;

  // Clicking the card that is already the sole status filter clears it again.
  function selectBucket(bucket: string | null) {
    const buckets = bucket && !isOnlyBucket(bucket) ? [bucket] : [];
    setFilters({ ...filters, buckets });
  }

  function clearOneFilter<K extends keyof ClientFilters>(key: K) {
    if (key === 'expiryFrom' || key === 'expiryTo') {
      setFilters({ ...filters, expiryFrom: undefined, expiryTo: undefined });
      return;
    }
    if (key === 'usageFromGB' || key === 'usageToGB') {
      setFilters({ ...filters, usageFromGB: undefined, usageToGB: undefined });
      return;
    }
    setFilters({ ...filters, [key]: emptyFilters()[key] });
  }
  // Every active filter as a removable chip, in the order the drawer lists them.
  function activeFilterChips(): { key: string; label: string; clear: () => void }[] {
    const without =
      <K extends keyof ClientFilters>(key: K, value: ClientFilters[K]) =>
      () =>
        setFilters({ ...filters, [key]: value });
    const fmt = (v?: number) => (v ? IntlUtil.formatDate(v, datepicker) : '…');
    return [
      ...filters.buckets.map((b) => ({
        key: `b-${b}`,
        label: bucketChipLabel(b, t),
        clear: without(
          'buckets',
          filters.buckets.filter((x) => x !== b),
        ),
      })),
      ...filters.protocols.map((p) => ({
        key: `p-${p}`,
        label: p,
        clear: without(
          'protocols',
          filters.protocols.filter((x) => x !== p),
        ),
      })),
      ...filters.inboundIds.map((id) => ({
        key: `i-${id}`,
        label: inboundLabel(id),
        clear: without(
          'inboundIds',
          filters.inboundIds.filter((x) => x !== id),
        ),
      })),
      ...filters.nodeIds.map((id) => ({
        key: `n-${id}`,
        label:
          id === 0
            ? t('pages.clients.filters.localPanel')
            : nodes.find((n) => n.id === id)?.name || `#${id}`,
        clear: without(
          'nodeIds',
          filters.nodeIds.filter((x) => x !== id),
        ),
      })),
      ...filters.groups.map((g) => ({
        key: `g-${g}`,
        label: `${t('pages.clients.group')}: ${g}`,
        clear: without(
          'groups',
          filters.groups.filter((x) => x !== g),
        ),
      })),
      ...(filters.expiryFrom || filters.expiryTo
        ? [
            {
              key: 'exp',
              label: `${t('pages.clients.expiryTime')}: ${fmt(filters.expiryFrom)} → ${fmt(filters.expiryTo)}`,
              clear: () => clearOneFilter('expiryFrom'),
            },
          ]
        : []),
      ...(filters.usageFromGB || filters.usageToGB
        ? [
            {
              key: 'use',
              label: `${t('pages.clients.traffic')}: ${filters.usageFromGB ?? 0}${filters.usageToGB ? `–${filters.usageToGB}` : '+'} GB`,
              clear: () => clearOneFilter('usageFromGB'),
            },
          ]
        : []),
      ...(filters.autoRenew
        ? [
            {
              key: 'renew',
              label: `${t('pages.clients.renew')}: ${filters.autoRenew === 'on' ? t('enabled') : t('disabled')}`,
              clear: () => clearOneFilter('autoRenew'),
            },
          ]
        : []),
      ...(filters.hasTgId
        ? [
            {
              key: 'tg',
              label: `${t('pages.clients.telegramId')}: ${filters.hasTgId === 'yes' ? t('pages.clients.has') : t('pages.clients.hasNot')}`,
              clear: () => clearOneFilter('hasTgId'),
            },
          ]
        : []),
      ...(filters.hasComment
        ? [
            {
              key: 'cm',
              label: `${t('pages.clients.comment')}: ${filters.hasComment === 'yes' ? t('pages.clients.has') : t('pages.clients.hasNot')}`,
              clear: () => clearOneFilter('hasComment'),
            },
          ]
        : []),
    ];
  }

  const pages = Math.max(1, Math.ceil(filtered / tablePageSize));

  return (
    <div className="col stag" style={{ gap: 18 }}>
      {messageContextHolder}
      {modalContextHolder}
      <ConfirmDelete
        open={bulkDel !== null}
        title={t('pages.clients.bulkDeleteConfirmTitle', { count: bulkDel?.length ?? 0 })}
        onCancel={() => setBulkDel(null)}
        onConfirm={() => runBulkDelete(bulkDel ?? [])}
      >
        {bulkDel && (
          <>
            {t('clientsPage.loseAccess', {
              names:
                bulkDel.length > 3
                  ? t('clientsPage.andMore', {
                      names: bulkDel.slice(0, 3).join(', '),
                      count: bulkDel.length - 3,
                    })
                  : bulkDel.join(', '),
            })}{' '}
            {t('pages.clients.bulkDeleteConfirmContent')}
          </>
        )}
      </ConfirmDelete>
      <div className="pagehead">
        <div>
          <h2 className="h2" style={{ fontSize: 28 }}>
            {t('menu.clients')}
          </h2>
          <p>{t('clientsPage.subtitle')}</p>
        </div>
        <div className="row wrap">
          <MoreMenu
            label={t('more')}
            actions={[
              {
                key: 'bulk',
                label: t('pages.clients.bulk'),
                icon: 'plus',
                onSelect: () => setBulkAddOpen(true),
              },
              { key: 'import', label: t('pages.clients.importClients'), onSelect: onImportClients },
              {
                key: 'resetAll',
                label: t('pages.clients.resetAllTraffics'),
                icon: 'refresh',
                onSelect: onResetAllTraffics,
              },
              {
                key: 'delDepleted',
                label: t('pages.clients.delDepleted'),
                icon: 'trash',
                danger: true,
                onSelect: onDelDepleted,
              },
              {
                key: 'delOrphans',
                label: t('pages.clients.delOrphans'),
                icon: 'trash',
                danger: true,
                onSelect: onDeleteOrphans,
              },
            ]}
          />
          <Btn icon="down" onClick={onExportClients}>
            {t('clientsPage.export')}
          </Btn>
          <Btn kind="pri" icon="plus" onClick={onAdd}>
            {t('clientsPage.add')}
          </Btn>
        </div>
      </div>
      {!fetched || !hostsFetched ? (
        <section className="tile">
          <Sk h={220} />
        </section>
      ) : fetchError || hostsError ? (
        <section className="tile edge st-bad">
          <DlEmpty
            art={<IlloOffline />}
            title={t('somethingWentWrong')}
            actions={
              <Btn kind="pri" icon="refresh" disabled={refreshing} onClick={onRefreshClick}>
                {t('refresh')}
              </Btn>
            }
          >
            {fetchError || hostsError}
          </DlEmpty>
        </section>
      ) : (
        <>
          <div className="row wrap sp">
            <div className="chips" role="group" aria-label={t('clientsPage.filters')}>
              <Chip
                on={filters.buckets.length === 0}
                count={fmtInt(summary.total)}
                onClick={() => selectBucket(null)}
              >
                {t('clientsPage.chip.all')}
              </Chip>
              <Chip
                on={isOnlyBucket('active')}
                count={fmtInt(summary.active)}
                onClick={() => selectBucket('active')}
              >
                {t('clientsPage.chip.active')}
              </Chip>
              <Chip
                on={isOnlyBucket('expiring')}
                count={fmtInt(summary.expiringCount)}
                onClick={() => selectBucket('expiring')}
              >
                {t('clientsPage.chip.expiring')}
              </Chip>
              <Chip
                on={isOnlyBucket('depleted')}
                count={fmtInt(summary.depletedCount)}
                onClick={() => selectBucket('depleted')}
              >
                {t('clientsPage.chip.depleted')}
              </Chip>
              <Chip
                on={isOnlyBucket('deactive')}
                count={fmtInt(summary.deactiveCount)}
                onClick={() => selectBucket('deactive')}
              >
                {t('clientsPage.chip.deactive')}
              </Chip>
              <Chip
                on={isOnlyBucket('online')}
                count={fmtInt(summary.onlineCount)}
                onClick={() => selectBucket('online')}
              >
                {t('clientsPage.chip.online')}
              </Chip>
            </div>
            <div className="row wrap">
              <DropChip
                icon="groups"
                label={t('pages.clients.group')}
                value={filters.groups.length === 1 ? filters.groups[0] : ''}
                onChange={(v) => setFilters({ ...filters, groups: v ? [v] : [] })}
                options={[
                  { value: '', label: t('clientsPage.any') },
                  ...groupOptions.map((g) => ({ value: g, label: g })),
                ]}
              />
              <DropChip
                icon="inbound"
                label={t('menu.inbounds')}
                value={filters.inboundIds.length === 1 ? String(filters.inboundIds[0]) : ''}
                onChange={(v) => setFilters({ ...filters, inboundIds: v ? [Number(v)] : [] })}
                options={[
                  { value: '', label: t('clientsPage.any') },
                  ...inbounds.map((ib) => ({ value: String(ib.id), label: inboundLabel(ib.id) })),
                ]}
              />
              <DropChip
                label={t('sort')}
                value={sortValueFor(sortColumn, sortOrder)}
                onChange={(value) => {
                  const opt = SORT_OPTIONS.find((o) => o.value === value);
                  setSortColumn(opt?.column ?? null);
                  setSortOrder(opt?.order ?? null);
                }}
                options={SORT_OPTIONS.map((o) => ({ value: o.value, label: t(o.labelKey) }))}
              />
              <button
                type="button"
                className={`chip${activeCount > 0 ? ' on' : ''}`}
                onClick={() => setFilterDrawerOpen(true)}
              >
                <Icon name="filter" />
                {t('filter')}
                {activeCount > 0 && <span className="n">{fmtInt(activeCount)}</span>}
              </button>
              <label className="search" style={{ minWidth: 240, cursor: 'text' }}>
                <Icon name="search" />
                <input
                  aria-label={t('search')}
                  placeholder={t('clientsPage.searchHint')}
                  value={searchKey}
                  onChange={(e) => setSearchKey(e.target.value)}
                  style={{
                    border: 0,
                    background: 'none',
                    outline: 'none',
                    color: 'var(--tx)',
                    font: 'inherit',
                    flex: 1,
                    minWidth: 0,
                  }}
                />
              </label>
            </div>
          </div>
          {activeCount > 0 && (
            <div className="row wrap" style={{ gap: 6 }}>
              {activeFilterChips().map((c) => (
                <span
                  key={c.key}
                  className="badge acc"
                  style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}
                >
                  {c.label}
                  <button
                    type="button"
                    className="ib sm"
                    style={{ width: 18, height: 18 }}
                    aria-label={`${t('clientsPage.removeFilter')} ${c.label}`}
                    onClick={c.clear}
                  >
                    <Icon name="x" size={12} />
                  </button>
                </span>
              ))}
              <Btn size="sm" kind="ghost" onClick={() => setFilters(emptyFilters())}>
                {t('pages.clients.clearAllFilters')}
              </Btn>
              <span className="mut" style={{ fontSize: 12.5, fontWeight: 650 }}>
                {t('pages.clients.showingCount', { shown: filtered, total })}
              </span>
            </div>
          )}
          {selectedRowKeys.length > 0 && (
            <div className="tile row wrap" style={{ padding: '10px 14px', gap: 12 }}>
              <span className="badge acc">
                {t('pages.clients.selectedCount', { count: selectedRowKeys.length })}
              </span>
              <span
                className="f1 mut"
                style={{
                  fontWeight: 650,
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                }}
              >
                {selectedRowKeys.slice(0, 4).join(', ')}
                {selectedRowKeys.length > 4 ? '…' : ''}
              </span>
              <Btn size="sm" icon="refresh" onClick={() => setBulkAdjustOpen(true)}>
                {t('pages.clients.adjust')}
              </Btn>
              <Btn size="sm" icon="groups" onClick={() => setBulkGroupOpen(true)}>
                {t('pages.clients.addToGroup')}
              </Btn>
              <Btn size="sm" icon="power" onClick={() => onBulkSetEnable(false)}>
                {t('pages.clients.disable')}
              </Btn>
              <MoreMenu
                label={t('more')}
                actions={[
                  {
                    key: 'enable',
                    label: t('pages.clients.enable'),
                    onSelect: () => onBulkSetEnable(true),
                  },
                  {
                    key: 'subLinks',
                    label: t('pages.clients.subLinks'),
                    icon: 'link',
                    onSelect: () => setSubLinksOpen(true),
                  },
                  {
                    key: 'attach',
                    label: t('pages.clients.attach'),
                    onSelect: () => setBulkAttachOpen(true),
                  },
                  {
                    key: 'detach',
                    label: t('pages.clients.detach'),
                    danger: true,
                    onSelect: () => setBulkDetachOpen(true),
                  },
                  {
                    key: 'ungroup',
                    label: t('pages.clients.ungroup'),
                    danger: true,
                    onSelect: onBulkUngroup,
                  },
                ]}
              />
              <Btn size="sm" kind="dng" icon="trash" onClick={onBulkDelete}>
                {t('delete')}
              </Btn>
              <IconBtn small icon="x" label={t('cancel')} onClick={() => setSelectedRowKeys([])} />
            </div>
          )}
          {total === 0 ? (
            <section className="tile">
              <DlEmpty
                art={<IlloEmpty size={130} />}
                title={t('clientsPage.empty')}
                actions={
                  <Btn kind="pri" icon="plus" onClick={onAdd}>
                    {t('clientsPage.add')}
                  </Btn>
                }
              >
                {t('clientsPage.emptyHint')}
              </DlEmpty>
            </section>
          ) : (
            <section
              className="tile dense"
              style={{ padding: 14, opacity: transitioning ? 0.6 : 1 }}
              aria-busy={transitioning}
            >
              <ClientTable
                rows={sortedClients}
                narrow={isMobile}
                selected={selectedRowKeys}
                inboundsById={inboundsById}
                clientSpeed={clientSpeed}
                togglingEmail={togglingEmail}
                bucketOf={clientBucket}
                isOnline={isOnline}
                onSelect={toggleSelect}
                onSelectAll={selectAll}
                onEdit={onEdit}
                onShowQr={onShowQr}
                onShowInfo={onShowInfo}
                onResetTraffic={onResetTraffic}
                onDelete={onDelete}
                onToggle={(row, on) => void onToggleEnable(row, on)}
                onGroup={(g) => setFilters({ ...filters, groups: [g] })}
              />
              {filtered === 0 && (
                <div className="mut" style={{ padding: 14, textAlign: 'center' }}>
                  {t('noData')}
                </div>
              )}
            </section>
          )}
          {filtered > 0 && (
            <div className="row sp wrap mut" style={{ fontWeight: 650, fontSize: 13, gap: 10 }}>
              <span>
                {t('clientsPage.showing', {
                  from: fmtInt((currentPage - 1) * tablePageSize + 1),
                  to: fmtInt(Math.min(currentPage * tablePageSize, filtered)),
                  total: fmtInt(filtered),
                })}
              </span>
              <div className="row" style={{ gap: 6 }}>
                <DropChip
                  label={t('clientsPage.perPage')}
                  value={String(tablePageSize)}
                  onChange={(v) => {
                    setPageSizeChoice(Number(v));
                    setCurrentPage(1);
                  }}
                  options={['10', '25', '50', '100', '200'].map((n) => ({ value: n, label: n }))}
                />
                <IconBtn
                  small
                  icon="back"
                  className="flipx"
                  label={t('inbounds.prev')}
                  disabled={currentPage <= 1}
                  onClick={() => setCurrentPage(currentPage - 1)}
                />
                <span className="badge acc num">{`${fmtInt(currentPage)} / ${fmtInt(pages)}`}</span>
                <IconBtn
                  small
                  icon="chev"
                  className="flipx"
                  label={t('inbounds.next')}
                  disabled={currentPage >= pages}
                  onClick={() => setCurrentPage(currentPage + 1)}
                />
              </div>
            </div>
          )}
        </>
      )}

      <LazyMount when={formOpen}>
        <ClientFormModal
          open={formOpen}
          mode={formMode}
          client={editingClient}
          attachedIds={editingAttachedIds}
          attachedExternalLinks={editingExternalLinks}
          tunnelAllowedIPs={editingTunnelAllowedIPs}
          inbounds={inbounds}
          tgBotEnable={tgBotEnable}
          groups={allGroups}
          save={onSave}
          resetTraffic={resetTraffic}
          onOpenChange={setFormOpen}
        />
      </LazyMount>
      <LazyMount when={infoOpen}>
        <ClientInfoModal
          open={infoOpen}
          client={infoClient}
          inboundsById={inboundsById}
          tunnelAllowedIPs={viewingTunnelAllowedIPs}
          isOnline={infoClient ? isOnline(infoClient.email) : false}
          subSettings={subSettings}
          hosts={hosts}
          onOpenChange={setInfoOpen}
        />
      </LazyMount>
      <LazyMount when={qrOpen}>
        <ClientQrModal
          open={qrOpen}
          client={qrClient}
          inboundsById={inboundsById}
          tunnelAllowedIPs={viewingTunnelAllowedIPs}
          subSettings={subSettings}
          hosts={hosts}
          onOpenChange={setQrOpen}
        />
      </LazyMount>
      <LazyMount when={bulkAddOpen}>
        <ClientBulkAddModal
          open={bulkAddOpen}
          inbounds={inbounds}
          groups={allGroups}
          onOpenChange={setBulkAddOpen}
          onSaved={() => setBulkAddOpen(false)}
        />
      </LazyMount>
      <LazyMount when={bulkAdjustOpen}>
        <ClientBulkAdjustModal
          open={bulkAdjustOpen}
          count={selectedRowKeys.length}
          onOpenChange={setBulkAdjustOpen}
          onSubmit={async (addDays, addBytes, flow, limitHwid, adTag) => {
            const msg = await bulkAdjust(
              [...selectedRowKeys],
              addDays,
              addBytes,
              flow,
              limitHwid,
              adTag,
            );
            if (msg?.success) {
              setSelectedRowKeys([]);
              return msg.obj ?? { adjusted: 0 };
            }
            return null;
          }}
        />
      </LazyMount>
      <LazyMount when={subLinksOpen}>
        <SubLinksModal
          open={subLinksOpen}
          emails={selectedRowKeys}
          clients={clients}
          subSettings={subSettings}
          onOpenChange={setSubLinksOpen}
        />
      </LazyMount>
      <LazyMount when={bulkGroupOpen}>
        <BulkAddToGroupModal
          open={bulkGroupOpen}
          count={selectedRowKeys.length}
          groups={allGroups}
          onOpenChange={setBulkGroupOpen}
          onSubmit={async (group) => {
            const msg = await bulkAddToGroup([...selectedRowKeys], group);
            if (msg?.success) {
              setSelectedRowKeys([]);
              return (msg.obj as { affected?: number } | undefined) ?? { affected: 0 };
            }
            return null;
          }}
        />
      </LazyMount>
      <LazyMount when={bulkAttachOpen}>
        <BulkAttachInboundsModal
          open={bulkAttachOpen}
          count={selectedRowKeys.length}
          inbounds={inbounds}
          onOpenChange={setBulkAttachOpen}
          onSubmit={async (inboundIds) => {
            const msg = await bulkAttach([...selectedRowKeys], inboundIds);
            if (msg?.success) {
              setSelectedRowKeys([]);
              return msg.obj ?? { attached: [], skipped: [], errors: [] };
            }
            return null;
          }}
        />
      </LazyMount>
      <LazyMount when={bulkDetachOpen}>
        <BulkDetachInboundsModal
          open={bulkDetachOpen}
          count={selectedRowKeys.length}
          inbounds={inbounds}
          onOpenChange={setBulkDetachOpen}
          onSubmit={async (inboundIds) => {
            const msg = await bulkDetach([...selectedRowKeys], inboundIds);
            if (msg?.success) {
              setSelectedRowKeys([]);
              return msg.obj ?? { detached: [], skipped: [], errors: [] };
            }
            return null;
          }}
        />
      </LazyMount>
      <LazyMount when={filterDrawerOpen}>
        <FilterDrawer
          open={filterDrawerOpen}
          onOpenChange={setFilterDrawerOpen}
          filters={filters}
          onChange={setFilters}
          inbounds={inbounds}
          protocols={protocolOptions}
          groups={groupOptions}
          nodes={nodes}
        />
      </LazyMount>
      <LazyMount when={textOpen}>
        <TextModal
          open={textOpen}
          onClose={() => setTextOpen(false)}
          title={textTitle}
          content={textContent}
          fileName={textFileName}
          json
        />
      </LazyMount>
      <LazyMount when={promptOpen}>
        <PromptModal
          open={promptOpen}
          onClose={() => setPromptOpen(false)}
          title={promptTitle}
          okText={promptOkText}
          initialValue={promptInitial}
          loading={promptLoading}
          json
          onConfirm={onPromptConfirm}
        />
      </LazyMount>
    </div>
  );
}

function bucketChipLabel(b: string, t: (k: string) => string): string {
  switch (b) {
    case 'active':
      return t('subscription.active');
    case 'expiring':
      return t('depletingSoon');
    case 'depleted':
      return t('depleted');
    case 'deactive':
      return t('disabled');
    case 'online':
      return t('online');
    default:
      return b;
  }
}
