# Tor outbound (Dalan 0.5.1) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a per-country Tor outbound: Dalan runs one managed Tor per enabled country on a loopback SOCKS port, injects it into Xray as `tor-<cc>`, shows live status, and offers a per-country Test button.

**Architecture:** New `internal/tor` package (torrc rendering, bundle install, status, throwaway test run, Xray injection) on top of `engine.Supervisor`. New `TorExit` model and `TorService`. New `/panel/api/tor/*` controller. Frontend `TorModal` beside Warp/Nord/PIA, Tor rows in the Outbounds table, Tor tags in routing pickers.

**Tech Stack:** Go 1.27, Gin, GORM, Tor Expert Bundle 16.0.1 (tor 0.4.9.16); React 19, Ant Design 6, TypeScript, Zod.

**Spec:** `docs/superpowers/specs/2026-11-17-tor-outbound-design.md`

## Global Constraints

- Only Persian and English strings are real; the other 12 locales carry the English text. Every new key goes in all 14 files in `internal/web/translation/` via `/home/claude/tools/i18n.py add <key> <en> <fa>` and is used in the same commit.
- Comment blocks in Go/TS: 2 lines max. Name things so they need no comment.
- Country names in the UI are English only, via `countryFlag` and `countryName(code, 'en')`.
- Tor SOCKS binds `127.0.0.1` only. Never `0.0.0.0`.
- Tor exit ports come from 62700-62789. Maximum enabled exits: 10 (`tor.MaxExits`).
- Pinned bundle: `tor.PinnedVersion = "16.0.1"`; archive URL `https://dist.torproject.org/torbrowser/16.0.1/tor-expert-bundle-linux-<x86_64|aarch64|i686>-16.0.1.tar.gz`; checksums from `sha256sums-signed-build.txt` in the same directory. The environment cannot reach that host, so URLs and layout are unverified: ledger it and say so in the PR.
- New `g.POST`/`g.GET` in `internal/web/controller/` needs an `endpoints.ts` entry, `make gen`, copy `frontend/public/openapi.json` to `docs/public/openapi.json`, `cd docs && pnpm gen:api`. New response structs go in openapigen `StructAllow` and carry `example:` tags.
- DB change: add `&model.TorExit{}` to the AutoMigrate list in `internal/database/db.go` and to `migrationModels()` in `internal/database/migrate_data.go`.
- Go: stdlib `testing` only, table-driven, real deps over mocks, `dbtest.InitDB(t, filepath.Join(t.TempDir(), "x-ui.db"))`. gofumpt formatting; no raw zero-width characters in string literals. Every test must name the failure it catches and be mutation-checked (revert the code, watch it fail).
- State-changing inbound/client ops stay on `runtime.Runtime`; Tor exits are panel-local like Nord/PIA and are not nodes' concern.
- Commits: `type(area): summary`, body with the why, ending with `Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>` and `Claude-Session: https://claude.ai/code/session_01YJ4rUKMzFM9JvtXJLedwnK`. Branch `feat/dalan-0.5.1`; version file `internal/config/dalan_version` becomes `0.5.1` (no trailing newline).
- Tools: `source /home/claude/gometa/env.sh`; `export PATH=/home/claude/.npm-global/bin:$PATH`; `make dist-stub` once. golangci-lint and gofumpt cannot run here; CI runs them.

## Review Focus

- Deleting an exit that a routing rule or balancer still names: refuse with a clear error (like a tunnel in a pool), never leave a dangling `tor-` tag.
- Tor binary not installed: Xray still starts, the exit shows "not installed", nothing crashes or loops restarting.
- Two Test clicks at once: the second answers "busy"; no Tor process or temp directory outlives either call.
- Panel restart: enabled exits come back by themselves, and stop on shutdown.
- Hostile bridge text (embedded newline, `\r`, NUL, an extra directive like `ExitPolicy accept *:*`): rejected or rendered only as a `Bridge` line, never as another torrc directive.
- An inbound or tunnel already on a port in 62700-62789: the exit skips it; an exit port is never reused by two exits.

---

### Task 1: `TorExit` model, `TorService`, bridges setting

**Files:**

- Create: `internal/database/model/tor_exit.go`, `internal/tor/countries.go`, `internal/tor/countries_test.go`, `internal/web/service/tor.go`, `internal/web/service/tor_test.go`
- Modify: `internal/database/db.go` (AutoMigrate list), `internal/database/migrate_data.go` (`migrationModels`), `internal/web/service/setting.go` (new string setting `torBridges`, default empty)

