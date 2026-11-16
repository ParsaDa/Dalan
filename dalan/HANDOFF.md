# Dalan — handoff

Written 2026-11-15, updated 2026-11-16 for 0.4.5, for whoever continues this work (a person or a coding agent). Read this first, then `CLAUDE.md` (upstream's rules, all still binding), `dalan/UPSTREAM.md` and `dalan/TUNNELS.md`.

## What Dalan is

A fork of [3x-ui](https://github.com/MHSanaei/3x-ui) (`ParsaDa/dalan`, GPL-3.0, open source) that adds managed tunnels between servers. Owner: Parsa (`ParsaDa`). He works in Persian; write to him in Persian, plainly, without assuming deep Go knowledge.

Why it exists: during internet shutdowns in Iran only connections opened from outside toward Iran survive. Existing panels do not manage tunnels, so operators wire them by hand with separate scripts on both servers.

## Decisions already made (do not reopen without the owner)

| Decision | By |
|---|---|
| Fork 3x-ui and build tunnels into it, rather than a separate panel or a rewrite | owner |
| Users always connect to the Iran server; traffic exits from servers abroad | owner |
| **The main panel runs on the Iran server.** Servers abroad are nodes and must dial in | owner |
| Tunnels are reverse by default (exit server dials the entry server); direct tunnels also wanted | owner |
| Name: Dalan. Terminal command `dalan` (alias of `x-ui`) | owner |
| Two engines from the start: Xray VLESS reverse and BackPack | owner |
| A node chosen by the operator is a backup server: encrypted data store only, no automatic takeover | owner |
| UI redesign comes after tunnels work, and starts with a brainstorm with the owner | owner |
| Internal names stay unchanged (Go module path, `x-ui` binary, `/etc/x-ui`, `XUI_*`, unit names) so upstream keeps merging | Claude, accepted |
| Dalan's own version (`internal/config/dalan_version`) is used only by the update check; `internal/config/version` stays the 3x-ui version, because a master compares a node's reported panel version against upstream release numbers before pushing newer protocols (`internal/web/service/inbound_protocol.go`) | Claude |
| BackPack runs as its own unmodified binary (AGPL, trademarked name); its code is never copied | Claude |
| PasarGuard code (Python, AGPL) and nx-panel (closed source) are not used; only ideas | Claude |

Design documents live in the claude.ai Project "VPN and Networking": `design/2026-11-15-panel-design.md` and `design/2026-11-15-addendum-xray-reverse.md` (Persian). Section 7.2 of the first describes the reverse management channel (built in 0.3.1).

## State of the repository

`main` is at Dalan **0.4.0** (PR #13, released as `v0.4.0`); branch `feat/dalan-0.4.5` carries **0.4.5**. `v0.3.1` has working binaries but its Docker image never built (the Dockerfile compiled `main.go` alone and missed `enrol_cmd.go`); `v0.3.2` fixes that. Releases so far: `v0.1.0`, `v0.2.0`, plus the rolling `dev-latest` pre-release built on every push to `main`.

| PR | What |
|---|---|
| #1 | Foundation: updater and install scripts point at this repo, own version, `dalan` command, brand in UI and README, `scripts/dalan/rebrand.sh` + `check-brand.sh`, upstream sync script and runbook, upstream-only workflows skipped |
| #2 | Container images go to GHCR only (upstream's Docker Hub target removed by the rebrand script) |
| #3 | Reverse tunnels on Xray VLESS reverse, setup link, `/panel/api/tunnels/*`, Tunnels page |
| #4 | 0.2.0 |
| #5 | Tunnel status: real request through the tunnel, latency and exit location |
| #6 | BackPack engine: config rendering, process supervisor, verified installer, status, UI |
| #7 | 0.3.0 |

Merged in PR #9 (0.3.1): reverse node enrolment and the management channel, item 1 of the list below. Design: `dalan/plans/2026-11-15-node-enrolment-design.md`. Operator guide: "Nodes that dial in" in `dalan/TUNNELS.md`.

### What 0.4.5 adds

- **Reverse / direct label** on every tunnel (`ServerTunnel.Mode`, empty reads as reverse) in the Tunnels page and the outbounds table.
- **Direct layer-3 tunnels** (BackPack `[l3]`): `backpack.renderDirect`, link version 3 (`tunnel.setupLinkBackpackOptions`) mirrors the `local_ip`/`peer_ip` pair for the exit end; `listensOnPort` decides which end opens its port (the exit end of a direct tunnel); status is the link interface `dl<id>` (`TunnelService.directStatus`); no relay port, so it is never an Xray outbound (`xrayreverse.Routable`).
- **All reverse options of BackPack 1.9.0**: `backpack.Catalog` (`internal/tunnel/backpack/options.go`), served by `GET /panel/api/tunnels/options`, validated by `ValidateOptions`/`RenderConfig`, stored in `ServerTunnel.Options`, shown by `TunnelOptionFields.tsx`. Pin moved to `v1.9.0`.
- **Edit in place**: `TunnelService.Update` (`tunnel_edit.go`), `POST /panel/api/tunnels/update/:id`; an entry end made for a node pushes the new link first (`pushToNode`); `Import` of a link whose name is already a dialing end updates it (`updateDialingEnd`).
- **Live traffic**: the websocket `outbounds` push also patches the tunnels query (`withLiveTraffic`); **tunnels in the Outbounds table** (read-only rows, no separate tunnel list there).

### What 0.4.0 adds (PR #13)

- **Tunnels as Xray outbounds.** An entry tunnel is the outbound `tunnel-<name>`; an inbound or routing rule can choose it. A management tunnel (`ManagePort > 0`) is never offered. `xrayreverse.Routable` says which tunnels count.
- **Node management over the existing tunnel**, BackPack engine log lines in the panel log, per-tunnel traffic totals (Apply switches outbound stats on for entry tunnels in the generated config).
- **One-panel tunnel creation**: *New tunnel → Create the other end on a node*; enrolling a node can also create its data tunnel (`withTunnel`, default on); deleting removes both ends, with "remove this end only" when the node is unreachable. `refuseWhileInUse` stops deleting a tunnel a pool or a node still uses.
- **Spare tunnels**: `internal/tunnel/pool` (pure decision: three checks in a row, at least 50 ms and 30% faster, or the tunnel in use is down), `internal/tunnel/xrayreverse/pool.go` (balancer `pool-<name>`, `ApplyPools`), `TunnelPoolService`, job `tunnel_pool_job.go` every 15 s, hot switch through `XrayService.OverrideBalancer`.
- **Route test**: `internal/tunnel/dial.go`, `Remote.DialPorts`, `TunnelService.RouteTest` — this server listens, the node dials, a port counts only if the connection arrived here.
- **Engine delivery and update**: `backpack.Fetch/Place/Rollback/LatestVersion`, `TunnelService.UpdateEngine` (rolls back when connected tunnels do not return within 75 s), the node serves a release in 512 KiB parts (`EngineReleaseChunk`, `Remote.FetchEngineRelease`) because every node call has a 10 s limit. A stopped engine's status file is deleted so it cannot read as connected.
- **Install and enrol in one command**: the enrolment dialog shows `XUI_ENROL='<link>' bash <(curl … install.sh)`. The hook in `install.sh` is added by `scripts/dalan/rebrand.sh` (`add_enrol_on_install`) so upstream syncs keep it.
- **Container**: `internal/frontdoor` (path routing from the panel's own inbounds/settings, `FlushInterval -1` for XHTTP, upgrades pass), `service.FrontDoorService`, `DockerPrepare.sh` (PORT → `DALAN_FRONT_PORT`, panel to `XUI_PORT=20530`, `DALAN_ADMIN_*`, one-time `DALAN_ENROL`). Guide: `dalan/CONTAINER.md`.

### Where the tunnel code is

- `internal/database/model/server_tunnel.go` — `ServerTunnel`, one end of a tunnel. Roles `portal` (entry, listens) and `bridge` (exit, dials). Engines `xray`, `backpack`.
- `internal/tunnel/setup.go` — `NewPortal`, `SetupLink`, `ParseSetupLink`. Link format `dalan://tunnel/<base64url JSON>`; version 1 = Xray, version 2 = BackPack, version 3 = BackPack with options or direct mode. The REALITY private key never enters a link.
- `internal/tunnel/xrayreverse/apply.go` — injects enabled Xray tunnels into the *generated* Xray config (never the stored template). Portal: VLESS inbound whose client has `reverse.tag`, plus a routing rule sending the chosen inbound tags to outbound `tunnel-<name>`, plus a loopback probe inbound on `62900 + id%500`. Bridge: VLESS outbound with `reverse.tag` and a rule sending its traffic to the exit outbound.
- `internal/tunnel/probe.go` — one GET through the probe port; reports latency, exit IP and country (Cloudflare trace).
- `internal/tunnel/backpack/` — `RenderConfig` (TOML), `Supervisor` (one process per tunnel, restart on change or crash, config file 0600 and removed with the tunnel), `ReadSnapshot` (`<name>.metrics.json`), `Install` (pinned `v1.9.0`, verified against the release's `SHA256SUMS`), `Catalog`/`ValidateOptions` (options.go), `renderDirect` and `DirectInterface` (config.go).
- `ServerTunnel.ManagePort` marks a management tunnel. `xrayreverse.Apply` gives the portal a loopback `dokodemo-door` inbound on that port and gives the bridge a freedom outbound whose `redirect` pins every tunnelled connection to `127.0.0.1:<ManagePort>` (the node's panel). `internal/web/service/tunnel_management.go` creates, replaces and deletes these; the ordinary tunnel routes refuse to delete or probe them.
- `internal/tunnel/enrol/` — the enrolment link (`dalan://enrol/...`), the sealed request and reply (X25519 + HKDF + AES-GCM, key in the link) and the node's HTTP client.
- `internal/web/service/enrolment/` — `Create`, `Redeem`, `Delete`, `NodeDeleted` on the main panel and `Join` on the node. It is its own package because `service/panel` imports `service`. `internal/database/model/node_enrolment.go` is its table. Routes in `internal/web/controller/enrolment.go`: `/panel/api/enrolments/*` and the public `POST <base path>dalan/enrol`. CLI: `enrol_cmd.go` (`x-ui enrol <link>`); `x-ui.sh` gets its `enrol` case from `scripts/dalan/rebrand.sh` and restarts Xray afterwards.
- `frontend/src/pages/nodes/EnrolNode.tsx`, `frontend/src/api/queries/useEnrolments.ts`, tests in `frontend/src/test/enrol-node.test.tsx`.
- `internal/web/service/tunnel.go`, `internal/web/controller/tunnel.go` — service and routes. The hook into config generation is one block in `XrayService.GetXrayConfig` (`internal/web/service/xray.go`); BackPack engines start in `startTask` and stop in the stop path of `internal/web/web.go`.
- `frontend/src/pages/tunnels/TunnelsPage.tsx`, `frontend/src/api/queries/useTunnels.ts`, tests in `frontend/src/test/tunnels-page.test.tsx`.
- Engine files at runtime: binary `<bin folder>/backpack` (normally `/usr/local/x-ui/bin/backpack`), configs and status in `<db folder>/tunnels/` (normally `/etc/x-ui/tunnels/`).

Every place Dalan touches upstream code is listed in `dalan/UPSTREAM.md`. Keep that table current; new features go in new files and packages.

## What is verified and what is not

Verified by automated tests that run in CI:
- Two real Xray cores in-process: a request entering the portal leaves from the bridge, and the probe works through the tunnel's own probe port (`internal/tunnel/xrayreverse/e2e_test.go`, excluded under `-race`, see below).
- A REALITY tunnel built from a setup link is accepted by Xray's own config loader.
- A management tunnel between two real Xray cores reaches the bridge's panel port and, whatever destination is asked for, nothing else on the bridge.
- Two real panel processes with real Xray cores (`make node-e2e`, which now builds the core from go.mod): a node enrols with the CLI, goes online through its tunnel, receives an inbound pushed over it, and the link cannot be used twice.
- Service, controller and page behaviour; installer against a local release server (good checksum, wrong checksum, missing checksum, failed download keeps the current engine); supervisor with a stand-in engine.

Verified once by hand, not in CI:
- A real BackPack v1.8.5 binary (0.4.0; not repeated for v1.9.0) (built from that tag with `CGO_ENABLED=0`) on both ends carries a forwarded TCP port: `DALAN_BACKPACK_E2E_BINARY=/path/to/backpack go test ./internal/tunnel/backpack/`.
- A real install of v0.1.0 on a server by the owner: panel came up, `dalan` command works.

**Not verified — waiting on the owner's tests:**
- Everything new in 0.4.5 on real servers: a direct tunnel (needs root and `/dev/net/tun`; no real v1.9.0 binary was run), reverse `ws`/`wss` behind a CDN with `edge_ip`, editing a node-made tunnel, the option catalog against the real engine (it was written from the v1.9.0 source and config reference, not from a running engine). Whether the `v1.9.0` GitHub release carries `backpack_linux_<arch>.tar.gz` and `SHA256SUMS` was not checked (the tag exists); if the install button answers 404, that is why.
- An Xray REALITY tunnel between two real servers (owner was testing `v0.2.0`; result not reported yet).
- Node enrolment between two real servers, and with REALITY: the automated tests run both panels on one host with tunnel security "none". `dalan enrol` through the real `x-ui.sh` (systemd, Alpine and Docker restart paths) has not been run.
- Tunnel status against a real tunnel.
- The BackPack install button on a real server. The installer itself was run once against the real GitHub release of v1.8.5 (download, checksum, unpack). Dalan 0.3.0 pinned v2.0.0, which BackPack has not published, so its install button always answers 404.
- BackPack transports other than `tcpmux`; anything across a filtered link.
- The conflict path of `scripts/dalan/sync-upstream.sh` (upstream had not moved since the fork).
- Whether the Docker image workflow succeeds on a tag after PR #2 (it failed on `v0.1.0`; not rechecked for `v0.2.0`).

## Known gaps and rough edges

- Translations: only English and Persian are translated; the other 12 locale files carry the English text for every `pages.tunnels.*` key.
- The panel still shows the 3x-ui version number (3.11.0) as its version; Dalan's version appears only in the update dialog.
- Xray tunnels: only REALITY over raw TCP (or no security). No WS/XHTTP/gRPC/TLS for the tunnel link yet.
- An Xray tunnel is tested only from the entry end. The exit end shows "tested from the entry server".
- Traffic totals are shown for entry tunnels that are Xray outbounds; direct tunnels show none.
- BackPack: forwarded-port conflicts with local inbounds are not checked; `skip_optz` is left at the engine default, so BackPack applies its own TCP tuning to the host.
- Deleting a tunnel on one server leaves the other end in place. The same holds for an enrolled node: deleting it on the main panel leaves the bridge `node-<name>` and the main panel's API token on the node.
- Enrolment: a node whose panel listens on one specific IP is refused (the bridge delivers to 127.0.0.1); each node takes its own tunnel port on the main panel; an https node is registered with certificate verification off, because the hop is loopback through the tunnel; a management tunnel has no status of its own in the Tunnels page (the node's status is its health); a Join that imports the data tunnel after the management tunnel has no rollback; joining a second main panel with the same data-tunnel name clashes.
- `docs/content/docs/en/reference/api/meta.json` lists neither `tunnels` nor `node-enrolment`, so those generated pages are not in the docs site navigation.
- A data race inside xray-core's VLESS reverse monitor (`proxy/vless/outbound`, handler read while `New` is still writing it) makes the two-core traffic test unusable under `-race`; the file carries `//go:build !race`. Worth reporting upstream.
- `node-e2e` in CI failed once with `bind: address already in use` in upstream's harness and passed on re-run. Treat one such failure as a flake, a second as real.

## What is left, in order

1. **Real-server tests of 0.4.0** (owner): tunnels as outbounds, pools, route test, engine update through a node, install-and-enrol, and the Docker image on Railway. Nothing here has run on real servers; the Docker image was not built in the sandbox (no daemon), CI builds it on the tag.
2. **Direct tunnel gaps**: not an Xray outbound, cannot cross a CDN (UDP/raw/ICMP carriers), the `spoof` carrier and BackPack's Naive/Xray helper sections are not offered. The **backup server** (design §7.7) is cancelled for now. Pool balancer tags are not offered in the Routing page's balancer picker (they can be typed as `pool-<name>`); BackPack members of a pool report no latency and only replace a tunnel that is down.
3. **One shared port for all management tunnels**, and Xray tunnels over transports other than REALITY/TCP.
4. **Node tabs**: the owner's idea — each enrolled node gets a tab in the main panel that shows that node's whole management panel (the main panel already has the node's API token and a path to it through the management tunnel; needs request forwarding plus a node-scoped UI). Fits the UI redesign.
5. **UI redesign** — only after a brainstorm with the owner; he asked to be consulted first.
6. Later: more engines (Arange-tun, C-Tunnel for GRE, GOST), user-management ideas from PasarGuard, an optional SmartDNS module (idea from hyperdns).

## How to work in this repo

- Follow `CLAUDE.md`: TDD with tests that can fail, comments ≤ 2 lines, conventional commits with a body, one focused diff per PR.
- A new route needs, in the same PR: an entry in `frontend/src/pages/api-docs/endpoints.ts`, `make gen`, copy `frontend/public/openapi.json` to `docs/public/openapi.json`, and `cd docs && pnpm gen:api`. A new struct in a response needs `StructAllow` in `tools/openapigen/main.go`.
- A new i18n key goes in all 14 files under `internal/web/translation/` and must be used in the same commit (`frontend/src/test/i18n-dead-keys.test.ts`).
- A new model goes in both `internal/database/db.go` (AutoMigrate list) and `internal/database/migrate_data.go` (`migrationModels`).
- Go imports: standard library, third party, then this module, each as its own group (goimports with a local prefix). Requests need a context (`noctx`), including `exec.CommandContext`.
- Local gate: `make verify` (needs Go 1.27, Node 26, `npm ci` in `frontend/`, Playwright Chromium). CI additionally runs `make race`, `make vulncheck`, live Postgres and `make node-e2e`.
- `make brand-check` must stay green. Never hand-edit a URL in a root `*.sh` file; change `scripts/dalan/rebrand.sh` and re-run it.
- Upstream updates: `bash scripts/dalan/sync-upstream.sh`, then `make verify`. See `dalan/UPSTREAM.md`.

## Releasing

1. Raise `internal/config/dalan_version` in a PR and merge it. The update check compares this with the newest release tag, so it must move before the tag.
2. Create the release on GitHub with a new tag `vX.Y.Z` on `main`, marked latest.
3. `release.yml` uploads the build to that release as a pre-release upload; afterwards confirm the release is still marked **Latest**. The installer and the in-panel update only see the latest release.

Note: CI does not run for a PR that changes only `dalan_version` (path filters).

## Reference projects that were reviewed

3x-ui (base), PasarGuard panel (ideas for user management), nx-panel (closed source, unusable), BackPack (engine, proved itself in the long shutdown per the owner), Arange-tun, C-Tunnel, Gost-Tunnel-Manager (candidate engines), veilway (alpha, not a server-to-server tunnel), hyperdns (SmartDNS/SNI proxy, not a tunnel), and the owner's `ParsaDa/flxui` (single-port launcher for 3x-ui on PaaS).
