# Rakhsh Engine Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans (inline) to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Dalan 0.5.0 runs Rakhsh as a third tunnel engine with every tunnel-shaping option, created, edited, shown and routed like BackPack tunnels.

**Architecture:** New package `internal/tunnel/rakhsh` (keys, catalog, TOML rendering, status, installer) beside `backpack`; the process supervisor is extracted into `internal/tunnel/engine` so both engines share it. `ServerTunnel` gains two key columns; setup link version 4 carries the keys; `xrayreverse`, `TunnelService` and the controller dispatch on engine; the existing option form gets an engine parameter and a links editor.

**Tech Stack:** Go 1.27, Gin, GORM, `crypto/ecdh`; React 19 + Ant Design 6 + Zod; Rakhsh v0.1.0 binary (external).

**Spec:** `docs/superpowers/specs/2026-11-16-rakhsh-engine-design.md`

## Global Constraints

- Repo rules in `CLAUDE.md` bind every task: red-first tests that name the failure they catch, comments ≤ 2 lines, gofumpt/goimports layout, `noctx`, conventional commits ending with the two attribution lines.
- Engine id `rakhsh`; Rakhsh pinned to `v0.1.0` (`rakhsh.PinnedVersion`); assets are bare binaries `rakhsh-<rust-target>` with `<asset>.sha256` beside them (targets: `x86_64-unknown-linux-musl`, `aarch64-unknown-linux-musl`, `armv7-unknown-linux-musleabihf`, `arm-unknown-linux-musleabi`, `i686-unknown-linux-musl`).
- Roles: portal = Rakhsh `entry`, bridge = `exit`. Reverse mode writes `dial = "exit"`; direct mode writes `dial = "entry"`.
- Key format: X25519, 32 bytes, standard base64 (`rakhsh keygen`). `static_secret` is never serialised to JSON.
- Setup link version 4 (`dalan://tunnel/<base64url JSON>`); versions 1–3 keep parsing.
- Relay: `RelayPort > 0` makes a Rakhsh tunnel Xray-routable in both modes, with `backpack.RelayCredentials` semantics.
- A new route or struct follows the CLAUDE.md chain (endpoints.ts, StructAllow, `make gen`, copy openapi to `docs/public/`, `pnpm gen:api`). A new i18n key goes in all 14 locale files and is used in the same commit.
- Dalan version `0.5.0` in `internal/config/dalan_version`; Persian + English text only (other locales carry English).

## Review Focus

- Editing the exit end must keep its keys and token (a regenerated key silently unpairs the tunnel).
- The bridge's secret key must never appear in `GET /panel/api/tunnels/list` or in logs.
- Two Rakhsh tunnels on one host need distinct control sockets and config files.
- `links` entries with a duplicate carrier+address, or a raw carrier without `peer_ip`, must be rejected with a readable message, not rendered.
- Status with the engine missing, stopped or just restarted must say so, not read as connected.

---

### Task 1: Engine constant, key columns, key pairs

**Files:**
- Modify: `internal/database/model/server_tunnel.go`, `internal/database/db.go` (migration notes), `internal/tunnel/xrayreverse/apply.go`, `internal/tunnel/xrayreverse/pool.go`
- Create: `internal/tunnel/rakhsh/keys.go`, `internal/tunnel/rakhsh/keys_test.go`

**Interfaces:**
- Produces: `model.TunnelEngineRakhsh = "rakhsh"`; `model.ExternalEngine(engine string) bool` (true for backpack and rakhsh); `ServerTunnel.RakhshSecret string` (`json:"-"`, column `rakhsh_secret`) and `ServerTunnel.RakhshPeerPublic string` (column `rakhsh_peer_public`, JSON `rakhshPeerPublic`); `rakhsh.GenerateKeyPair() (secret, public string, err error)`; `rakhsh.PublicKey(secret string) (string, error)`.

