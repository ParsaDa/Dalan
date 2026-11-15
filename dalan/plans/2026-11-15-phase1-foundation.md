# Dalan Phase 1 — Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn the `ParsaDa/dalan` fork of 3x-ui into a separately named, separately released panel that installs and self-updates from its own repository and can keep merging upstream 3x-ui.

**Architecture:** Branding is a small, reproducible layer on top of untouched upstream code. Go code reads the repository slug and display name from one new file; shell scripts and workflows are rebranded by an idempotent script that is re-run after every upstream merge; a check script fails the build if an upstream URL survives where it would send a Dalan install back to stock 3x-ui.

**Tech Stack:** Go 1.27, Bash, GitHub Actions, React 19 + Ant Design 6 (two brand strings only).

**Spec:** Project docs `design/2026-11-15-panel-design.md` (§5, §7.9, §8 item 1) and `design/2026-11-15-addendum-xray-reverse.md`. These live in the claude.ai Project "VPN and Networking", not in this repo.

**Out of this plan:** tunnels, node enrolment, backup server, container image, UI redesign. Each gets its own plan (spec §8 items 2–6).

## Global Constraints

- Follow `CLAUDE.md` in this repo: conventional commits, comments ≤ 2 lines, TDD with tests that can fail, `make verify` green before the PR.
- Do NOT rename: the Go module path `github.com/mhsanaei/3x-ui/v3`, the binary `x-ui`, `/etc/x-ui`, `/usr/local/x-ui`, `XUI_*` env vars, systemd unit names, the `x-ui` CLI command. Reason: every rename is a permanent merge conflict with upstream, and unchanged paths let an existing 3x-ui server switch to Dalan in place.
- Do NOT change `internal/config/version` (upstream's file; it is the "based on 3x-ui X.Y.Z" value).
- Repository slug: `ParsaDa/dalan`. Display name: `Dalan`. Container image: `ghcr.io/parsada/dalan`. First version: `0.1.0`, tag `v0.1.0`.
- `mhsanaei/mtg-multi` URLs stay as they are (a dependency, not this project).
- New Dalan-only files live under `dalan/` (docs, plans) and `scripts/dalan/` (scripts), so they never collide with upstream paths.
- All work on branch `claude/dalan-foundation`, one PR into `main`.
- GPL-3.0 stays; README must credit 3x-ui and its author.

## Review Focus

1. **An existing 3x-ui server runs the Dalan installer.** Expected: database and settings in `/etc/x-ui` are kept. Covered by the no-rename constraint; Task 3 asserts the path strings are untouched by the rebrand.
2. **The panel's "Update" button on a Dalan server.** Expected: it downloads Dalan, never stock 3x-ui. Task 1 tests the exact URLs.
3. **`rebrand.sh` is run twice, or on an already-rebranded tree.** Expected: no change the second time. Task 3 tests it.
4. **An upstream merge brings a new script or workflow containing `MHSanaei/3x-ui`.** Expected: the build fails and names the file. Task 3's check globs rather than using a fixed list, and its test adds such a file.
5. **Latest release tag is `v0.1.1` while the panel is `0.1.0`, or equal, or the release list is empty.** Expected: update offered / not offered / a clear error, never a crash. Task 2 tests all three through the real `isNewerVersion` path.

---

### Task 1: Brand constants and updater URLs

**Files:**
- Create: `internal/config/brand.go`
- Modify: `internal/web/service/panel/panel.go` (const `panelUpdaterURL` at ~line 44; `fetchPanelRelease` at ~line 439)
- Modify: `internal/web/service/discord/discord.go:23` (user agent)
- Test: `internal/web/service/panel/panel_brand_test.go`

**Interfaces:**
- Produces: `config.RepoSlug` (`"ParsaDa/dalan"`), `config.DisplayName` (`"Dalan"`), `func panelUpdaterScriptURL() string`, `func panelReleaseAPIURL(tag string) string` in package `panel`.

- [ ] **Step 1: Write the failing test**

```go
func TestPanelReleaseAPIURL(t *testing.T) {
	cases := []struct{ tag, want string }{
		{"", "https://api.github.com/repos/ParsaDa/dalan/releases/latest"},
		{"dev-latest", "https://api.github.com/repos/ParsaDa/dalan/releases/tags/dev-latest"},
	}
	for _, c := range cases {
		if got := panelReleaseAPIURL(c.tag); got != c.want {
			t.Errorf("panelReleaseAPIURL(%q) = %q, want %q", c.tag, got, c.want)
		}
	}
}

func TestPanelUpdaterScriptURL(t *testing.T) {
	want := "https://raw.githubusercontent.com/ParsaDa/dalan/main/update.sh"
	if got := panelUpdaterScriptURL(); got != want {
		t.Errorf("panelUpdaterScriptURL() = %q, want %q", got, want)
	}
}
```

Failure it catches: the update button installing stock 3x-ui over Dalan.

- [ ] **Step 2: Run to verify it fails**

Run: `make dist-stub && go test ./internal/web/service/panel/ -run 'TestPanel(ReleaseAPI|UpdaterScript)URL'`
Expected: build failure, `undefined: panelReleaseAPIURL`.

- [ ] **Step 3: Implement**

`internal/config/brand.go`: package `config`, exported constants `RepoSlug` and `DisplayName` with the values above. In `panel.go`, remove the `panelUpdaterURL` constant, add the two functions building URLs from `config.RepoSlug`, and use them at every former use of the constant and inside `fetchPanelRelease`. In `discord.go`, build the user-agent URL from `config.RepoSlug`.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/web/service/panel/ ./internal/web/service/discord/`
Expected: PASS.

- [ ] **Step 5: Commit**

`feat(brand): read repository slug from one place and point the updater at Dalan`

---

### Task 2: Dalan version separate from the upstream base version

**Files:**
- Create: `internal/config/dalan_version` (content: `0.1.0`)
- Modify: `internal/config/config.go` (embed the new file; `GetBaseVersion`)
- Modify: `internal/web/service/panel/panel.go` only if `isNewerVersion` needs no change — expected none
- Test: `internal/config/config_test.go`, `internal/web/service/panel/panel_brand_test.go`

**Interfaces:**
- Produces: `config.GetBaseVersion()` now returns the Dalan version (`"0.1.0"`); new `config.GetUpstreamVersion() string` returns the content of `internal/config/version`.

- [ ] **Step 1: Write the failing tests**

```go
// config_test.go
func TestVersionsAreSeparate(t *testing.T) {
	if got := GetBaseVersion(); got != "0.1.0" {
		t.Errorf("GetBaseVersion() = %q, want %q", got, "0.1.0")
	}
	if got := GetUpstreamVersion(); got != strings.TrimSpace(version) || got == GetBaseVersion() {
		t.Errorf("GetUpstreamVersion() = %q, must be the upstream file and differ from the Dalan version", got)
	}
}

// panel_brand_test.go
func TestUpdateOfferedOnlyForNewerDalanRelease(t *testing.T) {
	cases := []struct {
		latest string
		want   bool
	}{{"v0.1.1", true}, {"v0.1.0", false}, {"v0.0.9", false}}
	for _, c := range cases {
		if got := isNewerVersion(c.latest, config.GetBaseVersion()); got != c.want {
			t.Errorf("isNewerVersion(%q, %q) = %v, want %v", c.latest, config.GetBaseVersion(), got, c.want)
		}
	}
}
```

Also add a test that drives `fetchPanelRelease`'s caller with an `httptest` server returning `404` and asserts a non-nil error whose text contains `404`. This needs the API host to be replaceable: introduce an unexported package variable `githubAPIBase = "https://api.github.com"` used by `panelReleaseAPIURL`, and set it in the test with `t.Cleanup` restoring it.

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./internal/config/ ./internal/web/service/panel/`
Expected: FAIL — `GetBaseVersion() = "3.11.0", want "0.1.0"` and `undefined: GetUpstreamVersion`.

- [ ] **Step 3: Implement**

Embed `dalan_version` next to the existing `version` embed. `GetBaseVersion` trims and returns it; `GetUpstreamVersion` returns the trimmed upstream value. Fix any existing test in `config_test.go` that pinned the old value by reading its intent, not by deleting it.

- [ ] **Step 4: Run tests**

Run: `make test-go`
Expected: PASS. If a node-sync or version-gate test fails, read it: a master and its nodes are all Dalan, so gates must compare Dalan versions; a gate that is about an upstream feature level must use `GetUpstreamVersion`.

- [ ] **Step 5: Commit**

`feat(version): give Dalan its own version and keep the 3x-ui base version readable`

---

### Task 3: Reproducible rebrand of scripts and workflows

**Files:**
- Create: `scripts/dalan/rebrand.sh`, `scripts/dalan/check-brand.sh`, `scripts/dalan/rebrand_test.sh`
- Modify (by running the script, not by hand): `install.sh`, `update.sh`, `x-ui.sh`, `.github/workflows/docker.yml`, `.github/workflows/release.yml` (the two `-X github.com/mhsanaei/3x-ui/v3/...` LDFLAGS lines must NOT change — they are the Go module path)
- Modify: `Makefile` (new target `brand-check`, added to `verify`)

**Interfaces:**
- Produces: `scripts/dalan/rebrand.sh [ROOT]` — rewrites in place, exit 0. `scripts/dalan/check-brand.sh [ROOT]` — exit 0 when clean, exit 1 printing `file:line` for every offender. Task 5 calls both.

Rules both scripts implement:
- Target files: `*.sh` in the repo root, and `.github/workflows/*.yml`.
- Replace, case-insensitively, `github.com/MHSanaei/3x-ui`, `raw.githubusercontent.com/MHSanaei/3x-ui`, `api.github.com/repos/MHSanaei/3x-ui` and `ghcr.io/mhsanaei/3x-ui` with the same host and `ParsaDa/dalan` (image: `ghcr.io/parsada/dalan`).
- Never touch: `github.com/mhsanaei/3x-ui/v3` (module path), anything containing `mtg-multi`, the strings `/etc/x-ui`, `/usr/local/x-ui`, `x-ui-linux-`.

- [ ] **Step 1: Write the failing test** — `scripts/dalan/rebrand_test.sh`, plain Bash, exits non-zero on the first failed assertion. It builds a temp tree with:
  - `a.sh` containing `https://github.com/MHSanaei/3x-ui/releases/download/v1/x-ui-linux-amd64.tar.gz`, `https://raw.githubusercontent.com/mhsanaei/3x-ui/main/install.sh`, `https://github.com/mhsanaei/mtg-multi/releases/latest`, and `/etc/x-ui/x-ui.db`
  - `.github/workflows/r.yml` containing `ghcr.io/mhsanaei/3x-ui` and `-X github.com/mhsanaei/3x-ui/v3/internal/config.buildCommit=abc`

  Assertions, in order:
  1. `check-brand.sh` on the fresh tree exits 1 and its output names `a.sh` and `r.yml`.
  2. After `rebrand.sh`: `a.sh` contains `https://github.com/ParsaDa/dalan/releases/download/v1/x-ui-linux-amd64.tar.gz` and `https://raw.githubusercontent.com/ParsaDa/dalan/main/install.sh`; still contains the `mtg-multi` line and `/etc/x-ui/x-ui.db` byte-for-byte; `r.yml` contains `ghcr.io/parsada/dalan` and the unchanged `-X github.com/mhsanaei/3x-ui/v3/internal/config.buildCommit=abc`.
  3. `check-brand.sh` now exits 0.
  4. A second `rebrand.sh` run leaves `sha256sum` of every file unchanged.
  5. Adding a new file `new.sh` with an upstream URL makes `check-brand.sh` exit 1 naming `new.sh`.

- [ ] **Step 2: Run to verify it fails**

Run: `bash scripts/dalan/rebrand_test.sh`
Expected: fails, `rebrand.sh: No such file`.

- [ ] **Step 3: Implement the two scripts** to the rules above (`set -euo pipefail`; `sed -E` with the `I` flag; default ROOT is the repo root from `git rev-parse --show-toplevel`).

- [ ] **Step 4: Run the test**

Run: `bash scripts/dalan/rebrand_test.sh`
Expected: prints `rebrand_test: OK`, exit 0.

- [ ] **Step 5: Apply to the repo and wire the gate**

Run: `bash scripts/dalan/rebrand.sh && bash scripts/dalan/check-brand.sh`
Expected: exit 0. Then `git diff --stat` shows only `install.sh`, `update.sh`, `x-ui.sh`, `docker.yml`; `git diff | grep '^[-+]' | grep -c 'x-ui/v3'` prints `0`.
Add to `Makefile`: target `brand-check` running `rebrand_test.sh` then `check-brand.sh`; append `brand-check` to the `verify` prerequisites.

- [ ] **Step 6: Commit**

`feat(brand): rebrand installer, updater and image name with a re-runnable script`

---

### Task 4: Visible name, README and fork-only workflow guards

**Files:**
- Modify: `frontend/src/layouts/AppSidebar.tsx:370,458`, `frontend/src/pages/login/LoginPage.tsx:189`
- Modify: `frontend/public/manifest.webmanifest` (`name`, `short_name` only; icon files stay until the UI redesign phase)
- Modify: `frontend/src/test/navigation-contract.test.tsx` only where it asserts the brand text
- Modify: `README.md`, `README.fa_IR.md` (top section only)
- Modify: `.github/workflows/claude-issue-analyst.yml`, `claude-pr-review.yml`, `docs-deploy.yml` (add `github.repository == 'MHSanaei/3x-ui' &&` to each job-level `if:`)

**Interfaces:**
- Consumes: nothing from earlier tasks.

- [ ] **Step 1: Write the failing test** — in `frontend/src/test/`, a test `brand.test.tsx` that renders `AppSidebar` the way `navigation-contract.test.tsx` does and asserts `screen.getByText('Dalan')` exists and `screen.queryByText('3X-UI')` is null; same for the collapsed rail: text `DL`.

- [ ] **Step 2: Run to verify it fails**

Run: `cd frontend && npx vitest run src/test/brand.test.tsx`
Expected: FAIL, unable to find text `Dalan`.

- [ ] **Step 3: Implement** — sidebar expanded `Dalan`, collapsed `DL`, drawer `Dalan`, login `Dalan`; manifest `name` and `short_name` `Dalan`. Translation JSON files are not touched in this phase.

- [ ] **Step 4: README** — replace the top of both READMEs with: project name Dalan; one paragraph "a fork of 3x-ui that adds managed tunnels between servers; under active development, tunnels are not released yet"; the install command `bash <(curl -Ls https://raw.githubusercontent.com/ParsaDa/dalan/main/install.sh)`; a "Based on" section crediting [3x-ui](https://github.com/MHSanaei/3x-ui) by MHSanaei, GPL-3.0. Leave the rest of the upstream README below a heading "Upstream documentation".

- [ ] **Step 5: Workflow guards** — no test can reach workflow YAML; the demonstration is that after the PR merges, the Actions tab shows those three workflows as skipped.

- [ ] **Step 6: Run the frontend gate**

Run: `cd frontend && npm run typecheck && npm test`
Expected: PASS.

- [ ] **Step 7: Commit**

`feat(brand): show Dalan in the panel and README, and skip upstream-only workflows`

---

### Task 5: Upstream sync procedure

**Files:**
- Create: `scripts/dalan/sync-upstream.sh`, `dalan/UPSTREAM.md`

**Interfaces:**
- Consumes: `scripts/dalan/rebrand.sh`, `scripts/dalan/check-brand.sh` (Task 3).

- [ ] **Step 1: Implement `sync-upstream.sh`**: adds remote `upstream` = `https://github.com/MHSanaei/3x-ui.git` if missing; refuses to run on a dirty tree; `git fetch upstream main`; creates branch `sync/upstream-<YYYYMMDD>`; `git merge --no-commit upstream/main`; for every conflicted file among root `*.sh` and `.github/workflows/docker.yml`, takes the upstream side (`git checkout --theirs`) and stages it; runs `rebrand.sh`; stages the result; if conflicts remain, lists them and exits 1 leaving the merge in progress; otherwise runs `check-brand.sh` and prints the next command (`make verify`).

- [ ] **Step 2: Verify against the real upstream**

Run: `bash scripts/dalan/sync-upstream.sh`
Expected: either "Already up to date" or a completed merge with `check-brand.sh` exit 0. If upstream has moved, finish with `make verify` and include the merge in this branch.

- [ ] **Step 3: Write `dalan/UPSTREAM.md`**: when to sync (before each Dalan release, and when upstream ships a security fix); the one command; what to do on remaining conflicts (Dalan code lives in new files, so a conflict means a touch point listed here changed — list them: `panel.go` URL helpers, `config.go` version embed, three brand strings in the frontend, three workflow guards, README top section); the rule that Dalan features go in new files and packages.

- [ ] **Step 4: Commit**

`chore(upstream): add the upstream sync script and its runbook`

---

### Task 6: Verify, open the PR, cut v0.1.0

- [ ] **Step 1:** Run `make verify`. Expected last line: `verify: OK`. Setup needed once: Node 26, `npm ci` in `frontend/`, `npx playwright install --with-deps chromium`.
- [ ] **Step 2:** Push `claude/dalan-foundation`, open a PR into `main` titled `feat: Dalan foundation — brand, own releases, upstream sync`.
- [ ] **Step 3 (owner action):** In `ParsaDa/dalan` → Settings → Actions, enable workflows (forks start with Actions off) and allow "Read and write permissions" for `GITHUB_TOKEN` so `release.yml` can publish.
- [ ] **Step 4:** After merge, confirm `git ls-remote --tags origin v0.1.0` is empty, then tag `v0.1.0` on `main` and push the tag.
- [ ] **Step 5:** Verify the release: the `release.yml` run is green; the release `v0.1.0` in `ParsaDa/dalan` has `x-ui-linux-amd64.tar.gz`, `x-ui-linux-arm64.tar.gz` and their `.sha256` files; `curl -sI https://github.com/ParsaDa/dalan/releases/latest/download/x-ui-linux-amd64.tar.gz` ends in HTTP 200.
- [ ] **Step 6 (owner action):** On a throwaway VPS run the README install command; the panel opens, the sidebar says Dalan, and the update page reports version `0.1.0` with no update available.
