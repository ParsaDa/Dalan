# Dalan 0.6.2 — release notes

Fixes from running 0.6.1 on three servers.

## Fixed

- **First-run wizard is one short step.** It asks only for the server name, the country and the IP or domain, every field is optional, and Skip works. It no longer creates an inbound, so it cannot fail with "something went wrong" or "port already exists" and loop. The address you type is the one clients, links and subscriptions use when no other public host is set.
- **Menus and dropdowns are no longer cut off.** The "…" menu of the last row in Inbounds and Clients, and dropdowns inside dialogs (such as "Download on"), open over the table or dialog and flip upward near the screen edge.
- **The accent colour is kept.** Settings → Appearance saves it in the panel, so it survives a reload and follows you to another browser.

## Changed

- **Engine install shows the last four releases.** Opening an engine's options (BackPack, Rakhsh) looks up its newest four releases through the chosen route, or through a node, and installs or updates to the one you pick. When the list cannot be read, the dialog says this server cannot reach GitHub and suggests a tunnel, another route or a link. Installing without a pick still uses the default release.
- **Dashboard node tabs recolour the whole frame.** Choosing an external node switches the accent of the entire page to that node's colour.
- **Outbounds holds outbounds only.** Routing rules live in Routing. Subscription outbounds and NordVPN stay, and Tor and PIA now sit inside Outbounds. Old links to the moved pages still work.
- **Multi-admin is gone.** Extra admin accounts are removed from Settings and from the interface.

## Notes

- No migration needs action. The accent and the server address are stored as panel settings and take their default on first start.
- Tor installs still download its pinned bundle; the release picker covers BackPack and Rakhsh.
- Translations for the new text are written in Persian; the other languages show English until translated.
