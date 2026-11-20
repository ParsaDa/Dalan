# StatusPill
A rounded label for a state: icon plus word, tinted by status. Never colour alone.

- 24 px tall, 12 px bold, 13% tint of its own colour as the fill.
- `ok` → `success-text`, `warn` → `warning-text`, `bad` → `danger-text`, `off` → `text-muted`. These never follow the node accent.
- `acc` is for non-status labels (Live, Recommended) and does follow the accent.
- On a status change the pill flashes once (`.flash`, 900 ms ring). It does not pulse on every refresh.
- Badge: a square-ish 20 px tag for protocol names and counts.