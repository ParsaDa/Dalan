# Sparkline
A small trend line inside a tile: 2.25 px stroke, round caps and joins, area fading from 28% accent to nothing, a dot at the latest point.

- Stroke `accent-*-line`; glow (`drop-shadow` 6 px of the accent) only under Full.
- Live update: drop the oldest point, append the new one, and morph the path `d` in `d-slow`. A dashed muted line is the comparison period.
- 24 points for 2 minutes at the 5 s heartbeat. No axes inside tiles.
- The charts library is not needed for sparklines: they are plain SVG paths built in the component.