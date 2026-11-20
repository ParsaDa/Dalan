# NodeTab
A pill per directly connected node under the header; picking one switches the whole panel into that node.

- 40 px tall (phone chip 36 px), `radius-pill`, 24 px round SVG flag, name, latency in 12 px, a 7 px status dot with an `aria-label` of the status word.
- Active: fill is that node's accent (`accent-*`), text `on-brand`, soft accent shadow. The root accent class changes to the node, so every accent token follows.
- "This server" uses the server icon and saffron. Nodes default to the accent of their flag; the admin can override it in Edit node.
- Only direct nodes get tabs. Chained nodes appear inside their parent's dashboard.
- Phone: one scrollable row with the current node always visible, then a `+N` chip that opens a sheet with search.
- The group is `role="toolbar"`, each tab a `button` with `aria-pressed`.
- Motion: see the Motion section, "Node switch".