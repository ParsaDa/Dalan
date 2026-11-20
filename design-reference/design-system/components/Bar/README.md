# Bar
A gradient progress bar for CPU, memory, disk and quota.

- 8 px (12 px `.lg` for quotas), `radius-pill`, `track` behind. Fill runs from a slightly deeper mix to `accent-*-line`, with a soft accent glow under Full effects.
- Animate the fill with `transform: scaleX()` from the start edge (`transform-origin` flips in RTL), `d-slow`, `e-out`. Never animate width.
- At 85% and above the fill turns `warning` and a "High" pill with an icon appears; the value is always printed as a number too.