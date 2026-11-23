# Dalan 0.6.1 — release notes

## New

- **Every page now follows the 0.6.0 design.** The first look of 0.6.0 only carried the colours and the glass; 0.6.1 rebuilds the panel frame (grouped rail with Simple and Advanced modes, header with search and Ctrl K, node tabs, phone tab bar) and each page from the design: Dashboard, Activity, Nodes and the node form, Tunnels, Outbounds and Routing, Hosts, Groups, Inbounds, Clients, Logs, Tor and PIA, Settings, Xray config, Developer, Login, the subscription page, and the not-found and error pages.
- **Dashboard** shows server health, the state of every tunnel, online users with throughput and monthly traffic, alerts, recent activity and one card per connected node.
- **Alerts** are stored: a node going offline, high CPU or memory on a node, an Xray crash, and users who expire this week. An alert closes by itself when the cause is gone.
- **Activity log** records who changed what (clients, tunnels, nodes, Xray restarts, settings). It has its own page and a filter for who made the change.
- **Monthly traffic against a plan.** Settings has a monthly cap and a reset day; the dashboard meter appears when a cap is set.
- **First-run wizard** (`/setup`) names the server, picks its country and creates a first inbound (Reality on 443, or Trojan WS on 8080 for a CDN). It appears only on a panel with no inbound that has not finished it, so existing panels never see it.
- **Settings** is one page with sections for general, appearance, subscription, security, backups, notifications, developer and updates. Backups and the update dialogs moved here. Old links to settings tabs still work.
- Messages use the design toast, and deleting many clients asks you to type DELETE (or حذف in Persian).
- The panel shows Persian digits when the language is Persian, and every page has a phone layout and a right-to-left layout.

## Notes

- No migration needs action; the new tables (alerts, activity, traffic periods) are created on the first start.
- Some parts of the design have no data behind them yet and are left out or simplified: reachability from Iran on hosts, plan text and inbound badges on groups, the 24-hour chart on inbounds, a BackPack, Rakhsh and Tor log source, the count of users routed through Tor, a "match system" theme and the standby server in backups. Iran has no drawn flag, so a server set to Iran shows the globe.
- Subscription outbounds, the basic routing editor and form fields inside dialogs still use Ant Design, restyled to the theme.
- Translations for the new text are written in Persian; the other languages show English until translated.
