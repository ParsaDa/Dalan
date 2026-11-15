# Dalan — handoff

Written 2026-11-15 for whoever continues this work (a person or a coding agent). Read this first, then `CLAUDE.md` (upstream's rules, all still binding), `dalan/UPSTREAM.md` and `dalan/TUNNELS.md`.

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

Design documents live in the claude.ai Project "VPN and Networking": `design/2026-11-15-panel-design.md` and `design/2026-11-15-addendum-xray-reverse.md` (Persian). Section 7.2 of the first describes the reverse management channel that is not built yet.

## State of the repository

`main` is at Dalan **0.3.0** (version bumped, tag `v0.3.0` to be created by the owner). Releases so far: `v0.1.0`, `v0.2.0`, plus the rolling `dev-latest` pre-release built on every push to `main`.

| PR | What |
|---|---|
| #1 | Foundation: updater and install scripts point at this repo, own version, `dalan` command, brand in UI and README, `scripts/dalan/rebrand.sh` + `check-brand.sh`, upstream sync script and runbook, upstream-only workflows skipped |
| #2 | Container images go to GHCR only (upstream's Docker Hub target removed by the rebrand script) |
| #3 | Reverse tunnels on Xray VLESS reverse, setup link, `/panel/api/tunnels/*`, Tunnels page |
| #4 | 0.2.0 |
| #5 | Tunnel status: real request through the tunnel, latency and exit location |
| #6 | BackPack engine: config rendering, process supervisor, verified installer, status, UI |
| #7 | 0.3.0 |

Unreleased on branch `claude/new-session-zu5aqy` (no PR opened yet): reverse node enrolment and the management channel, item 1 of the list below. Design: `dalan/plans/2026-11-15-node-enrolment-design.md`. Operator guide: "Nodes that dial in" in `dalan/TUNNELS.md`.

### Where the tunnel code is

- `internal/database/model/server_tunnel.go` — `ServerTunnel`, one end of a tunnel. Roles `portal` (entry, listens) and `bridge` (exit, dials). Engines `xray`, `backpack`.
- `internal/tunnel/setup.go` — `NewPortal`, `SetupLink`, `ParseSetupLink`. Link format `dalan://tunnel/<base64url JSON>`; version 1 = Xray, version 2 = BackPack. The REALITY private key never enters a link.
- `internal/tunnel/xrayreverse/apply.go` — injects enabled Xray tunnels into the *generated* Xray config (never the stored template). Portal: VLESS inbound whose client has `reverse.tag`, plus a routing rule sending the chosen inbound tags to outbound `tunnel-<name>`, plus a loopback probe inbound on `62900 + id%500`. Bridge: VLESS outbound with `reverse.tag` and a rule sending its traffic to the exit outbound.
- `internal/tunnel/probe.go` — one GET through the probe port; reports latency, exit IP and country (Cloudflare trace).
- `internal/tunnel/backpack/` — `RenderConfig` (TOML), `Supervisor` (one process per tunnel, restart on change or crash, config file 0600 and removed with the tunnel), `ReadSnapshot` (`<name>.metrics.json`), `Install` (pinned `v2.0.0`, verified against the release's `SHA256SUMS`).
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
- A real BackPack v2.0.0 binary (built from source with `CGO_ENABLED=0`) on both ends carries a forwarded TCP port: `DALAN_BACKPACK_E2E_BINARY=/path/to/backpack go test ./internal/tunnel/backpack/`.
- A real install of v0.1.0 on a server by the owner: panel came up, `dalan` command works.

**Not verified — waiting on the owner's tests:**
- An Xray REALITY tunnel between two real servers (owner was testing `v0.2.0`; result not reported yet).
- Node enrolment between two real servers, and with REALITY: the automated tests run both panels on one host with tunnel security "none". `dalan enrol` through the real `x-ui.sh` (systemd, Alpine and Docker restart paths) has not been run.
- Tunnel status against a real tunnel.
- The BackPack installer against the real GitHub release. Asset name `backpack_linux_<arch>.tar.gz`, archive member `backpack` and `SHA256SUMS` were taken from BackPack's `install.sh`, not from a real download.
- BackPack transports other than `tcpmux`; anything across a filtered link.
- The conflict path of `scripts/dalan/sync-upstream.sh` (upstream had not moved since the fork).
- Whether the Docker image workflow succeeds on a tag after PR #2 (it failed on `v0.1.0`; not rechecked for `v0.2.0`).

## Known gaps and rough edges

- Translations: only English and Persian are translated; the other 12 locale files carry the English text for every `pages.tunnels.*` key.
- The panel still shows the 3x-ui version number (3.11.0) as its version; Dalan's version appears only in the update dialog.
- Xray tunnels: only REALITY over raw TCP (or no security). No WS/XHTTP/gRPC/TLS for the tunnel link yet.
- An Xray tunnel is tested only from the entry end. The exit end shows "tested from the entry server".
- No traffic totals per tunnel in the UI (BackPack's snapshot has them; Xray has outbound stats).
- BackPack: no way to update an installed engine from the panel; on a server that cannot reach GitHub the binary must be copied by hand; forwarded-port conflicts with local inbounds are not checked; `skip_optz` is left at the engine default, so BackPack applies its own TCP tuning to the host.
- Deleting a tunnel on one server leaves the other end in place. The same holds for an enrolled node: deleting it on the main panel leaves the bridge `node-<name>` and the main panel's API token on the node.
- Enrolment: a node whose panel listens on one specific IP is refused (the bridge delivers to 127.0.0.1); each node takes its own tunnel port on the main panel; an https node is registered with certificate verification off, because the hop is loopback through the tunnel; a management tunnel has no status of its own in the Tunnels page (the node's status is its health); Dalan must already be installed on the node.
- `docs/content/docs/en/reference/api/meta.json` lists neither `tunnels` nor `node-enrolment`, so those generated pages are not in the docs site navigation.
- A data race inside xray-core's VLESS reverse monitor (`proxy/vless/outbound`, handler read while `New` is still writing it) makes the two-core traffic test unusable under `-race`; the file carries `//go:build !race`. Worth reporting upstream.
- `node-e2e` in CI failed once with `bind: address already in use` in upstream's harness and passed on re-run. Treat one such failure as a flake, a second as real.

## What is left, in order

1. **Reverse node enrolment and management channel** — built on the branch above, waiting for the owner's test on two real servers and a PR. Still open inside it: one shared port for all management tunnels, and install plus enrol in one command.
2. **One-panel tunnel creation**: with the management channel in place, the Iran panel creates both ends itself instead of copy-pasting a link, and a tunnel is created automatically when a node is added.
3. **Spare tunnels and automatic switching**: several tunnels per route on different transports and ports, health from the probe (Xray) and snapshot (BackPack), switch only after the alternative has been better several times in a row. For Xray, a balancer over `tunnel-<name>` outbounds is the natural mechanism. Plus a route test before creation (which transports pass between two servers), direct tunnels, and the backup server (design §7.7).
4. **Delivering engine binaries and updates to the Iran server through the node abroad**, so nothing depends on the Iran server reaching GitHub.
5. **Container build** for Orbit, Railway and AWS: one HTTP port in front of the panel, subscription and WS/XHTTP inbounds (the logic of the owner's `ParsaDa/flxui`), node abroad only, Xray engine as the main one because BackPack transports beyond WS/WSS need root or UDP.
6. **UI redesign** — only after a brainstorm with the owner; he asked to be consulted first.
7. Later: more engines (Arange-tun, C-Tunnel for GRE, GOST), user-management ideas from PasarGuard, an optional SmartDNS module (idea from hyperdns), automatic takeover by the backup server.

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
