# Node link on BackPack and Rakhsh, Xray tunnel engine removed — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans (inline) to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** A node joins with one command over a BackPack (default) or Rakhsh tunnel, and the Xray tunnel engine is gone.

**Architecture:** The node's own entry tunnel (engine chosen in the dialog) plus a management rider on it replaces the Xray management portal. The rider's loopback forward is already rendered by `engine.WithManagement`. Everything that exists only for `Engine == "xray"` is deleted; shared routing helpers move to a package named `tunnelroute`.

**Tech Stack:** Go 1.27, Gin, GORM, stdlib `testing`; React 19, Ant Design 6, Vitest.

**Spec:** `dalan/plans/2026-11-20-node-link-on-engines-design.md`

## Global Constraints

- Comments: 2 lines max per block; rename rather than annotate.
- TDD red→green; a test written after code is proven by reverting the code. No fake tests.
- Go tests: stdlib only, table-driven where it fits, real DB via `dbtest.InitDB`, assert exact values.
- New/renamed endpoint → `frontend/src/pages/api-docs/endpoints.ts`, `make gen`, copy `frontend/public/openapi.json` → `docs/public/openapi.json`, `cd docs && pnpm gen:api`. New response structs go in `tools/openapigen/main.go` `StructAllow`.
- New i18n key → all 14 files in `internal/web/translation/`, referenced in the same commit; run `npm test`.
- DB changes → migration in `internal/database/db.go`.
- State-changing ops go through `runtime.Runtime`; every `runtime.Remote` call keeps its cell in `internal/web/node_contract_test.go`.
- Conventional commits, body explains why; trailers as given by the session.
- Env: `source /home/claude/gometa/env.sh` in the same Bash call; Node 26 via `export PATH=/home/claude/.npm-global/bin:$PATH`; `make dist-stub` before `go build`; frontend format with `npx oxfmt`.

## Review Focus

- Users' traffic on a shared tunnel must never reach the node's panel port: the forward is loopback on both ends (Task 2 test).
- A node's old Xray channel vanishing silently: nodes enrolled over Xray show "enrol again" (Task 5, 6).
- Upgrading a database holding Xray tunnels, riders and pool members must not fail or leave dangling pool references (Task 5 test).
- A used, expired or version-1 link must stay refused (Task 2 tests).
- Deleting a node removes its tunnel and rider on both ends, as before (Task 1 test kept).

---

### Task 1: Enrolment creates the node's tunnel on BackPack or Rakhsh

**Files:**
- Modify: `internal/web/service/enrolment/enrolment.go` (Spec, Create, createTunnelWithChannel)
- Modify: `internal/web/service/enrolment/enrolment_test.go`

**Interfaces:**
- Produces: `Spec{Name, PanelURL, Address string; Port int; Engine, Transport string; Options map[string]any; TunnelId int}` (no `Security`, `ServerName`, `WithTunnel`). `Create` with `TunnelId == 0` always makes the node's entry tunnel named `<name>` via `TunnelService.CreatePortal(tunnel.PortalSpec{Name, Address, Port, Engine, Transport, Options}, nil)` and a rider `node-<name>` via `CreateManagementRider`; `Engine == ""` means BackPack and `Transport == ""` means `wssmux` (BackPack only). `TunnelId > 0` is unchanged.