- [ ] **Step 1: Write the failing tests.** `TestGenerateKeyPairDerivesPublic`: pair decodes to two 32-byte values and `PublicKey(secret)` equals `public`; two calls differ. `TestPublicKeyRejectsBadSecret`: non-base64 and 31-byte input return errors containing "secret". `TestExternalEngine` table: xray false, backpack true, rakhsh true. `TestRoutableCountsRakhshRelay`: a rakhsh portal with `RelayPort 63500` is Routable, with 0 is not.
- [ ] **Step 2: Run** `go test ./internal/tunnel/rakhsh/ ./internal/database/model/ ./internal/tunnel/xrayreverse/ -run 'KeyPair|PublicKey|ExternalEngine|RoutableCountsRakhshRelay'`. Expected: FAIL, undefined symbols.
- [ ] **Step 3: Implement** the constant, `ExternalEngine`, the two fields (AutoMigrate adds the columns; confirm no `migrate_data.go` change is needed because no new model), and the key functions with `crypto/ecdh` X25519. Replace every `== model.TunnelEngineBackpack` / `!= ...` in `xrayreverse/apply.go` and `pool.go` with `model.ExternalEngine(t.Engine)` so Rakhsh follows the same relay path.
- [ ] **Step 4: Run** the same command plus `go test ./internal/tunnel/xrayreverse/`. Expected: PASS.
- [ ] **Step 5: Commit** `feat(tunnels): add the rakhsh engine id, key columns and X25519 key pairs`.

### Task 2: Option catalog and validation

**Files:**
- Create: `internal/tunnel/rakhsh/options.go`, `internal/tunnel/rakhsh/options_test.go`
- Modify: `tools/openapigen/main.go` (allow `rakhsh.Option`)

**Interfaces:**
- Produces: `rakhsh.Option` (same fields and JSON as `backpack.Option`, plus `KindLinks = "links"`), `rakhsh.Catalog []Option`, `rakhsh.OptionsFor(mode string) []Option`, `rakhsh.ValidateOptions(mode string, options map[string]any) error`, `rakhsh.Carriers []string` (the carrier names of the spec), `rakhsh.RawCarriers []string` (need root).
- Option values: `links` is `[]any` of `map[string]any` with `carrier`, `addr` and any per-link override key from the catalog (`sni`, `ws_path`, `ws_credential`, `decoy`, `edge_ip`, `tls_fingerprint`, `cert_path`, `key_path`, `quic_mask`, `quic_version`, `reliability`).

- [ ] **Step 1: Write failing tests.** `TestCatalogCoversExampleConfig`: reads `rakhsh.example.toml` (embedded copy `testdata/rakhsh.example.toml`) and fails if a top-level key outside the excluded router sections (`inbound`, `outbound`, `route`, `dns`, `openvpn`, `xui_*`, `accept_sync`, `sync_*`, `access_log`, `control_socket`, `token`, `static_secret`, `peer_public`, `role`, `dial`, `tunnel_listen`, `tunnel_connect`, `carrier`, `lanes`) is neither in the catalog nor in an explicit `deliberatelyOmitted` list in the test. `TestValidateOptionsRejects` table: unknown key; `lanes` 0; `preset` "fast"; `limit_up` "100" (no unit); `quota_period` "year"; `fec_data` without `fec_parity`; `l3_addr` "10.0.0.1/99"; `links` entry with unknown carrier; two `links` with the same carrier and addr; a raw carrier link without `peer_ip` in options — each asserting the exact error text. `TestValidateOptionsAccepts`: a full realistic option set per mode. `TestOptionsForMode`: direct-only and reverse-only options are filtered.
- [ ] **Step 2: Run** `go test ./internal/tunnel/rakhsh/ -run 'Catalog|ValidateOptions|OptionsFor'`. Expected: FAIL.
- [ ] **Step 3: Implement** the catalog (key, kind, role, mode, group, hints in English and Persian, bounds) following `backpack/options.go`'s layout and validators (reuse its patterns by copying the small ones; do not import `backpack` from `rakhsh`). Groups: `identity`, `tunnel`, `carrier`, `links`, `pacing`, `raw`, `layer3`, `limits`.
- [ ] **Step 4: Run** the same command. Expected: PASS. Also `go test ./tools/openapigen/`.
- [ ] **Step 5: Commit** `feat(tunnels): add the Rakhsh option catalog and validation`.

