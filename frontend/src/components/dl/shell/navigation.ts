import type { IconName } from '../Icon';

export interface NavItem {
  path: string;
  icon: IconName;
  labelKey: string;
  simpleLabelKey?: string;
  count?: 'clients';
}

export interface NavGroup {
  titleKey: string;
  items: NavItem[];
}

const DASHBOARD: NavItem = {
  path: '/',
  icon: 'dash',
  labelKey: 'menu.dashboard',
  simpleLabelKey: 'shell.nav.home',
};
const ACTIVITY: NavItem = { path: '/activity', icon: 'activity', labelKey: 'menu.activity' };
const INBOUNDS: NavItem = {
  path: '/inbounds',
  icon: 'inbound',
  labelKey: 'menu.inbounds',
  simpleLabelKey: 'shell.nav.connections',
};
const CLIENTS: NavItem = {
  path: '/clients',
  icon: 'users',
  labelKey: 'menu.clients',
  simpleLabelKey: 'shell.nav.users',
  count: 'clients',
};
const GROUPS: NavItem = { path: '/groups', icon: 'groups', labelKey: 'menu.groups' };
const HOSTS: NavItem = { path: '/hosts', icon: 'hosts', labelKey: 'menu.hosts' };
const TUNNELS: NavItem = { path: '/tunnels', icon: 'tunnel', labelKey: 'menu.tunnels' };
const OUTBOUNDS: NavItem = { path: '/outbound', icon: 'outbound', labelKey: 'menu.outbounds' };
const ROUTING: NavItem = { path: '/routing', icon: 'routing', labelKey: 'menu.routing' };
const NODES: NavItem = {
  path: '/nodes',
  icon: 'nodes',
  labelKey: 'menu.nodes',
  simpleLabelKey: 'shell.nav.servers',
};
const TOR: NavItem = { path: '/tor', icon: 'tor', labelKey: 'menu.torPia' };
const XRAY: NavItem = { path: '/xray', icon: 'xray', labelKey: 'menu.xray' };
const LOGS: NavItem = { path: '/logs', icon: 'logs', labelKey: 'menu.logs' };
const SETTINGS: NavItem = { path: '/settings', icon: 'settings', labelKey: 'menu.settings' };

export const ADVANCED_NAV: NavGroup[] = [
  { titleKey: 'shell.nav.overview', items: [DASHBOARD, ACTIVITY] },
  { titleKey: 'shell.nav.usersAccess', items: [INBOUNDS, CLIENTS, GROUPS, HOSTS] },
  { titleKey: 'shell.nav.network', items: [TUNNELS, OUTBOUNDS, ROUTING, NODES, TOR] },
  { titleKey: 'shell.nav.system', items: [XRAY, LOGS, SETTINGS] },
];

// Simple mode: plain names, raw Xray options hidden; everything stays reachable from search.
export const SIMPLE_NAV: NavGroup[] = [
  { titleKey: '', items: [DASHBOARD, CLIENTS, INBOUNDS, NODES, TUNNELS] },
  { titleKey: '', items: [ACTIVITY, SETTINGS] },
];

export function navFor(simple: boolean, routed: ReadonlySet<string>): NavGroup[] {
  return (simple ? SIMPLE_NAV : ADVANCED_NAV)
    .map((g) => ({ ...g, items: g.items.filter((i) => routed.has(i.path)) }))
    .filter((g) => g.items.length > 0);
}

export function isActivePath(itemPath: string, pathname: string): boolean {
  if (itemPath === '/') return pathname === '/' || pathname === '';
  return pathname === itemPath || pathname.startsWith(`${itemPath}/`);
}