- [ ] **Step 1: Write failing tests** in `enrolment_test.go`: `TestAnEnrolmentWithoutAnEngineMakesABackpackTunnelOnWssmux` (tunnel `de1` has `Engine == "backpack"`, `Transport == "wssmux"`, `RelayPort != 0`; rider `node-de1` has `ParentId == data.Id`, `ManagePort != 0`); `TestAnEnrolmentOnRakhshMakesARakhshTunnel` (`Engine == "rakhsh"`, `RakhshSecret != ""`); `TestAnEnrolmentRefusesAnUnknownEngine` (`Engine: "xray"` returns an error containing `tunnel engine "xray" is not supported`, and no tunnel row remains). Convert `TestANodeWithATunnelGetsAnOrdinaryTunnelThatItsManagementRidesOn` and `TestCancellingAPendingEnrolmentWithATunnelRemovesBothTunnels` to the new Spec (drop `WithTunnel`; spec helper `enrolmentSpec` loses Security/ServerName).
- [ ] **Step 2: Run** `go test ./internal/web/service/enrolment/ -run 'Enrolment' -count=1`. Expected: compile errors or FAIL on the new tests (fields missing).
- [ ] **Step 3: Implement** the Spec change and Create; default engine and transport resolved in Create before calling `CreatePortal`; delete the management-portal branch (`CreateManagementPortal` call) from Create only (the function itself goes in Task 3).
- [ ] **Step 4: Run** the same command. Expected: PASS for the new and converted tests; tests of the Xray management portal still in the file (`TestRedeemingAnEnrolmentRegistersTheNodeOnItsManagementPort`, `TestTwoEnrolmentsGet...`) are converted to BackPack the same way or deleted when they only restate the Xray port logic.
- [ ] **Step 5: Commit** `feat(enrolment): create a node's tunnel on BackPack or Rakhsh`.

### Task 2: Redeem grants the node its tunnel; version 2 link; loopback-only forward

**Files:**
- Modify: `internal/web/service/enrolment/enrolment.go` (Redeem, grant, Join)
- Modify: `internal/tunnel/enrol/enrol.go` (`linkVersion = 2`)
- Test: `internal/web/service/enrolment/enrolment_test.go`, `internal/tunnel/enrol/enrol_test.go`, `internal/tunnel/engine/management_test.go`

**Interfaces:**
- Consumes: Task 1's tunnel and rider. `enrol.Grant{Tunnel, DataTunnel string}` unchanged: `DataTunnel` is the node's bridge of its own tunnel; `Tunnel` stays empty for riders on external engines.

- [ ] **Step 1: Write failing tests:** (a) `enrol_test.go`: `TestAVersionOneLinkIsRefused` — a hand-built link with `"v":1` fails `ParseLink` with `enrolment link is version 1; this panel understands version 2`. (b) `management_test.go`: `TestManagementForwardIsLoopbackToLoopbackOnBackpackAndRakhsh` — for each engine, rendered config contains the forward `127.0.0.1:<managePort>=127.0.0.1:<nodePanelPort>` and no other listen address for it (table over both engines, asserting the exact forward string and that no `0.0.0.0` appears in it). (c) enrolment: `TestJoiningStoresTheNodesOwnTunnelAsItsBridge` — after Redeem on a fresh BackPack enrolment, `Join`'s import path stores a bridge named `<name>` with the portal's UUID; `TestARefusedJoinKeepsTheChannelToTheCurrentMainPanel` and `TestAnEnrolmentLinkWorksOnceAndOnlyWithItsToken` stay green.
- [ ] **Step 2: Run** `go test ./internal/tunnel/... ./internal/web/service/enrolment/ -count=1`. Expected: the version test FAILs (still 1); the others fail only where they depend on Xray.
- [ ] **Step 3: Implement:** bump `linkVersion`; in Redeem, when the enrolment has `DataTunnelId`, `grant.DataTunnel` is set (already so) and `grant.Tunnel` stays empty on external engines; in Join drop the `ImportManagementBridge` call (its branch is dead once Task 3 removes Xray) and import `grant.DataTunnel` via `s.tunnels.Import(grant.DataTunnel, "")`.
- [ ] **Step 4: Run** the same command. Expected: PASS.
- [ ] **Step 5: Commit** `feat(enrolment): hand the node its tunnel and require link version 2`.

### Task 3: Remove the Xray engine from tunnel creation and management

