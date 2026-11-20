# Node link on BackPack and Rakhsh, and removal of the Xray tunnel engine — design

Approved by the owner on 2026-11-20 (direction chosen after the code showed what the Xray tunnel carries; details below are the builder's calls unless marked).

## Why

The Xray reverse tunnel is the only engine that the node enrolment of `dalan/plans/2026-11-15-node-enrolment-design.md` knows. The owner decided to drop the Xray tunnel engine completely and to connect nodes through BackPack or Rakhsh instead. Those engines already carry a node's management: an entry tunnel gets a loopback forward to the node's panel port (`engine.WithManagement`, `CreateManagementRider`). What blocks them today is that a node cannot join through a tunnel it has not got yet, and that every fresh node is given an Xray channel.

## Goals

- "Add node with a command" asks how the node connects: which engine (BackPack by default, or Rakhsh) and which tunnel: a new tunnel for this node, or an existing entry tunnel the node already dials, including the one end users' traffic rides.
- A fresh node joins with one command and ends up with the dialling end of one tunnel; the panel reaches the node's panel port through a loopback forward on that tunnel.
- The Xray tunnel engine is gone: model constant, creation paths, config generation, probe path, UI option, docs.
- A node enrolled over Xray before this release is not left silently offline: the panel says it must be enrolled again.

## Non-goals

- Automatic migration of an Xray-enrolled node. The old channel disappears with the upgrade, so there is no path to move such a node remotely; it is enrolled again.
- A new management protocol. Tunnels stay the carrier.
- Changing how BackPack or Rakhsh themselves hide their traffic.

## Anti-goals

- Users' traffic on a shared tunnel reaching the node's panel port through the tunnel itself. The forward lives on a loopback port of the main panel and points at the node's loopback; the node's Xray keeps blocking private and loopback destinations, and a test pins the forward to loopback on both sides.
- A used or expired link working again. Unchanged.
- Silent loss of the old behaviour: an Xray tunnel or Xray-enrolled node that disappears without a message.

## Approaches considered

1. **Keep the Xray engine, hide it from the tunnel UI.** Smallest change, but the owner wants Xray tunnels redesigned later from scratch, and a hidden engine keeps all its code and its tests alive. Rejected by the owner.
2. **Carry node management on BackPack or Rakhsh (chosen).** The pieces exist; the change is the enrolment entry point and the removal of the Xray parts.
3. **A dedicated lightweight management protocol.** Another binary or another listener to keep hidden from the filter; much larger. Rejected.

## Design

### Enrolment spec

`enrolment.Spec` loses `Security` and `ServerName` (REALITY belongs to Xray) and `WithTunnel`, and gains `Engine` ("backpack" default, or "rakhsh"), `Transport` and `Options` (passed to the tunnel as the Tunnels page does; BackPack defaults to the `wssmux` transport). `Address` and `Port` stay: where the node dials on this server. `TunnelId` stays: ride an existing entry tunnel.

### Create

- `TunnelId > 0`: unchanged, a rider on that tunnel; its engine is the tunnel's. The node must already dial that tunnel.
- Otherwise: create the node's own entry tunnel named after the node (engine from the spec, relay port so it is also routable as an exit) and a rider on it named `node-<name>`. This replaces both the old "dedicated management tunnel" and the old "also make a tunnel for this node's traffic" option: one tunnel does both, and the Tunnels page lists it.

### Redeem and Join

- The grant always carries one tunnel link: the node's own tunnel when the panel made it; nothing when the node rides an existing tunnel.
- On the node, importing that link stores a bridge marked as the management channel (`ManagePort` = the node's panel port), replacing any earlier one: a node answers to one main panel. The marker keeps today's protections (cannot be deleted or probed from the node's Tunnels page).
- The enrolment link version goes from 1 to 2, so an old node gets a clear "version" error instead of a half join.

### Xray removal

Remove everything that exists only for `Engine == "xray"`: `TunnelEngineXray`, the Xray branch of `tunnel.NewPortal`/`SetupLink`/`ParseSetupLink` and REALITY key generation for tunnels, the portal and bridge generation, probe inbound and `ProbePort` in `xrayreverse.Apply`, the Xray branches in `tunnel_management.go` and `engine_route.go`, the Xray option and fields in the Tunnels and enrolment dialogs, their tests and the Xray parts of `dalan/TUNNELS.md`. Keep what BackPack, Rakhsh, pools and Tor use (routable outbound tags, pool balancers, relay outbounds and inbounds, outbound stats) and move it out of a package called `xrayreverse` into one named for what it does (`tunnelroute`). `Engine`'s database default changes from `xray` to `backpack`.

### Migration

On upgrade, in `internal/database/db.go`: delete every tunnel with engine `xray` (and the riders and pool memberships that point at them), and mark the enrolments whose channel was one of them as `needs_reenrol`. The Nodes page shows those nodes with a "Enrol again" notice and the command dialog prefilled with the node's name; enrolling again replaces the stale node row.

### UI

The enrolment dialog gets "Engine" and "How should it connect?" (new tunnel for this node, or an existing tunnel, listing the tunnels that can carry it, with a note on which ones users' traffic already rides). The Tunnels page loses the Xray option. The final look comes with the 0.6.0 redesign; this change keeps the current styling.

## Verification

- Unit: enrolment `Create` for BackPack and Rakhsh in both modes; `Redeem` grant contents; `Join` stores a marked bridge and replaces an older one; the version bump refuses a v1 link.
- The management forward renders as loopback to loopback in BackPack and Rakhsh configs.
- Migration test on a database seeded with Xray tunnels, riders, pools and an Xray-enrolled node.
- `make node-e2e` (a real master and node): a fresh node joins over BackPack and goes online through the tunnel. `internal/nodee2e` currently builds an Xray core for the enrolment test; that setup changes with this.
- Frontend: dialog tests for engine and tunnel choice; Nodes page notice for a node that must be enrolled again.

## Risks and checks owed to the owner

- BackPack's transports (`tcp, tcpmux, stealth, ws, wss, wsmux, wssmux, kcp, quic, udp, xdi, pck`) differ from REALITY's disguise. The node link is the most critical connection, so the default is `wssmux` and the docs say which transports to prefer; whether any of them hides as well as REALITY has to be judged on a real Iranian link.
- Every node needs the engine binary. Engines install by default on a first install since 0.5.2; older nodes run `x-ui install-engines`.
- Release notes: Xray tunnels stop working and nodes enrolled over Xray must be enrolled again.
