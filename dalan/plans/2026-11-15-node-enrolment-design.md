# Reverse node enrolment and management channel — design

Approved by the owner on 2026-11-15. Item 1 of "What is left" in `dalan/HANDOFF.md`; design §7.2.

## Why

The main panel runs on the Iran server. 3x-ui's master dials its nodes, and in a shutdown only connections opened from outside toward Iran survive, so the master loses every node abroad. The node must dial in, and the master must manage it through that connection.

## What the operator sees

1. On the Iran panel, Nodes → "Add node with a command": node name, management tunnel port on this server, SNI.
2. The panel shows one command: `dalan enrol dalan://enrol/...`. Its token works once and expires after 30 minutes.
3. The command is run on the server abroad, where Dalan is already installed. The node calls the Iran panel, opens a reverse tunnel, and appears online in the node list.

After that the Iran panel opens no connection toward the outside.

## Goals

- Enrolment and all later management work with only outside-to-Iran connections.
- The existing node layer (`runtime.Remote`, heartbeat, sync) is used unchanged; the node is registered at `127.0.0.1:<port>`.
- No extra binary: the management tunnel uses the Xray engine.

## Non-goals

- A tunnel for user traffic created with the node (item 2).
- Installing Dalan and enrolling in one command.
- One shared port for the management tunnels of all nodes.

## Anti-goals

- Traffic arriving through a management tunnel reaching anything on the node other than the node's own panel port.
- The node's API token or the tunnel secret readable on the wire when the Iran panel is served over plain HTTP.
- An enrolment token that can be used twice or after it expires.

## Approaches considered

1. **Enrolment endpoint plus a dedicated management tunnel (chosen).** The node posts its panel details and a fresh API token to the Iran panel and receives the tunnel's dialing end. Matches design §7.2; the Iran panel learns the node's scheme and base path directly.
2. **Everything in the link, no endpoint.** The Iran panel would pre-generate the tunnel and the API token and put both in the link. No public endpoint, but the Iran panel cannot learn the node's scheme and base path, and a VLESS reverse tunnel has no bridge-to-portal direction to report them through.
3. **Management over the user-traffic tunnel.** One tunnel per node for both. Users routed through the tunnel could then address the node's panel port; keeping management on its own tunnel makes the anti-goal structural.

## Design

### Management port on an Xray tunnel (PR 1)

`ServerTunnel.ManagePort` marks a management tunnel.

- Portal (Iran): a loopback-only inbound on `127.0.0.1:ManagePort`, routed into `tunnel-<name>`. No probe inbound and no `RouteInbounds`.
- Bridge (node): everything arriving through the tunnel goes to a dedicated freedom outbound that redirects to `127.0.0.1:ManagePort` (the node's panel port) and allows only that address. The destination the portal asked for is ignored.

### Enrolment (PR 2)

- `NodeEnrolment` row: node name, token hash, expiry, used-at, the portal tunnel it created.
- Link `dalan://enrol/<base64url JSON>`: panel URL, one-time token, the panel's X25519 public key for this enrolment.
- Public endpoint on the Iran panel. Request and response bodies are sealed with a key agreed from the link's public key, so a plain-HTTP panel leaks nothing. The token is consumed atomically.
- Node side, `x-ui enrol <link>` (`dalan enrol`): creates an API token, sends it with the panel's port, scheme and base path, stores the returned bridge tunnel with `ManagePort` = its own panel port, and restarts the panel.
- Iran side: creates the portal tunnel when the enrolment is created; on redemption registers `model.Node` at `127.0.0.1:<ManagePort>` with `AllowPrivateAddress`.
- Deleting an enrolled node deletes its management tunnel on the Iran side.

### UI and docs (PR 3)

Nodes page dialog with the command, English and Persian translations, `dalan/TUNNELS.md`, `dalan/UPSTREAM.md`, handoff.

## Verification

- PR 1: two real Xray cores; a request to the portal's loopback port reaches an HTTP server on the bridge's loopback, and a request addressed elsewhere still lands on that server only.
- PR 2: a real master and node panel (`make node-e2e`); after enrolment the node goes online through the tunnel. Reused and expired tokens are refused.
- PR 3: page tests.

## Calls made by the builder (owner may change)

- Dalan must already be installed on the node.
- The panel suggests the management port; the operator can change it.
- Token lifetime 30 minutes.