**Interfaces:**

- Produces: `model.TorExit{Id int; Country string (unique, lower-case, json "country"); Port int (json "port"); Enable bool (json "enable", default true)}`; `tor.NormalizeCountry(string) (string, bool)`; `tor.MaxExits = 10`; `tor.PortBase = 62700`, `tor.PortLimit = 62789`; `tor.ValidateBridges(text string) ([]string, error)` (trimmed non-empty lines); `(*TorService).List() ([]model.TorExit, error)`, `Add(country string) (*model.TorExit, error)`, `SetEnable(id int, enable bool) error`, `Delete(id int) error`, `Bridges() (string, error)`, `SetBridges(text string) error`. Each of Add, SetEnable, Delete, SetBridges ends by calling `s.applyChange()` (defined in Task 3; until then a no-op stub the task adds and Task 3 replaces).

- [ ] **Step 1: Write the failing tests** in `countries_test.go`: `TestNormalizeCountry` (`"DE"`→`"de"`, `" nl "`→`"nl"`; `""`, `"d"`, `"deu"`, `"d1"`, `"zz-"` rejected); `TestValidateBridges` (valid `obfs4 1.2.3.4:443 FINGERPRINT cert=abc iat-mode=0` kept; a line containing `\r`, NUL, or whose first token is not `obfs4`, `snowflake`, `webtunnel`, `meek_lite` or an `IP:port` is an error that names the line). In `tor_test.go` with `dbtest.InitDB`: `TestAddAssignsPortsInRange` (first exit gets 62700, second 62701; a tunnel RelayPort or inbound port equal to 62700 makes the next exit take the following free port), `TestAddRefusesDuplicateCountry`, `TestAddRefusesEleventhEnabledExit`, `TestDeleteRefusedWhileRoutingNamesTag` (xray template stored with a routing rule whose `outboundTag` is `tor-de` and, separately, a balancer selector containing it: both refuse; with neither, delete succeeds and removes the row).
- [ ] **Step 2: Run** `go test ./internal/tor ./internal/web/service -run 'Country|Bridges|TorService|Tor'` — Expected: FAIL (undefined symbols).
- [ ] **Step 3: Implement** the signatures above. Port choice: lowest free port in range not used by any `model.TorExit`, `model.Inbound` (node_id NULL) or `ServerTunnel.relay_port`. The in-use check reads the stored Xray template setting (same source `SaveXraySetting` writes) and looks at `routing.rules[].outboundTag` and `routing.balancers[].selector`.
- [ ] **Step 4: Run the same command** — Expected: PASS. Mutation-check: change the cap to 11, the range base, and drop the in-use check once each; each must turn a named test red.
- [ ] **Step 5: Commit** `feat(tor): TorExit model, exit service and bridge validation`.

### Task 2: torrc rendering, supervisor, status from log lines

**Files:**

- Create: `internal/tor/config.go`, `config_test.go`, `supervisor.go`, `supervisor_test.go`, `progress.go`, `progress_test.go`
- Modify: `internal/tunnel/engine/supervisor.go` (two optional fields), `internal/tunnel/engine/supervisor_test.go` if present (add a case)

**Interfaces:**

- Consumes: `model.TorExit`, `tor.ValidateBridges` (Task 1).
- Produces: `tor.RenderConfig(exit model.TorExit, bridges []string, dir Layout) (string, error)` where `Layout{Root, BundleDir string}` gives `DataDirectory`, `GeoIPFile`, transport paths; `tor.NewSupervisor(binary, dir string, bridges func() []string, progress *Progress) *engine.Supervisor`; `tor.Progress` with `Set(country string, percent int)`, `Get(country string) (int, bool)`, `Reset(country string)`; `tor.ParseBootstrap(line string) (percent int, ok bool)`. `engine.Supervisor` gains `Ext string` (config file suffix, default `.toml`) and `Env []string` (extra environment, default none) so Tor can use `.torrc` and `LD_LIBRARY_PATH`.

