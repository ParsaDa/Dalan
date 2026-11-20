# Dalan 0.6.0 — release notes

## Breaking

- **Xray tunnels stop working.** The Xray (REALITY) tunnel engine was removed; tunnels run only on BackPack and Rakhsh. On the first start Dalan deletes every Xray tunnel, the management tunnels that rode on them and their places in spare pools. Make a BackPack or Rakhsh tunnel to replace each one.
- **Nodes enrolled over an Xray tunnel must enrol again.** They stay in the node list with a "must be enrolled again" notice. Press **Enrol again** and run the new command on the node; its inbounds and history are kept. The node needs its engine binary first (`x-ui install-engines`).
- Setup links of version 1 (Xray) and enrolment links of version 1 are refused.
- The API drops `security`, `serverName`, `realityPublicKey` and `shortId` from tunnels, and `security`, `serverName` and `withTunnel` from `POST /panel/api/enrolments/add`, which takes `engine`, `transport` and `tunnelId` instead.

## New

- Adding a node with a command asks for the engine (BackPack by default, or Rakhsh) and for a new tunnel for the node or an existing entry tunnel.
- **A new look.** The panel has the Dalan design: dark and light themes, a glass rail, a glow background, a phone tab bar and a logo. An **Effects** setting (Auto, Full, Light, Off) trades visuals for speed; Auto picks Light on weak hardware and Off when the system asks for reduced motion. Twelve accent colours are in Settings → General, and the command palette cycles the effects level.
- **Nodes have a country and an accent colour.** Set them in the node form. The dashboard shows a tab per enabled node (heartbeat data: CPU, memory, latency, uptime) and recolours itself with that node's accent; custom hex accents that look like warning or success colours are refused.
- **A greeting and a nodes status line** open the dashboard, and counters count up when they change.
- **Persian and Arabic mirror the layout** (right-to-left), including the rail and Ant Design components.
- The login and subscription pages use the same look.

## Notes

- Flags cover 16 countries; any other country shows a globe.
- The subscription page still ships the shared Ant Design chunks (about 775 KB gzipped), so its size budget is not met; this was already so before 0.6.0.
- Translations for the new text are written in Persian; the other languages show English until translated.