### Task 3: Render the configuration

**Files:**
- Create: `internal/tunnel/rakhsh/config.go`, `internal/tunnel/rakhsh/config_test.go`

**Interfaces:**
- Consumes: Task 1 fields and `ValidateOptions`.
- Produces: `rakhsh.RenderConfig(t model.ServerTunnel, controlSocket string) (string, error)`; `rakhsh.ControlSocket(dir, name string) string`; `rakhsh.RelayForward(port int) string` returning the `[[forward]]` pair for the loopback relay.

- [ ] **Step 1: Write failing tests** (compare rendered text exactly): reverse portal on `wss` with a forward, relay and two links renders `role = "entry"`, `dial = "exit"`, `tunnel_listen`, `carrier`, `token`, `static_secret`, `peer_public`, `[[link]]` blocks in order, and one `[[forward]]` per forward plus the relay; the matching bridge renders `role = "exit"`, `tunnel_connect` and no forwards; direct mode swaps `dial` to `entry` and the listen/connect sides; `l3_addr` is mirrored `.1/.2`; a UDP forward (`"51820=127.0.0.1:51820/udp"`) renders `udp = true`; an unknown role, a missing secret, or a forward containing a quote returns an error naming the tunnel. Golden strings are written by hand from `rakhsh.example.toml`, never regenerated.
- [ ] **Step 2: Run** `go test ./internal/tunnel/rakhsh/ -run RenderConfig`. Expected: FAIL.
- [ ] **Step 3: Implement** `RenderConfig`: typed fields first, then options from the catalog in catalog order, with `links` and per-forward limits written as TOML tables; quote every string with `strconv.Quote`.
- [ ] **Step 4: Run** the package tests. Expected: PASS.
- [ ] **Step 5: Commit** `feat(tunnels): render Rakhsh configuration for both ends`.

### Task 4: Shared supervisor, Rakhsh status, installer

**Files:**
- Create: `internal/tunnel/engine/supervisor.go` (+ test moved from `backpack/supervisor_test.go`), `internal/tunnel/engine/output.go`, `internal/tunnel/rakhsh/status.go`, `internal/tunnel/rakhsh/install.go` and tests
- Modify: `internal/tunnel/backpack/supervisor.go`, `internal/tunnel/backpack/output.go` (thin wrappers over `engine`)

**Interfaces:**
- Produces: `engine.Supervisor{Name string; Binary, Dir string; RestartDelay time.Duration; LogLine func(tunnel, line string); Wants func(model.ServerTunnel) bool; Render func(model.ServerTunnel) (string, error); Args func(configPath string) []string}` with `Reconcile([]model.ServerTunnel) error` and `StopAll()`; `backpack.NewSupervisor(binary, dir string) *engine.Supervisor` keeping today's behaviour; `rakhsh.NewSupervisor(binary, dir string) *engine.Supervisor` running `<binary> run -c <config>`; `rakhsh.Status{Connected bool; LanesUp, TargetLanes int; RTTMs *float64; LossPct float64; Peer string}`; `rakhsh.ReadStatus(ctx, binary, socket string) (Status, error)` (runs `status --socket S --json`); `rakhsh.Fetch/Place/Rollback/InstalledVersion/PinnedVersion` with `backpack`'s semantics and the asset names above.

