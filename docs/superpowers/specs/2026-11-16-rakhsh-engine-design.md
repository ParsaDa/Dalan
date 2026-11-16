# Rakhsh as the third tunnel engine (Dalan 0.5.0)

## Intent

Parsa wants Dalan to run three engines side by side in the Tunnels page: Xray VLESS reverse, BackPack, and
**Rakhsh** ([CluvexStudio/rakhsh](https://github.com/CluvexStudio/rakhsh), AGPL-3.0, Rust, one static binary,
v0.1.0). Rakhsh brings what the other two lack: several carriers on one tunnel with automatic failover, `resume`
(streams survive a dead lane), datagram carriers with FEC, layer 3, per-forward limits and quotas. He wants **all
its options**, in the same panel, with the same create / edit / label / outbound behaviour as the other engines.
A fourth engine, and a tunnel of Dalan's own, may follow later.

Why: during shutdowns the carrier that works changes by the hour. One engine is one bet; three are a hedge.

## Goals, non-goals, anti-goals

Goals: create, edit, delete, status, traffic and Xray-outbound routing for Rakhsh tunnels exactly like BackPack's;
every tunnel-shaping option of Rakhsh reachable from the form; node-created tunnels (one panel makes both ends).

Non-goals: Rakhsh's standalone router (`[[inbound]]`, `[[outbound]]`, `[route]`, `[dns]`, `[openvpn]`, x-ui sync),
the control-center binary, loss-aware switching (Rakhsh does not do it; Parsa deferred it), Windows/macOS.

Anti-goal: a second place to write routing rules. Xray stays the router; a Rakhsh tunnel is just a pipe that Xray
reaches through a loopback relay port, the way BackPack tunnels already work.

## Approaches

1. **Engine beside BackPack, driven by a catalog** (chosen). New package `internal/tunnel/rakhsh` mirroring
   `internal/tunnel/backpack`: `RenderConfig`, `Supervisor`, `Install`, `Catalog`, status reader. Same
   `ServerTunnel` model, same option form. Costs one more package; reuses every proven pattern.
2. **Generic "external engine" abstraction first, then port BackPack and Rakhsh onto it.** Cleaner on paper, but it
   rewrites a working, released engine and touches every call site. A refactor, not a feature; rejected for 0.5.0.
3. **A free-text TOML box.** Cheapest, and `rakhsh check` would validate it, but it gives no form, no hints, no
   mirrored ends and no safe edit. Rejected.

## Design

**Roles and direction.** Dalan portal = Rakhsh `entry` (users connect here, holds the forwards); bridge = `exit`.
Reverse mode: `dial = "exit"` (the bridge dials the portal) — Rakhsh's default. Direct mode: `dial = "entry"`.
Layer 3 is an option (`l3_addr`, `/30` taken from the same 10.10.N pool BackPack direct uses), valid in either
mode, so the Reverse/Direct label keeps meaning "who dials".

**Keys.** The Noise handshake pins both static keys. The portal generates both X25519 pairs, keeps its own secret
and the bridge's public key, and the setup link (version 4) carries the bridge's secret, the portal's public key
and the token. This is exactly what Rakhsh's own `--join rakhsh://` link does (it carries the other end's whole
config); the link is a secret, as the UI already says of every setup link. For a node-created tunnel the link never
leaves the management channel. New columns: `rakhsh_secret` (never serialised) and `rakhsh_peer_public`.

**Config rendering.** Typed fields stay in the model (carrier = `Transport`, listen/connect = `Address:Port`,
forwards = `ForwardPorts`, relay = `RelayPort`); everything else is `Options`, validated against the catalog,
written at top level, in `[[forward]]`, or as `[[link]]` blocks (the structured `links` option: carrier, addr,
and the per-link overrides Rakhsh allows). `rakhsh check` is run by the e2e test against rendered configs.

**Catalog groups** (key, kind, role, mode, hint in English and Persian, like BackPack's): identity (`encryption`,
`resume`, `resume_grace`), tunnel (`lanes`, `max_connections`, `preset`, `heartbeat`, `keepalive`, socket buffers,
`bind`), carrier (`sni`, `ws_path`, `ws_credential`, `decoy`, `cert_path`, `key_path`, `acme_*`, `tls_fingerprint`,
`tls_frag`, `edge_ip`, `quic_mask`, `quic_version`, `congestion`), links (`links`, `link_policy`, `link_timeout`,
`link_probe`, `backup_connect`), pacing (`pace_*`), raw (`peer_ip`, `reliability`, `faketcp_header`,
`raw_window_mb`, `raw_congestion`, `raw_lanes`, `fec_data`, `fec_parity`, `link_encrypt`, `spoof_*`), layer 3
(`l3_*`, `mss_clamp`, `gre_key`), limits (`limit*`, `quota*`). Raw carriers (`faketcp`, `icmp`, `spoof`, `l2tcp`,
`shabdiz-gre|gre6|esp|ipip|6to4|etherip|l2tp|mpls|l2`) are offered but marked "needs root / CAP_NET_RAW".

**Xray reach.** `RelayPort > 0` gives the portal a SOCKS outbound `tunnel-<name>` to the loopback relay and the
bridge a loopback SOCKS inbound that routes to the exit outbound, with the same `RelayCredentials`. Because
Rakhsh forwards work in direct mode too, a Rakhsh tunnel is Xray-routable in both modes (BackPack's L3 is not).
UDP forwards (`udp = true`) are plain forwards, not part of the relay.

**Supervisor and status.** One `rakhsh run -c <toml>` per enabled tunnel, config 0600, restarted on change or
crash, with a per-tunnel `control_socket` in the tunnels folder. Status is `rakhsh status --socket <path> --json`:
connected when `lanes_up > 0`; the page shows lanes up/target, rtt, loss and the active link. Engine output goes to
the panel log with the same output log BackPack uses.

**Installer.** Pinned `v0.1.0` (the only tag). Assets are bare binaries named `rakhsh-<rust-target>` with a
`<asset>.sha256` next to them; verified before install, previous binary kept for rollback, node delivery in
512 KiB parts. Same `UpdateEngine` path, now per engine.

**Edit.** The entry end edits shared settings and pushes the new link to the node first; the exit end edits only
its exit outbound and local options — the semantics `tunnel_edit.go` already has.

**Frontend.** The engine selector gains Rakhsh; `TunnelOptionFields` reads `GET /panel/api/tunnels/options?engine=`;
a small links editor edits the `links` list. Reverse/Direct tags and the outbound-table rows need no change.

## Rulings (what I decided; cost if wrong)

- Router parts of Rakhsh are not exposed. Cost: a person who wants Rakhsh's own `[route]` cannot have it; adding
  the sections later is additive.
- The link carries the bridge's secret key, as Rakhsh's own link does. Cost: a leaked link lets someone run the
  bridge; the token is already enough to attempt the handshake. A manual public-key paste-back would avoid it at
  the price of a two-step setup.
- No loss-aware switching: Rakhsh's failover reacts to dead or blocked carriers, not slow ones.

## Verification and limits

Unit tests for rendering, validation, keys, link version 4, edit semantics, status parsing and installer (against a
local release server). An e2e test, gated on `DALAN_RAKHSH_E2E_BINARY`, runs `rakhsh check` on rendered configs
and carries a port through a real pair. **Not verifiable here:** that the v0.1.0 release carries the assumed assets
(the releases page was not reachable), behaviour on real servers, raw carriers, and anything under Orbit/Railway
(raw carriers and layer 3 need capabilities containers do not grant).
