# Modal
Dialogs for create, edit and confirm. Dense glass, 24 px radius, 24 px padding, a blurred scrim.

- Opens from the button that called it: `transform-origin` at that button, scale .97 → 1 and rise 14 px in `d-slow` `e-out`. Closes in 180 ms with `e-in`.
- Title left, close icon button right; actions bottom right (primary last). Destructive confirms type the word to confirm.
- Focus moves into the dialog and returns to the button on close; Esc and the scrim close it.
- Phone: a bottom sheet with a grab handle instead of a centred dialog.