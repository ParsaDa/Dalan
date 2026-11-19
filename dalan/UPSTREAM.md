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
| `Makefile` | `brand-check` target in `verify` (also runs `deploy/test/container-prepare.sh`); `node-e2e` also builds an Xray core for the enrolment test |
| `internal/web/service/xray.go` | one block before the final `dropRefusedScript` calls in `GetXrayConfig` injects tunnels |
| `internal/web/controller/api.go` | registers `/tunnels` and `/enrolments` (the latter also adds the public `dalan/enrol` route) |
| `internal/web/controller/node.go` | `del` closes the management tunnel of a deleted node |
| `main.go` | `enrol` command (its code is in `enrol_cmd.go`) and its help line |
| `x-ui.sh` | `enrol` case, added by `scripts/dalan/rebrand.sh` |
| `install.sh` | `XUI_ENROL` step before the final summary, added by `scripts/dalan/rebrand.sh` |
| `Dockerfile`, `DockerEntrypoint.sh` | copy and source `DockerPrepare.sh` (platform `PORT`, `DALAN_*` variables); `main.go` also starts the front door (`internal/frontdoor`) |
| `.github/workflows/ci.yml` | shellcheck list includes `DockerPrepare.sh` and `deploy/test/container-prepare.sh` |
| `internal/web/web.go` | starts BackPack and Rakhsh tunnels after Xray in `startTask`, stops them before Xray in the stop path |
| `internal/database/db.go`, `migrate_data.go` | `ServerTunnel` and `NodeEnrolment` in the model lists |
| `tools/openapigen/main.go` | `ServerTunnel`, `NodeEnrolment` and the `enrolment` package in `StructAllow` |
| `frontend/src/routes/panelRoutes.tsx`, `CommandPalette.tsx` | Tunnels route and its palette keywords |
| `frontend/src/pages/nodes/NodeList.tsx` | `<EnrolNode />` button in the toolbar |
| `frontend/src/pages/api-docs/endpoints.ts` | Tunnels and Node enrolment sections |
| `frontend/src/pages/xray/outbounds/` (`OutboundsTab.tsx`, `useOutboundColumns.tsx`, `OutboundCardList.tsx`, `outbounds-tab-types.ts`, `outbounds-tab-helpers.ts`) | tunnel rows in the outbounds table and cards, read-only, with a Reverse tag |
| `internal/web/service/setting.go`, `xray_setting.go` | the `torBridges` setting and the Tor exits payload of the Xray settings |
| `internal/web/service/xray.go` | a second block injects the Tor exit outbounds (`tor.Apply`) |
| `internal/web/web.go`, `controller/api.go`, `database/db.go`, `migrate_data.go` | start and stop Tor exits with the panel, mount `/tor`, register `TorExit` |
| `frontend/src/pages/xray/XrayPage.tsx`, `hooks/useXraySetting.ts`, `lib/xray/injected-outbound-tags.ts` | Tor window state, Tor status data, `tor-` tags counted as injected outbounds |
| `frontend/src/api/websocketBridge.ts` | the `outbounds` push also refreshes the tunnels' traffic in the query cache |
| `internal/web/translation/*.json` | `menu.tunnels`, `pages.tunnels`, `pages.enrolments` and `pages.tor` |

Keep upstream's change and re-apply the Dalan line on top.

## Rules that keep this cheap

- New Dalan code goes in new files and new packages.
- Never rename the Go module path, the `x-ui` binary, `/etc/x-ui`, `XUI_*` variables or the service units.
- Never edit `internal/config/version`; it is the 3x-ui version Dalan is built on, and nodes are checked against it before newer protocols are pushed to them. Dalan's own version is `internal/config/dalan_version`.
- Never hand-edit a URL in a root `*.sh` file; change `scripts/dalan/rebrand.sh` and re-run it.
