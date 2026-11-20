# Table with chips
Lists of clients, inbounds, hosts, logs: a dense glass tile holding a table, with filter chips above.

- Header 12 px bold muted, 10 × 14 px cells, rows 12 px vertical padding, soft row rules (`line-soft`), hover tint 6% accent.
- First and last cells get 20 px inline padding so the table lines up with the tile edge. Wide tables scroll inside the tile.
- Filter chips: 34 px pills with a count; selected chip takes a 16% accent fill and an accent-line border; `aria-pressed`.
- Bulk selection shows a slim action bar above the table.
- Phone: the table becomes a list of rows (name, status pill, usage bar, one action).