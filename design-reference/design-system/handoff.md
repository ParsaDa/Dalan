# Hand-off for engineers

Stack stays: React 19, Ant Design 6 with a deep custom theme and CSS, Vite, TypeScript. `components/bundle.css` in this system is the reference stylesheet the artboards use; port its classes into the theme and a small `dalan.css`.

## Ant Design 6 theme

- `ConfigProvider theme`: `algorithm: darkAlgorithm | defaultAlgorithm` by theme; `token.colorPrimary` = the current accent fill; `colorBgBase` = `bg`; `colorBgContainer` = `glass`; `colorBgElevated` = `glass-dense`; `colorText` = `text`; `colorTextSecondary` = `text-muted`; `colorBorder` = `control-border`; `colorBorderSecondary` = `glass-border`; `colorSuccess/Warning/Error` = status tokens; `borderRadius` 12, `borderRadiusLG` 18, `borderRadiusSM` 8; `fontFamily` = `sans` (Persian: `fa`); `controlHeight` 40, `controlHeightLG` 52; `motionDurationFast/Mid/Slow` = 140/240/420 ms; `motionEaseOut` = `e-out`.
- Component tokens: `Button.primaryColor` = `on-brand`; `Switch` handle colours as in Toggle; `Table.headerBg` transparent, `rowHoverBg` = 6% accent; `Tabs` are not used for node tabs (custom NodeTab); `Modal.contentBg` = `glass-dense`; `Segmented` for Effects and Simple/Advanced.
- Turn on `cssVar` mode so the accent can change by swapping one class on the root without re-rendering the tree. Set `token.colorPrimary` from a CSS variable that the root class drives.

## Plain CSS vs JS

| Piece | How |
|---|---|
| Glass, glow background, edges, pills, bars, toggles, chips, skeletons, entrance stagger, route fade, sweep keyframes | Plain CSS (bundle.css) |
| Accent colour tween | CSS: `@property` registered colours on the root + transition. One class change per switch |
| Node switch orchestration | JS: set root class, flip an `a`/`b` key on the content wrapper so keyframes restart, start sweep. ~40 lines |
| Count-up numbers | JS: one `requestAnimationFrame` loop per visible tile group, 700 ms; skip under Off and reduce motion |
| Sparkline morph | CSS `transition: d` on the path; JS builds the path string |
| Modal from button | JS: read the button rect, set `transform-origin` before mount |
| Pause ambient when hidden | JS: `visibilitychange` toggles a `paused` class on the root; also stop the 5 s refresh |
| Effects auto Light | JS: `navigator.hardwareConcurrency <= 4` or `deviceMemory <= 4` or a dropped-frame probe in the first 2 s → Light, saved per device |

## Cut first if performance is poor

1. Glow drift (keep static glows).
2. Sparkline glow `drop-shadow`.
3. Backdrop blur on table rows and rail (keep on tiles), then blur 24 → 8 px.
4. Accent colour tween (snap colours, keep the sweep).
5. Count-up on more than 12 numbers at once (count only what is on screen).

## Size budget

- No animation library. Motion is CSS + Web Animations API.
- Fonts, panel: Manrope variable Latin subset (~38 KB) + Vazirmatn variable Arabic subset (~48 KB), `font-display: swap`. Vazirmatn loads only when the locale is Persian or Persian text appears (use `unicode-range`).
- Fonts, login and subscription page: Manrope 600 and 800 static (~24 KB) + Vazirmatn 500 and 800 static (~30 KB) only when Persian. No icon font: icons are inline SVG from one sprite (~9 KB).
- On demand: API reference (Settings › Developer), code editor (Xray config), charts library (only full-size charts on Activity and node traffic pages), JetBrains Mono, the QR encoder on the subscription page (or render the QR on the server).
- Subscription page target: under 60 KB of JS and CSS gzip, first paint on 3G under 2 s. It shares tokens and bundle.css rules but not the panel bundle.
- Flags: one SVG sprite of the flags in use (~6 KB), not a flag library.

## Per page

Dashboard and node tabs: data from node heartbeats already in memory; switching a node never waits on the network. Clients, logs: virtualised lists over 200 rows. Xray config: editor chunk loaded on route enter, with a skeleton. Tunnels: render engines from the backend list so a new engine needs no UI change.

## Accessibility

Real buttons and links, `aria-pressed` on tabs and chips, `role="switch"` on toggles, `aria-live="polite"` on the status line and progress stages, focus ring 2 px `accent-*-line`, 44 px touch targets on phones, every status as icon + word.
