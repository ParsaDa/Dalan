# Motion

A motion system, not decoration: every switch, load and state change moves with intent. Built only from CSS and the Web Animations API. The live demos are the Motion and Loading artboards on the canvas; the hero prototype (Main) uses all of it.

## Tokens

| Token | Value | For |
|---|---|---|
| `d-fast` | 140 ms | press, hover, chip select, pill colour |
| `d-base` | 240 ms | toggle, tab fill, segmented control, fades |
| `d-slow` | 420 ms | tile entrance, route change, modal, sheet, bar fill, sparkline morph |
| `d-sweep` | 640 ms | node switch: accent sweep and accent colour tween |
| `d-count` | 700 ms | number count-up |
| `stagger` | 40 ms | between tiles, at most 12 steps (480 ms) |
| ambient | 38–52 s | glow drift, alternate; paused when hidden |
| `e-out` | cubic-bezier(.22,1,.36,1) | arriving |
| `e-io` | cubic-bezier(.65,0,.35,1) | moving across |
| `e-spring` | cubic-bezier(.34,1.56,.64,1) | small pops |
| `e-in` | cubic-bezier(.55,0,1,.45) | leaving |

Animate `transform` and `opacity` first. Colour tweens use registered custom properties (`@property --acc`, `--acc-line`, `--acc-text` as `<color>`) on the panel root only. Never animate `width`, `height`, `top`, `left` or `margin`. Bars use `scaleX`, rings use `stroke-dashoffset`, sparklines morph `d`.

## Moments

**Node switch** (the signature). t = 0: the tapped tab fills with its accent (`d-base`). 0–640 ms: a soft band of the new accent sweeps across the whole frame from the start edge to the end edge (reversed in RTL) while the root accent tweens, so tabs, buttons, bars, edges and the glow change colour inside the sweep. 0–420 ms + 40 ms per tile: tile content slides 18 px from the end side and fades in. Data for the node is already cached from heartbeats, so nothing waits on the network; if it is not, tiles show skeletons in the same slots. It feels like walking into another room, not reloading.

**Route change.** The old page is removed at once (never wait for an exit); the new page rises 10 px and fades in `d-slow`. The rail's active glow slides to the new item.

**Modal and sheet.** Modal grows from the button that opened it (transform-origin at the button): scale .97 → 1, rise 14 px, `d-slow` `e-out`; closes in 180 ms `e-in`. Phone sheets slide up `d-slow`. The scrim fades `d-base`.

**Loading.** First load: the animated mark (draw, rise, light), then tiles rise in with the stagger. Skeletons match the real layout and shimmer with a moving highlight (a translated pseudo-element, not a background-position loop). Long actions (install an engine, update the panel, restart Xray) show a bar plus a list of real stages from the backend, each with a spinner, then a check; the user can hide it and keep working.

**Live data, every 5 s.** Numbers count from old to new in `d-count` with `e-out` (tabular figures, so nothing jumps). Sparklines drop the oldest point and morph. Bars glide. "Updated Ns ago" ticks each second. A status pill flashes once only when the status word changes. Online → offline: the tile edge, pill and tab dot change colour in `d-base`, the node card shows its offline state; calm, no shaking.

**Controls.** Press scale .97 (`d-fast`). Toggle knob springs; busy toggles breathe until the server answers. Chips fill in `d-base`. Copy and Save swap icon and word with a spring pop ("Copied", "Saved") and return after 1.6–3 s. Errors shake nothing; they show an icon and a sentence under the field.

**Entrance.** Tiles rise 14 px with scale .985, `d-slow`, 40 ms stagger, on first load of a page only, never on refresh.

**Ambient.** Three glows drift slowly behind the glass; they take the node accent tint during a node switch.

**Subscription page and setup wizard.** Lighter: the usage ring fills and the days count up once on load, the QR pops in with a spring, wizard steps slide 24 px in the reading direction, the final step plays the logo animation.

## Effects levels

| Moment | Full | Light | Off / reduce motion |
|---|---|---|---|
| Node switch | sweep 640 + colour tween + slide | sweep 420, glow static | fade 140; colour changes at once |
| Route | rise 420 | rise 240 | fade 140 |
| Numbers | count-up | count-up | instant |
| Sparkline | morph + glow | morph | instant |
| Modal | scale + blur scrim | scale + small blur | fade, solid scrim |
| Ambient glow | drifts | still | hidden |
| Skeleton | shimmer | shimmer | slow opacity pulse |

## Rules

- 60 fps on mid-range phones: at most three blurred layers on screen; under Light the blur drops to 8 px, under Off to none.
- Reduce motion replaces every movement with an instant change or a fade of 120 ms or less.
- RTL mirrors every slide and sweep (`--dir: -1`).
- Never block input on an animation. Every transition can be interrupted: a second tap on another node starts its switch from the current state.
