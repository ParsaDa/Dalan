# Toggle
An on/off switch for things that run: Xray, tunnels, inbounds, engines. A real `button` with `role="switch"` and `aria-checked`.

- 46 × 28 px; off: 1.5 px `control-border` outline and a `text-muted` knob; on: accent fill and an `on-brand` knob.
- Knob travels with `e-spring` in `d-base`; it stretches 4 px while pressed. Busy (`.busy`): knob breathes until the server answers, then settles.
- Turning off something users depend on (Xray, a tunnel) asks once in a small popover unless the panel is in Advanced mode with confirmations off.