# Dalan

Dalan (دالان) is a corridor: a passage through a wall. The panel exists to get people past internet censorship in Iran, and its look says that quietly: dark ink, paper cream, and a saffron light at the end of the passage. **Calm, premium, alive.** Glass and soft light, never neon. Information first; the effects sit behind the data.

Screens, live prototypes and every page: [Dalan 0.6.0 design canvas](https://claude.ai/artifact/2WxRin6u9HWf1KbcfpR73V).

## Voice

- Plain words a non-technical admin understands: "Users", "Servers", "Connections" in Simple mode; "Clients", "Nodes", "Inbounds" in Advanced.
- Say what happened and what to do next. "Paris is not answering. Last heartbeat 6 minutes ago. Settings are safe." Not "Error 504".
- Numbers are exact and live; labels are short. Sentence case everywhere. No exclamation marks, no emoji.
- Every status is an icon plus a word. Colour is never the only signal.
- English is the default; Persian is first class and mirrors the whole layout and motion.

## Colour

Palette C, "Saffron and ink". Dark first; light equally designed. Tokens: `bg`, `surface`, `glass`, `glass-dense`, `text`, `text-muted`, `brand`, `brand-line`, `brand-text`, `on-brand`.

- **Two accent layers.** Saffron (`brand`) is the accent on This server, the logo and the login page. When you open a node, its own accent replaces saffron in the frame only: active tab, primary button, bars, tile edges, sparklines, charts and the background glow tint. Status colours never follow the accent.
- **Every accent has three shades.** Fill (`accent-*`), line (`accent-*-line`, 3:1 for bars and strokes) and text (`accent-*-text`, 4.5:1). In light mode the line and text shades are found by lowering OKLCH lightness at the same hue until the contrast holds against cream, white and glass over the brightest glow, plus a 0.15 margin. A fixed darkening percentage is not enough.
- **On-accent text** is computed (ink or white, whichever is higher). It is ink on all twelve accents.
- **Status:** `success` mint-green, `warning` orange-red (moved away from yellow so it never reads as saffron), `danger` red. Each has a `-text` shade.

| Accent | Fill | On it | Dark text | Light line | Light text | Default for |
|---|---|---|---|---|---|---|
| Saffron | `#FFB020` | ink | `#FFB020` | `#B57A04` | `#906102` | Brand. This server, logo, login. |
| Rose | `#FF3D6E` | ink | `#FE6380` | `#F93669` | `#D3034F` | Turkey and other red flags. |
| Sun | `#FFC83D` | ink | `#FFC83D` | `#A98006` | `#866505` | Germany, gold-led flags. |
| Tulip | `#F062C8` | ink | `#F466CC` | `#DB4EB5` | `#BB2D98` | Netherlands (replaces orange #FF8A3D, too close to warning). |
| Cobalt | `#4C8DFF` | ink | `#619AFF` | `#4283F4` | `#2766D4` | France, blue-led flags. |
| Iris | `#9B8CFF` | ink | `#9D90FE` | `#8675E6` | `#6C59C7` | UK. |
| Lime | `#B6F23C` | ink | `#B6F23C` | `#6A9206` | `#547500` | Free pick; yellow-green, not success mint. |
| Cyan | `#2CCCE4` | ink | `#2CCCE4` | `#0494A6` | `#037685` | Finland, Argentina, sky-blue flags. |
| Amethyst | `#C27CF5` | ink | `#C680F9` | `#AC67DE` | `#904ABF` | Free pick. |
| Ice | `#8EC2FF` | ink | `#8EC2FF` | `#5889C3` | `#3C6DA4` | Light-blue flags (Greece, Israel-free palette, Uruguay). |
| Sand | `#E2BE8A` | ink | `#E2BE8A` | `#A3814F` | `#856533` | Neutral warm, for nodes with no clear flag colour. |
| Steel | `#9AA8BC` | ink | `#9AA8BC` | `#7A879A` | `#5F6C7E` | Neutral cool; also the custom-colour fallback. |

**Netherlands:** the brief's orange `#FF8A3D` sat 6° of hue from warning `#FF7A45`; at a glance an orange tab looked like a warning. It is replaced by **Tulip** `#F062C8` (still Dutch, clearly not a status). Orange stays out of the accent palette, and the custom-colour picker refuses hues within 20° of warning or success.

## Glass, background, effects

- Tiles are glass on every page (`glass`, `glass-border`, `blur-full`, `radius-lg`, `shadow-tile`). Dense screens use `glass-dense`.
- The background is three soft radial glows drifting slowly (38–52 s, alternate), two in the current accent and one in mint. No photos. The glow is capped at `glow-dark` 14% / `glow-light` 16%: stronger glows break text contrast over glass (tested; see Contrast).
- **Effects: Full / Light / Off.** Full: drifting glows, full blur, all motion. Light (auto on weak devices): still glows, 8 px blur, essential motion only. Off: solid `surface` cards, no blur, no ambient motion; forced by the system reduce-motion setting. Ambient motion pauses when the tab is hidden.

## Type

Manrope (Latin) and Vazirmatn (Persian), both variable. Same size scale for both; Persian line heights about 15% taller and no letter spacing. Numbers are tabular everywhere they update live. JetBrains Mono loads only with the Xray editor and logs. Budgets in Hand-off.

## Shape and space

Radii 8 / 12 / 14 / 18 / 24 / pill. Spacing 4 · 8 · 12 · 16 · 18 · 20 · 24 · 28 · 36. A 12-column grid with 18 px gaps on laptop; one column with 14 px gaps on phone. Tiles 18 px radius, phone 20.

## Iconography

A custom 24 px set: 1.75 px stroke, round caps and joins, 2 px inner corners, open shapes. The brand arch appears in the icons that are about passages: Inbounds, Outbounds and Tunnels are arches with arrows. Active icons take `accent-*-line`; otherwise `text` or `text-muted`. Directional icons (chevrons, back) mirror in RTL; everything else does not. Flags are SVG everywhere, round-cropped at 24 px, because Windows does not draw flag emoji. The set and the flags are in Assets.

## Logo

Three nested arches with a saffron doorway and a point of light above. The doorway takes the active accent; the arches take `text`. Minimum 16 px; clear space is the doorway width. The animated version draws the outer arch, then the inner, then the doorway rises with a spring and the light appears (about 1.1 s). As a loader it breathes. The wordmark is "Dalan" in Manrope 800 at -3% tracking, and "دالان" in Vazirmatn 800.

## Data visualisation

Accent lines with round caps, a 6–8 px glow under Full, area fills fading from about 30% to nothing, a muted dashed comparison series, one tooltip chip, no gridline labels inside tiles. Rings for health and subscription days, rounded caps. See the Sparkline and Bar components.

## Sections

Colour contrast, Motion, Hand-off and Pages are separate sections of this system.