**Files:**
- Modify: `internal/database/model/server_tunnel.go` (drop `TunnelEngineXray`; gorm default `backpack`; fix doc comments)
- Modify: `internal/tunnel/setup.go` (drop `setupLinkXray` path, `validateXray`, REALITY key generation, `Security`/`ServerName` handling; `NewPortal` default engine BackPack; `ParseSetupLink` rejects version 1 with `tunnel link is version %d; this panel understands versions %d to %d`)
- Modify: `internal/web/service/tunnel_management.go` (delete `CreateManagementPortal`, `ImportManagementBridge`, `DeleteManagement` stays for riders, the `ManagePort`-on-bridge handling; riders no longer require Xray)
- Modify: `internal/web/service/tunnel.go`, `tunnel_pool.go`, `xray.go`, `engine_route.go`, `internal/web/controller/tunnel*.go` as the compiler and tests direct
- Delete/convert: Xray-only tests (`setup_test.go`, `tunnel_test.go`, `tunnel_rakhsh_test.go` parts, controller tests asserting Xray refusal)

**Interfaces:**
- Produces: `tunnel.PortalSpec` without `Security`/`ServerName`; `TunnelService.CreatePortal(spec, routeInbounds)` accepts engines `backpack` and `rakhsh` only (`tunnel engine "xray" is not supported`).

- [ ] **Step 1: Write failing tests:** `TestNewPortalRefusesTheXrayEngine` (error text exact), `TestAnEmptyEngineMeansBackpack`, `TestParseSetupLinkRefusesAnXrayVersionOneLink` (exact message). Run `go test ./internal/tunnel/ -count=1`; expected FAIL.
- [ ] **Step 2: Implement** the deletions listed; keep `make dist-stub && go build ./... && go vet ./...` green after each file group.
- [ ] **Step 3: Run** `make test-go` after the build is green. Expected: PASS with Xray-only tests deleted, none skipped to hide a failure; report any survivor that asserted Xray behaviour.
- [ ] **Step 4: Commit** `refactor(tunnel)!: remove the Xray tunnel engine` (body: why, what stops working, link to the design).

### Task 4: Move shared routing out of `xrayreverse`

**Files:**
- Rename: `internal/tunnel/xrayreverse/` → `internal/tunnel/tunnelroute/` (keep `Routable`, `PortalOutboundTag`, `ActiveMember`, `PoolBalancerTag`, `ApplyPools`, `EnableOutboundStats`, relay outbound/inbound generation); delete Xray portal/bridge generation, `ProbePort`, REALITY inbound/outbound settings, and `e2e_test.go` for the Xray engine
- Modify: importers (`internal/tor/inject.go`, `internal/web/service/*`, tests)

- [ ] **Step 1:** `git mv`, fix imports mechanically, run `go build ./...`; no behaviour change yet. Expected: build green.
- [ ] **Step 2: Write failing test** `TestApplyAddsRelayOutboundsForExternalEnginesAndNothingForXray`-style replaced by: `TestApplyWritesNoPortalOrBridgeForRemovedXrayRows` — a row with `Engine == "xray"` left in a database produces no inbound/outbound in the generated config (the migration in Task 5 deletes them; this pins Apply as safe if one slips through).
- [ ] **Step 3: Implement** the deletions; remove the Xray probe path in `engine_route.go` (lines using `ProbePort`).
- [ ] **Step 4: Run** `make test-go`. Expected: PASS.
- [ ] **Step 5: Commit** `refactor(tunnel): keep only the shared routing in tunnelroute`.

### Task 5: Migration and "enrol again" flag

**Files:**
- Modify: `internal/database/model/node_enrolment.go` (add `NeedsReenrol bool json:"needsReenrol" gorm:"column:needs_reenrol;default:false"`)
- Modify: `internal/database/db.go` (new `migrateRemoveXrayTunnels`, registered after AutoMigrate)
- Test: `internal/database/db_test.go` or the file where sibling migration tests live

**Interfaces:**
- Produces: `migrateRemoveXrayTunnels() error`: deletes every `server_tunnels` row with `engine = 'xray'` or empty engine, deletes riders (rows whose `parent_id` points at a deleted row), removes the deleted ids from pool membership (see `tunnel_pool.go` for the column or JSON), and sets `needs_reenrol = true` on enrolments whose `tunnel_id` or `data_tunnel_id` pointed at a deleted tunnel and whose `node_id > 0`. Idempotent.

