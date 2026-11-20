# Task: finish applying your Dalan 0.6.0 design to the real project

You designed the Dalan 0.6.0 UI (canvas: https://claude.ai/artifact/2WxRin6u9HWf1KbcfpR73V, design system: https://claude.ai/artifact/6Fu6pnhBE3fABajfh9BXzm, notes in the Project doc `design/2026-11-20-ui-0.6.0-design-delivered.md`). A first pass was built by another agent and is in PR #22 (`ParsaDa/Dalan`, branch `feat/dalan-0.6.0-ui`). That pass applied the LOOK (tokens, glass, glow, accents, logo, rail, phone tab bar, flags, node tabs, login/sub, effects setting, RTL) but NOT most of the CONTENT you designed. The owner found, for example, that the dashboard has no tunnel status. Your job: build what is missing, so the real panel matches your artboards.

## Repo and rules (read first)
- Repo `ParsaDa/Dalan` (a 3x-ui fork). Read `CLAUDE.md`, `dalan/HANDOFF.md`, `docs/architecture.md`, `frontend/CLAUDE.md` before coding. Frontend: React 19 + Ant Design 6 + Vite + TypeScript strict, in `frontend/`. Backend: Go (Gin, GORM).
- Start from `main` after PR #22 is merged (ask the owner, Parsa, if it is). Make a new branch, e.g. `feat/dalan-0.6.1-ui-content`. One PR, one commit per page or part, conventional commits. Open the PR with the GitHub REST API (`gh api repos/ParsaDa/Dalan/pulls --input file.json`; GraphQL is blocked). Do not merge: the owner merges.
- Hard rules from CLAUDE.md: TDD (write the test, watch it fail, then the code); comments at most 2 lines; every new English i18n key goes in all 14 files in `internal/web/translation/` AND is used in code in the same commit (Persian properly translated, others English copy); a new `g.GET/g.POST` needs an entry in `frontend/src/pages/api-docs/endpoints.ts`, then `make gen`, then copy `frontend/public/openapi.json` to `docs/public/openapi.json` and run `cd docs && pnpm gen:api`; new Go response structs go into openapigen's `StructAllow`; state changes go through `runtime.Runtime`; model changes need a migration in `internal/database/db.go`. Use Zod schemas in `frontend/src/schemas/`; no `any`.
- Use the existing tokens in `frontend/src/styles/dalan/` (`tokens.css`, `panel.css`) and the components in `frontend/src/components/ui/` (`PanelBackground`, `Flag`, `CountUp`, `NodeAccentScope`, `AccentSwatches`, `DalanLogo`). Node pages use `NodeAccentScope`.
- NEVER invent data. If a number or list in your artboard has no backend source, either build the smallest correct backend for it (with tests) or show the section only when real data exists and tell the owner which sections wait for backend work. No fake "2 alerts" or fake activity.
- Verify with real screenshots: build (`cd frontend && npx vite build`), `go build`, run the panel on a throwaway DB, and screenshot with headless Chrome (dark, light, phone 390 px, Persian RTL). Two real bugs in PR #22 were found only this way. Unit tests cannot see CSS. Before the PR: `npx oxfmt --check src`, `npx tsc --noEmit`, `npx oxlint src`, `npx vitest run --project unit --project components`, `make gen` leaves no diff. CI also runs Storybook with an axe contrast gate (text 4.5:1): check contrast on new components.

## What exists today (do not redo)
Tokens, theme engine, accents (12), effects levels, glass look, rail and phone tab bar, logo, node country + accent (backend + form), flags (16), dashboard greeting + node tabs + count-up tiles (CPU/RAM/SWAP/Storage, overall speed, connection stats, uptime/panel/IP), login + subscription look, Settings appearance, command palette (with effects cycling), RTL for fa/ar.

## What is missing: do these, in this order
Compare each live page with its artboard in the canvas (file names in brackets) and close the gap.

1. **Dashboard** [Main, Dashboard-Light, Dashboard-FA, Phone-Dashboard, Phone-Dashboard-FA]. The old dashboard content is still there. Build the designed one:
   - Header: greeting with the owner's name, status line ("all systems healthy · updated 4s ago"), search field with Ctrl K, effects switch.
   - **Server health card**: CPU, memory, disk, Xray core state + uptime.
   - **Tunnels card**: "3 of 4 up", one row per tunnel: route (Iran → node), engine (BackPack/Rakhsh), mode (reverse / direct / layer 3), latency, Up/Down. Data comes from the existing tunnel status API (`/panel/api/tunnels/*`, see `internal/web/controller/tunnel.go`).
   - **Usage and online users**: online now, throughput, monthly traffic percent (only if a monthly quota source exists).
   - **Alerts**: "N need attention" (node offline, node CPU high). The event bus has `node health`, `cpu.high`, `memory.high`, `xray.crash`, but nothing is stored; decide a minimal persisted alert/event store or derive alerts from live node state, and report which.
   - **Activity** feed (who did what). No audit log exists in the backend. Decide: a small `activity` table written from the controllers that change things, or leave this card out and tell the owner. Ask before building a big audit system.
   - **Connected nodes** cards: flag, name, latency, state (Online / High load / Offline), CPU, "last seen" for offline. Direct and chained counts.
   - Keep what the old dashboard showed that is still useful (speed chart, connection stats) as the designed "Advanced" view if your artboards show it that way.
2. **Navigation** [Nav-Modes]. The rail should have your groups (Overview: Dashboard, Activity; Users and access: Inbounds, Clients with count badge, Groups, Hosts; Network: Tunnels, Outbounds, Routing, Nodes, Tor and PIA; System: Xray config, Logs, Settings; Help) with the three nav modes you designed. Check which of these are separate pages today (Outbounds, Routing, Tor and PIA and Logs are modals or parts of other pages now) and make them reachable as the design shows.
3. **Nodes page** [Nodes, Node-Edit]: node summary cards and the tunnel flow diagram (you chose these instead of a world map), plus the redesigned edit form.
4. **Tunnels page** [Tunnels, Phone-Tunnels, Outbounds]: the designed list/cards with engine, mode, spare pools, and route test.
5. **Clients, Inbounds, Groups, Hosts** [Clients, Clients-Light, Phone-Clients, Inbounds, Groups, Hosts]: table and row design, filters, bulk bar, phone cards.
6. **Settings, Xray config, Logs, Developer (API docs)** [Settings, Phone-Settings, XrayConfig, Logs, Developer].
7. **Tor and PIA** [TorPIA], **Wizard (first run)** [Wizard, Wizard-Phone]. There is no first-run state in the backend; propose the smallest correct one and ask the owner before building.
8. **States** [States, Loading, Error, Motion]: empty, loading skeletons, error pages, motion spec (stagger, count-up) on every page, not only the dashboard.
9. **Phone and Persian pass** on every page you touched (RTL, bidi of numbers with units, tab bar).

## How to report
When each page is done: one line saying what now matches the artboard and what does not, with a screenshot path. At the end: list of sections you left out because the backend has no data, each with a recommendation. Do not claim a page matches unless you looked at a screenshot of it.