- [ ] **Step 1: Write failing tests.** Move the existing supervisor tests to the `engine` package unchanged (they must stay green for `backpack.NewSupervisor`); add: a Rakhsh supervisor with a stand-in script restarts on config change; `ReadStatus` against a stand-in binary printing a captured real `status --json` sample (taken from `Snapshot` in `src/status.rs`) returns `Connected` only when `lanes_up > 0`; a missing binary returns `ErrNotInstalled`; installer tests against a local release server: good checksum installs, bad and missing checksum refuse, a failed download keeps the current binary.
- [ ] **Step 2: Run** `go test ./internal/tunnel/engine/ ./internal/tunnel/rakhsh/ ./internal/tunnel/backpack/`. Expected: FAIL (new symbols), existing backpack tests FAIL until wrappers exist.
- [ ] **Step 3: Implement** by moving code, not rewriting it; the status reader parses only the fields it needs and tolerates extra ones.
- [ ] **Step 4: Run** the three packages with `-race`. Expected: PASS.
- [ ] **Step 5: Commit** `refactor(tunnels): share one engine supervisor` then `feat(tunnels): add Rakhsh status and installer` (two commits).

### Task 5: Setup link version 4

**Files:**
- Modify: `internal/tunnel/setup.go`, `internal/tunnel/setup_test.go`

**Interfaces:**
- Consumes: Tasks 1–3. Produces: `PortalSpec.Engine = "rakhsh"` accepted; `NewPortal` generates both key pairs, token (UUID field), relay port, keeps `RakhshSecret`/`RakhshPeerPublic` for the portal; `SetupLink` emits version 4 for Rakhsh with the bridge secret and portal public key; `ParseSetupLink` returns a bridge with `RakhshSecret` = bridge secret and `RakhshPeerPublic` = portal public.