- [ ] **Step 1: Failing tests.** `TestRenderConfig`: exact output contains `SocksPort 127.0.0.1:62700`, `ExitNodes {de}`, `StrictNodes 1`, `ClientOnly 1`, `Log notice stdout`, a `DataDirectory` under the layout root named for the country, and `GeoIPFile`/`GeoIP6File`; with no bridges no `UseBridges` line; with `obfs4 …` and `snowflake …` lines: `UseBridges 1`, one `Bridge …` line each, and `ClientTransportPlugin obfs4 exec <bundle>/pluggable_transports/lyrebird` (and snowflake, using the bundle's snowflake client path). `TestRenderConfigNeverBindsBeyondLoopback` over several ports. `TestParseBootstrap` with real log text (`Bootstrapped 45% (loading_descriptors): Loading relay descriptors`, `Bootstrapped 100% (done): Done`, and non-matching lines). Engine: a case showing `Ext` changes the config file name and `Env` reaches the child process (use the existing fake-engine style in the engine tests; a tiny shell script that writes `$LD_LIBRARY_PATH` to a file).
- [ ] **Step 2: Run** `go test ./internal/tor ./internal/tunnel/engine` — Expected: FAIL.
- [ ] **Step 3: Implement.** Supervisor adapter: each exit becomes a `model.ServerTunnel{Name: "tor-"+cc, Port: exit.Port, Enable: exit.Enable, Engine: "tor"}` so `Wants` matches `Engine == "tor"`; `Render` calls `RenderConfig`; `Args` is `-f <config>`; `LogLine` feeds `ParseBootstrap` into `Progress` and the panel log; `Leftovers` is empty. Bundle layout (`tor/tor`, `tor/pluggable_transports/…`, `data/geoip`, `data/geoip6`) is the best knowledge available and is unverified here: isolate every bundle path in `Layout` so one place changes if the real archive differs.
- [ ] **Step 4: Run** the same command — Expected: PASS; mutation-check `127.0.0.1`→`0.0.0.0` and drop `StrictNodes`.
- [ ] **Step 5: Commit** `feat(tor): render torrc and supervise one Tor per exit`.

### Task 3: bundle install, engine info, reconcile at startup and shutdown

**Files:**

- Create: `internal/tor/install.go`, `install_test.go`
- Modify: `internal/web/service/tor.go` (add `EngineInfo`, `InstallEngine`, `RollbackEngine`, `ReconcileExits`, `StopExits`, real `applyChange`), `internal/web/web.go` (call `ReconcileExits` where `ReconcileEngines` runs at line ~332 and `StopExits` beside `StopEngines` at ~818)

**Interfaces:**

- Consumes: Task 1 service, Task 2 `NewSupervisor`.
- Produces: `tor.PinnedVersion = "16.0.1"`; `tor.Fetch(ctx, client *http.Client, baseURL, version, arch string) (engine.Release, error)`; `tor.Place(rel engine.Release, version, arch, destDir string) error` (checks `sha256sums-signed-build.txt` entry for the asset, extracts the tar.gz into `destDir`, keeps the previous directory for `Rollback`); `tor.Rollback(destDir string) error`; `tor.InstalledVersion(destDir string) string`; `(*TorService).EngineInfo() (EngineInfo, error)` reusing the existing `EngineInfo` struct with `Engine: "tor"`; `(*TorService).InstallEngine() error`; `RollbackEngine() error`.

- [ ] **Step 1: Failing tests** with an `httptest` server serving a small real tar.gz built in the test plus its sums: `TestPlaceExtractsAndKeepsPrevious`, `TestPlaceRefusesWrongChecksum` (old bundle stays, error names the asset), `TestPlaceRefusesPathTraversalInArchive` (an entry `../evil` is rejected), `TestAssetNameForArch` (amd64→`x86_64`, arm64→`aarch64`, 386→`i686`, others error "not available for this processor"), `TestRollbackRestoresPrevious`.
- [ ] **Step 2: Run** `go test ./internal/tor -run 'Place|Asset|Rollback'` — Expected: FAIL.
- [ ] **Step 3: Implement** using `engine.Download`, `engine.CheckChecksum`, and version bookkeeping modelled on `engine.Replace`/`Rollback` but for a directory. `InstallEngine` downloads through `SettingService.NewProxiedHTTPClient(engineInstallTimeout)` like `InstallEngine` for tunnels, then `ReconcileExits`. `ReconcileExits` loads exits and calls `Supervisor.Reconcile`; a missing binary returns `tor.ErrNotInstalled` and is logged, never fatal. `applyChange` = `xrayService.SetToNeedRestart()` then `ReconcileExits`.
- [ ] **Step 4: Run** — Expected: PASS; mutation-check the traversal guard and the checksum comparison.
- [ ] **Step 5: Commit** `feat(tor): install the pinned Tor bundle and bring exits up with the panel`.

### Task 4: Xray injection and status

**Files:**

- Create: `internal/tor/inject.go`, `inject_test.go`, `status.go`, `status_test.go`
- Modify: `internal/web/service/xray.go` (call `tor.Apply` right after the tunnel injection block, ~line 505), `internal/web/service/tor.go` (`Outbounds`, `StatusAll`), `internal/web/controller/xray_setting.go` (add `torOutbounds` to the response next to `tunnelOutbounds`)

**Interfaces:**

- Consumes: `model.TorExit`, `Progress`, `tunnel.ProbeVia`, `tunnel.DefaultProbeURL`.
- Produces: `tor.OutboundTag(country string) string` = `"tor-"+country`; `tor.Apply(cfg *xray.Config, exits []model.TorExit) error` (appends `{"tag":"tor-<cc>","protocol":"socks","settings":{"servers":[{"address":"127.0.0.1","port":N}]}}` for each enabled exit; never touches the stored template); `tor.Status{State string; Progress int; LatencyMs int64; ExitIP, ExitLoc, Error string}` with `State` one of `not_installed`, `stopped`, `starting`, `connected`, `failed`; `var probeVia = tunnel.ProbeVia` (replaceable in tests); `(*TorService).StatusAll() (map[string]tor.Status, error)` (probes connected-looking exits concurrently with an 8 s timeout, caches each result 30 s); `service.TorOutbound{Tag, Country string; Enable bool}` and `(*TorService).Outbounds() ([]TorOutbound, error)`.

- [ ] **Step 1: Failing tests.** `TestApplyInjectsEnabledExitsOnly` (two exits, one disabled: the config has exactly one `tor-<cc>` outbound with the right port; existing outbounds are kept; an unparsable outbounds section returns an error). `TestStatusStates` through the `probeVia` seam: no binary → `not_installed`; process absent → `stopped`; progress 40 → `starting` with `Progress: 40`; progress 100 and a successful probe → `connected` with latency and `ExitLoc`; progress 100 and a failed probe → `failed` carrying the probe error. `TestStatusIsCachedFor30Seconds` (second call within the window does not probe again; a clock seam is allowed).
- [ ] **Step 2: Run** `go test ./internal/tor ./internal/web/service -run 'Apply|Status'` — Expected: FAIL.
- [ ] **Step 3: Implement.** Probe address `127.0.0.1:<port>`. In `xray_setting.go` add `xrayResponse["torOutbounds"]` when non-empty.
- [ ] **Step 4: Run** — Expected: PASS; mutation-check the enabled filter and the "connected only after a successful probe" rule.
- [ ] **Step 5: Commit** `feat(tor): inject Tor exits into Xray and report their status`.

### Task 5: the per-country Test

**Files:**

- Create: `internal/tor/trial.go`, `trial_test.go`
- Modify: `internal/web/service/tor.go` (`Test`)

**Interfaces:**

- Consumes: `tor.RenderConfig`, `tor.ParseBootstrap`, `probeVia`, installed binary path.
- Produces: `tor.TrialResult{OK bool; LatencyMs int64; ExitIP, ExitLoc, Reason, Error string}`; `tor.ErrTrialBusy`; `tor.Trial(ctx context.Context, binary string, country string, bridges []string, layout Layout) (TrialResult, error)`; `(*TorService).Test(country string) (tor.TrialResult, error)` (timeout 90 s; if that country already runs, return its live status mapped into `TrialResult` instead of starting a second Tor).

- [ ] **Step 1: Failing tests** using a fake `tor` shell script written to `t.TempDir()` (prints `Bootstrapped 100% (done): Done` and sleeps) and the `probeVia` seam: `TestTrialSucceedsAndCleansUp` (result OK; afterwards the process is gone and the temp directory does not exist), `TestTrialReportsNoExitRelays` (fake prints `Failed to find node for hop #1 of our path` / probe fails: `Reason` is a plain "no exit relays reachable in this country" and `OK` false), `TestTrialTimesOutAndCleansUp` (script never bootstraps; short injected timeout), `TestSecondTrialIsBusy` (two concurrent calls: exactly one returns `ErrTrialBusy`, no stray process).
- [ ] **Step 2: Run** `go test ./internal/tor -run Trial` — Expected: FAIL.
- [ ] **Step 3: Implement** with a package mutex (`TryLock`), `os.MkdirTemp`, a free port from `net.Listen(":0")` on loopback closed before use, `exec.CommandContext` with SIGTERM then kill, and `defer os.RemoveAll`. Reuse the supervisor's rendered torrc via `RenderConfig`. Map failure text to `Reason` in one small function.
- [ ] **Step 4: Run** — Expected: PASS; mutation-check removing the deferred cleanup and the busy lock.
- [ ] **Step 5: Commit** `feat(tor): test a country by running a throwaway Tor`.

### Task 6: controller, API docs, real-binary e2e

**Files:**

- Create: `internal/web/controller/tor.go`, `internal/tor/e2e_test.go`
- Modify: `internal/web/controller/` router registration (follow how `tunnel.go` is mounted), `frontend/src/pages/api-docs/endpoints.ts`, `tools/openapigen/main.go` (`StructAllow`), `frontend/public/openapi.json` and `docs/public/openapi.json` (generated/copied), `docs/content/docs/en/reference/api/` (generated)

**Interfaces:**

- Consumes: Task 1, 3, 4, 5 service methods.
- Produces (all under `/panel/api/tor`): `GET /list` → `[]{id, country, port, enable, status tor.Status, tag}`; `POST /add` (form `country`); `POST /del/:id`; `POST /setEnable/:id` (form `enable`); `POST /test` (form `country`) → `tor.TrialResult`; `GET /engine` → `EngineInfo`; `POST /engine/install`; `POST /engine/rollback`; `GET /bridges` → `{bridges string}`; `POST /bridges` (form `bridges`). Response messages use new i18n keys under `pages.tor.toasts.*`.

- [ ] **Step 1: Failing tests.** `TestRouteRegistryContract` must go red first by adding the routes to the controller without `endpoints.ts` entries (a missing entry fails it). Add `internal/web/controller/tor_test.go` with httptest through the real router: add → list shows the exit with `tag:"tor-de"`; add of `"xx1"` is a 200 with `success:false`; bridges round-trip; a bad bridge returns `success:false`. `e2e_test.go` runs a real Tor only when `TOR_E2E_BINARY` is set (skips otherwise) and asserts `Trial` for one country returns OK.
- [ ] **Step 2: Run** `make test-go` — Expected: the contract test FAILS until `endpoints.ts` is updated.
- [ ] **Step 3: Implement** the controller (thin: bind form, call service, `jsonObj`/`jsonMsg`), add the `endpoints.ts` entries with accurate summaries, add the response structs to `StructAllow`, then `make gen`, copy `openapi.json` to `docs/public/`, `cd docs && pnpm gen:api`, `oxfmt` the generated docs if CI formatting requires it.
- [ ] **Step 4: Run** `make test-go` and `make gen` then `git status` — Expected: PASS and no unexpected diff after a second `make gen`.
- [ ] **Step 5: Commit** `feat(tor): panel API for Tor exits, bridges and the engine`.

### Task 7: frontend data layer and Outbounds rows

**Files:**

- Create: `frontend/src/api/queries/useTor.ts`, `frontend/src/pages/xray/outbounds/torRows.ts` (only if the row mapping needs a helper)
- Modify: `frontend/src/schemas/xray.ts` (add `TorOutboundSchema` and `torOutbounds` to the response schema), `frontend/src/hooks/useXraySetting.ts` (expose `torOutbounds`), `frontend/src/lib/xray/injected-outbound-tags.ts` (accept `torOutbounds`), `frontend/src/pages/xray/XrayPage.tsx`, `frontend/src/pages/xray/outbounds/OutboundsTab.tsx`, `outbounds-tab-types.ts`, `useOutboundColumns.tsx`, `OutboundCardList.tsx`
- Test: `frontend/src/test/outbounds-tor-rows.test.tsx`, extend `use-outbound-tags.test.tsx`

**Interfaces:**

- Produces: `useTorExits()` (React Query over `/panel/api/tor/list`, refetch every 15 s while the Xray page is open), `useTorAdd/Delete/SetEnable/Test/Bridges/EngineInstall` mutations; `TorOutbound = {tag: string; country: string; enable: boolean}`; `injectedOutboundTags({subscriptionOutboundTags, tunnelOutbounds, torOutbounds})`; `OutboundsTab` gains props `torOutbounds?: TorOutbound[]`, `torStatus?: Record<string, {state: string; latencyMs: number; error: string}>` keyed by tag, and `onShowTor: () => void`.

- [ ] **Step 1: Failing tests.** `outbounds-tor-rows.test.tsx` modelled on `outbounds-tunnel-rows.test.tsx`: a Tor row shows tag `tor-de`, the German flag, a status chip reading connected with latency, one reading not connected with its error, and one reading starting; it has no Edit and no delete; a Manage action calls `onShowTor`; the card list shows the same on mobile. `use-outbound-tags` test: tor tags appear in the tag list.
- [ ] **Step 2: Run** `cd frontend && npx vitest run --project components outbounds-tor-rows use-outbound-tags` — Expected: FAIL.
- [ ] **Step 3: Implement** following the tunnel-row pattern exactly (rows appended after stored outbounds, protocol `'tor'`, row index unaffected). Zod schema is the source of truth; no hand-written types.
- [ ] **Step 4: Run** the same command and `npx tsc --noEmit` — Expected: PASS; mutation-check the status chip mapping.
- [ ] **Step 5: Commit** `feat(tor): list Tor exits in the Outbounds table and routing pickers`.

### Task 8: TorModal and wiring

**Files:**

- Create: `frontend/src/pages/xray/overrides/TorModal.tsx`, `TorModal.css`, `frontend/src/pages/xray/overrides/torCountries.ts`
- Modify: `frontend/src/pages/xray/overrides/index.ts`, `XrayPage.tsx` (state `torOpen`, render `TorModal`, pass `onShowTor`), `OutboundsTab.tsx` (a Tor entry in the same menu as Warp/Nord/PIA, ~line 600), translation files via `i18n.py`
- Test: `frontend/src/test/tor-modal.test.tsx`

**Interfaces:**

- Consumes: Task 7 hooks and `countryFlag`/`countryName`.
- Produces: `TOR_COUNTRIES: readonly string[]` in `torCountries.ts`: every ISO 3166-1 alpha-2 code Tor's geoip knows, lower-case, no duplicates (use the full assigned ISO list); `TorModal({open, onClose})`.

- [ ] **Step 1: Failing tests.** `torCountries` test: no duplicates, every entry matches `/^[a-z]{2}$/`, every entry has an English name from `countryName(code,'en')` that is not the bare code (catches typos like `uk` for `gb`), and the list has at least 200 entries. `tor-modal.test.tsx` with mocked HTTP: list renders flag plus English name and no Persian text; typing in the search box filters; Test shows a spinner on that row only and then a green result with latency, or a red result with the reason; Test of a second row while one runs is disabled; Add posts the country and the row then shows Remove; adding at the cap shows the server's error; the engine card shows "not installed" with an Install button that posts `/engine/install`; the bridges box (under Advanced, collapsed by default) saves and shows the server's validation error; a notice about TCP-only and about exits reading unencrypted traffic is present.
- [ ] **Step 2: Run** `cd frontend && npx vitest run --project components tor-modal torCountries` — Expected: FAIL.
- [ ] **Step 3: Implement** the modal following `PiaModal.tsx` (react-hook-form only where a form is needed; plain state is fine for the list). Add all new keys with `i18n.py add` (Persian text real, written for non-technical readers).
- [ ] **Step 4: Run** `npm test -- --run` for the frontend and `npx tsc --noEmit`, `npm run lint` — Expected: PASS including the i18n dead-key test.
- [ ] **Step 5: Commit** `feat(tor): Tor modal with country picker, per-country test and bridges`.

### Task 9: docs, version, full gate

**Files:**

- Create: `dalan/TOR.md`
- Modify: `internal/config/dalan_version` (`0.5.1`, no trailing newline), `dalan/HANDOFF.md`, `dalan/UPSTREAM.md` if it lists features, `docs/superpowers/plans/` formatting via `oxfmt`

- [ ] **Step 1:** Write `dalan/TOR.md`: what it does, the one-Tor-per-country model, ports, bridges, TCP-only, panel-local, the unverified-bundle caveat and how to place a bundle by hand.
- [ ] **Step 2:** Bump the version; update `HANDOFF.md`.
- [ ] **Step 3: Run** `make verify` (record each stage's result; golangci-lint and storybook may be unavailable here: say so) and `go test -race ./internal/tor ./internal/tunnel/engine`. Expected: green, or the exact unavailable stages named.
- [ ] **Step 4: Commit** `chore(release): Dalan 0.5.1`.
