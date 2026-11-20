# Rail
The left navigation: grouped sections, a glowing active item, collapsible to an icon rail; becomes a bottom bar on phones.

- 236 px open, 76 px collapsed (labels and group titles hide, tooltips show the name). Glass fill with blur, hairline on the inline end.
- Groups (Advanced): Overview, Users and access, Network, System. Simple mode shows seven plain items: Home, Users, Connections, Servers, Tunnels, Activity, Settings.
- Active item: accent-tinted fill at 14%, icon in `accent-*-line`, a 3 px accent bar with glow at the inline start. `aria-current="page"`.
- The "Users and access" group title is kept for the future roles area.
- Phone: bottom bar with Home, Users, a raised accent Add button, Tunnels, More. Everything else lives under More.