- [ ] **Step 1: Write failing tests.** Round trip for reverse and direct Rakhsh tunnels: the parsed bridge renders a valid config whose `peer_public` equals the portal's public key and whose `static_secret` is the bridge's, and the portal's `peer_public` equals the public key of the link's bridge secret. `TestRakhshLinkNeverCarriesPortalSecret`: decoded link JSON does not contain the portal secret. Rejections: Rakhsh with transport empty, unknown carrier, a Xray-only field set, version 4 with an Xray engine, version 9 still unknown.
- [ ] **Step 2: Run** `go test ./internal/tunnel/ -run 'Rakhsh|SetupLink'`. Expected: FAIL.
- [ ] **Step 3: Implement** minimal changes in `NewPortal`, `SetupLink`, `ParseSetupLink` (keep each function's existing structure; add a Rakhsh branch).
- [ ] **Step 4: Run** `go test ./internal/tunnel/...`. Expected: PASS.
- [ ] **Step 5: Commit** `feat(tunnels): carry Rakhsh keys in setup link version 4`.

### Task 6: Service, edit, node, controller wiring

**Files:**
- Modify: `internal/web/service/tunnel.go`, `tunnel_edit.go`, `tunnel_engine.go`, `tunnel_management.go`, `internal/web/controller/tunnel.go`, `internal/web/service/enrolment/enrolment.go` (if it special-cases BackPack), `internal/web/web.go` (stop path), `frontend/src/pages/api-docs/endpoints.ts`
- Tests: `tunnel_test.go`, `tunnel_edit_test.go`, `tunnel_management_test.go`, controller `tunnel_test.go`

**Interfaces:**
- Consumes: Tasks 1–5. Produces: `GET /panel/api/tunnels/options?engine=backpack|rakhsh` (default backpack, unchanged response for it); `TunnelService.EngineInfo()` returning one entry per external engine (`EngineInfo` gains `Engine string`; the list endpoint keeps the BackPack shape for compatibility and adds `engines`); `ReconcileEngines` reconciling both supervisors; `Status` using `rakhsh.ReadStatus`; `Update` validating via `rakhsh.RenderConfig`; `InstallEngine(engine)` and `UpdateEngine(ctx, nodeID, engine, version)` per engine.

- [ ] **Step 1: Write failing tests.** Create a Rakhsh portal through the controller and read it back: no `rakhshSecret` anywhere in the list JSON; options endpoint returns the Rakhsh catalog for `engine=rakhsh` and BackPack's for the default; status with a stand-in engine reports connected only with lanes up and an exact error when the binary is missing; editing the exit end keeps `RakhshSecret`/`RakhshPeerPublic`/token; editing the entry end pushes the new link to the fake node first and a refusing node changes nothing; deleting removes both ends; two Rakhsh tunnels get different control sockets; a Rakhsh tunnel with a relay port appears as outbound `tunnel-<name>` through `GetXrayConfig`.
- [ ] **Step 2: Run** `go test ./internal/web/... -run 'Rakhsh|TunnelOptions|EngineInfo'`. Expected: FAIL.
- [ ] **Step 3: Implement** by adding engine dispatch next to each existing BackPack branch; endpoints.ts entry for the changed query parameter; `make gen`, copy openapi, `pnpm gen:api`.
- [ ] **Step 4: Run** `go test ./internal/web/... ./internal/tunnel/...` and `make test-go`. Expected: PASS.
- [ ] **Step 5: Commit** `feat(tunnels): run, show, edit and route Rakhsh tunnels` (generated files in the same commit).

### Task 7: Frontend

**Files:**
- Modify: `frontend/src/pages/tunnels/TunnelsPage.tsx`, `TunnelEditModal.tsx`, `TunnelOptionFields.tsx`, `frontend/src/api/queries/useTunnels.ts`, `frontend/src/schemas/*` as needed
- Create: `frontend/src/pages/tunnels/TunnelLinksField.tsx`
- Tests: `frontend/src/test/tunnels-page.test.tsx`, new `rakhsh-tunnels.test.tsx`; translations in all 14 files

**Interfaces:**
- Consumes: Task 6 endpoints. Produces: engine selector option "Rakhsh"; option form driven by `useTunnelOptions(engine)`; `TunnelLinksField` (rows of carrier, address, per-link overrides; add/remove); engine install/update buttons per engine; a "needs root" hint on raw carriers.

- [ ] **Step 1: Write failing tests.** Choosing Rakhsh loads its options (request carries `engine=rakhsh`); required and raw-carrier hints show; adding two links submits them in order inside `options.links`; the edit modal shows the stored links and submits changes; switching engine clears options of the other engine; a Rakhsh row shows Reverse/Direct tag and an Edit button.
- [ ] **Step 2: Run** `cd frontend && npx vitest run --project components src/test/rakhsh-tunnels.test.tsx`. Expected: FAIL.
- [ ] **Step 3: Implement**; add keys with `/home/claude/tools/i18n.py add <key> <en> <fa>` and use them in the same commit.
- [ ] **Step 4: Run** `npm run lint`, `npx tsc --noEmit`, `npx vitest run --project unit`, `--project components`. Expected: PASS.
- [ ] **Step 5: Commit** `feat(tunnels): create and edit Rakhsh tunnels in the panel`.

### Task 8: Real binary test, docs, version

**Files:**
- Create: `internal/tunnel/rakhsh/e2e_test.go`
- Modify: `dalan/TUNNELS.md`, `dalan/HANDOFF.md`, `dalan/UPSTREAM.md`, `dalan/CONTAINER.md`, `internal/config/dalan_version`, `Dockerfile`/`DockerInit.sh` only if the image should bundle Rakhsh (it does not: installed on demand like BackPack)

- [ ] **Step 1: Write the gated e2e test** (`DALAN_RAKHSH_E2E_BINARY`): `rakhsh check -c` accepts rendered reverse and direct configs; a real pair on loopback carries a forwarded TCP byte stream over `tcp` and `wss`. Skips with a clear reason when unset.
- [ ] **Step 2: Run** it against the binary built in `/home/claude/cluvexstudio/rakhsh/target/release/rakhsh`. Expected: PASS; any rejected config key is fixed in the catalog or renderer, with a unit test first.
- [ ] **Step 3: Docs and version.** Document the engine, limits (raw carriers and layer 3 need root; containers; no loss-aware switching), the link-is-a-secret rule and the options; bump to 0.5.0.
- [ ] **Step 4: Run** `make verify` as far as the sandbox allows (Go tests, frontend lint/tsc/tests, gen-check).
- [ ] **Step 5: Commit** `docs(tunnels): document the Rakhsh engine; chore: bump Dalan to 0.5.0`; open the PR with `gh api repos/ParsaDa/Dalan/pulls`.
