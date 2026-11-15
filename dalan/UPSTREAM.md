# Taking updates from 3x-ui

Dalan is a fork of [3x-ui](https://github.com/MHSanaei/3x-ui). This is how upstream changes get in.

## When

- Before every Dalan release.
- As soon as upstream ships a security fix.

## How

From a clean checkout of `main`:

```bash
bash scripts/dalan/sync-upstream.sh
make verify
git commit
```

The script fetches upstream `main`, merges it into a new `sync/upstream-<date>` branch, takes upstream's version of the branded files (`install.sh`, `update.sh`, `x-ui.sh`, `.github/workflows/docker.yml`) and re-applies branding with `scripts/dalan/rebrand.sh`. Open a pull request from that branch.

Pass a tag to merge a specific release: `bash scripts/dalan/sync-upstream.sh v3.12.0`.

## When it stops with conflicts

Dalan features live in new files and packages, so a conflict means upstream changed one of the few places Dalan touches:

| Place | What Dalan changed |
|---|---|
| `internal/web/service/panel/panel.go` | release and updater URLs come from `panel_brand.go`; the update check compares `config.GetDalanVersion()` |
| `frontend/src/layouts/AppSidebar.tsx`, `frontend/src/pages/login/LoginPage.tsx` | brand text |
| `frontend/public/manifest.webmanifest` | app name |
| `.github/workflows/claude-issue-analyst.yml`, `claude-pr-review.yml`, `docs-deploy.yml` | skipped outside the upstream repository |
| `README.md`, `README.fa_IR.md` | top section |
| `Makefile` | `brand-check` target in `verify` |

Keep upstream's change and re-apply the Dalan line on top.

## Rules that keep this cheap

- New Dalan code goes in new files and new packages.
- Never rename the Go module path, the `x-ui` binary, `/etc/x-ui`, `XUI_*` variables or the service units.
- Never edit `internal/config/version`; it is the 3x-ui version Dalan is built on, and nodes are checked against it before newer protocols are pushed to them. Dalan's own version is `internal/config/dalan_version`.
- Never hand-edit a URL in a root `*.sh` file; change `scripts/dalan/rebrand.sh` and re-run it.
