# GlassTile
The base surface of every page: translucent fill, blurred backdrop, a hairline border and an 18 px radius, with an optional 2 px accent edge along the top.

- Fill `glass`, border `glass-border`, radius `radius-lg`, padding `space-6` (phone `space-4`, radius 20), shadow `shadow-tile`, backdrop `blur(blur-full) saturate(1.4)`.
- Edge (`.edge`): a 2 px gradient from `accent-*-line` to transparent across the top, start to end (mirrors in RTL). `.st-warn` / `.st-bad` swap it to `warning` / `danger`.
- Dense (`.dense`): `glass-dense` for tables, logs, editors, modals.
- Effects Off: fill becomes `surface`, no blur. Effects Light: `blur-light`.
- Header row `.th`: title `.tt` (label style, icon in accent line colour) left, one status pill or action right.
- Never put a left border stripe on a tile.
- Ant Design 6: map to `Card` with `token.colorBgContainer = glass`, `borderRadiusLG = 18`, `boxShadowTertiary = shadow-tile`, and the backdrop filter in `components.Card` via a CSS class (`.dl-tile`).