- [ ] **Step 1: Write failing test** `TestMigratingRemovesXrayTunnelsRidersAndPoolMembersAndFlagsTheirNodes` using `database.InitDB` on a hand-built legacy DB: Xray portal + rider + a BackPack tunnel in a pool with the Xray one + an enrolment with a node; assert the exact survivors (BackPack tunnel stays, pool has only it), enrolment `NeedsReenrol == true`, and a second run changes nothing.
- [ ] **Step 2: Run** it; expected FAIL (migration absent).
- [ ] **Step 3: Implement** the migration; register it.
- [ ] **Step 4: Run** `go test ./internal/database/... -count=1`. Expected: PASS.
- [ ] **Step 5: Commit** `feat(database): remove Xray tunnels on upgrade and flag nodes that must enrol again`.

### Task 6: Frontend — enrolment dialog, Tunnels page, Nodes notice

**Files:**
- Modify: `frontend/src/pages/nodes/EnrolNode.tsx`, `NodeList.tsx`, `frontend/src/pages/tunnels/TunnelsPage.tsx`, `frontend/src/schemas/` as needed, `frontend/src/test/enrol-node.test.tsx`, `tunnels-page.test.tsx`, and other tests that name Xray tunnels
- Modify: 14 files in `internal/web/translation/`

**Interfaces:**
- Consumes: `POST /panel/api/enrolments/add` body `{name, panelUrl, address, port, engine, transport, tunnelId?}`; enrolment list rows carry `needsReenrol`.

- [ ] **Step 1: Write failing tests:** `enrol-node.test.tsx`: dialog offers Engine (default BackPack, Rakhsh), "How should it connect?" (new tunnel for this node / an existing tunnel listing entry tunnels, with a mark on the one users ride), posts exactly `{engine: 'backpack', transport: 'wssmux', …}` for the defaults; no "Server name (SNI)" field. `nodes` test: a node whose enrolment has `needsReenrol` shows the "must be enrolled again" notice and an "Enrol again" button opening the dialog prefilled with its name. Tunnels page test: no Xray engine option.
- [ ] **Step 2: Run** `cd frontend && npx vitest run --project=components`. Expected: FAIL for the new assertions.
- [ ] **Step 3: Implement** dialog and notice; add each new i18n key to all 14 locale files with the English text where no translation exists is NOT allowed to be dead: every key referenced; Persian (fa-IR) gets a real translation.
- [ ] **Step 4: Run** `npm test`, `npm run typecheck`, `npx oxfmt --check`. Expected: PASS.
- [ ] **Step 5: Commit** `feat(nodes): choose the engine and tunnel when enrolling a node`.

### Task 7: Contract, docs, e2e, release notes

**Files:**
- Modify: `frontend/src/pages/api-docs/endpoints.ts` (enrolment add body example), `tools/openapigen/main.go` if a new struct appears, generated outputs, `docs/public/openapi.json`, `docs/content/docs/en/reference/api/*`
- Modify: `dalan/TUNNELS.md`, `dalan/HANDOFF.md`, `design/2026-11-15-addendum-xray-reverse.md` (Project doc: mark superseded), `dalan/plans/2026-11-15-node-enrolment-design.md` (note: superseded in part by the design of 2026-11-20), release notes draft in `dalan/`
- Modify: `internal/nodee2e/enrolment_test.go` to enrol over BackPack (skips without `XUI_NODE_E2E_BINARY` plus the BackPack binary as it does for Xray today)

- [ ] **Step 1:** run `make gen`; copy openapi; `cd docs && pnpm gen:api`. Expected: no diff beyond intended.
- [ ] **Step 2:** update docs and the Project doc; write release-note lines (Xray tunnels stop working; enrolled-over-Xray nodes enrol again).
- [ ] **Step 3:** run `make verify`; expected PASS (docs-openapi-check included). Run `make node-e2e` if the binaries exist locally, else say CI covers it.
- [ ] **Step 4: Commit** `docs: node link on engines, Xray tunnels removed` and push the branch.
