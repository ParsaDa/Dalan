# Tor outbound, by country (Dalan 0.5.1)

## Intent

Parsa used to run a script that started Tor and exposed a SOCKS5 port anyone could use. He wants that as a
first-class **outbound** in Dalan: pick a country, Dalan runs Tor with exits in that country, and Xray routing
can send chosen users, inbounds or a whole location through it. Tor runs inside Dalan itself, on the panel
server, with no foreign node needed.

Why: some traffic should leave through a Tor exit in a chosen country, and routing it the way Dalan already
routes tunnels (one outbound tag, any rule can target it) keeps one place for routing rules.

## Goals, non-goals, anti-goals

Goals:

- Add a **Tor** entry to the Outbounds tab. It opens a country picker listing as many countries as possible,
  each with a **Test** button saying whether Tor can reach that country now.
- A country added from the picker becomes an outbound `tor-<cc>` (for example `tor-de`). It shows in the
  Outbounds table with a live status (starting / connected with latency / not connected with a reason), and is
  selectable in routing rules and balancers like a tunnel outbound.
- Tor is installed and checked by Dalan (install button, pinned release, verified checksum), like the engines.
- Optional **bridges** (obfs4, snowflake) so Tor can start when the panel server itself sits where Tor is blocked.

Non-goals (0.5.1):

- A SOCKS5 port reachable from outside the server. The old script's open port is the hole this removes; an
  authenticated public port can come later if asked.
- Tor on nodes. Exits run on the panel server only.
- UDP. Tor carries TCP only, so QUIC and voice sent to a Tor outbound do not work; the UI says so.
- New-identity button, automatic country rotation, per-client country choice in the client UI.
- A fully verified list of countries that have exits. The list is every ISO country; the Test button is the truth.

Anti-goals: a second routing system, an unauthenticated open proxy, a "connected" label that is only a running
process (status must come from a real request through the SOCKS port).

## Approaches

1. **One managed Tor process per country, injected as a `socks` outbound** (chosen). Each enabled country runs
   its own Tor (`ExitNodes {cc}`, `StrictNodes 1`) on a loopback SOCKS port. At config generation Dalan injects
   `{tag:"tor-<cc>", protocol:"socks", 127.0.0.1:<port>}`, the way tunnels are injected. Costs memory per country
   (tens of MB each), so the number of enabled countries is capped.
2. **One Tor process, many SocksPorts, country picked per circuit over the control port.** One process, but Tor
   has no per-port `ExitNodes`; this needs control-port circuit management and stream attachment. Fragile and a
   lot of new code. Rejected.
3. **Use a system `tor` package.** Not present in the Docker image, differs per distro, no bridge transports.
   Rejected; the engine pattern (pinned download, optional hand-placed binary) already exists.

## Design

**Delivery.** The Tor Expert Bundle (Tor Project; contains `tor`, `geoip`, `geoip6` and the pluggable
transports for obfs4 and snowflake). Dalan pins a version and the SHA-256 of each architecture's archive **in
source**, so a swapped download fails; it installs under the bin folder, keeping the previous one for rollback
(same Place / Rollback / InstalledVersion helpers as the engines in `internal/tunnel/engine`). A hand-placed
binary still works. An architecture the bundle does not publish gets a plain "not available for this
processor" error.

**Model.** New table `TorExit` (`id`, `country` two-letter lower-case and unique, `port`, `enable`), with the
migration in `internal/database/db.go` and `migrationModels`. Bridge lines are one setting, `torBridges`
(newline separated), applied to every exit; non-empty means `UseBridges 1` plus the matching
`ClientTransportPlugin` lines.

**Runtime.** A supervisor runs one Tor per enabled exit with its own `DataDirectory`, `SocksPort
127.0.0.1:<port>` and `ClientOnly 1`, restarting a crashed one. Reuse `engine.Supervisor`'s start / stop /
restart logic rather than copying it; it is keyed on a tunnel row today, so how it is generalised is the
builder's call. Ports come from a loopback range (suggested 19050 up) and must not clash with any inbound or
tunnel port. After every change `xrayService.SetToNeedRestart()` as tunnels do.

**Xray injection.** At config generation, enabled exits become `socks` outbounds with tag `tor-<cc>`. They are
not stored in the template. The Xray-setting response gains `torOutbounds` (tag, country, status) so the
frontend lists them in the Outbounds table and offers them in routing and balancer pickers (extend
`injectedOutboundTags`).

**Status.** Per exit: not installed, starting (bootstrap percent read from Tor's output), connected (a real
request through the SOCKS port succeeded; latency shown), or not connected (reason: no exit relays in that
country, bridges needed, timed out). The request reuses `tunnel.ProbeVia`.

**Test button.** Starts a temporary Tor with that country on a random loopback port and its own throwaway
directory, waits for bootstrap, sends one request, reports the result, and always stops and removes it. One test
at a time per panel; about 90 seconds at most; a country that already runs reports its live status instead.
The test is advice: adding a country does not require a passing test.

**API.** Panel endpoints under `/panel/api/tor/`: list, add, delete, enable, test, engine info / install /
rollback, bridges get / set. Each new route needs its `endpoints.ts` entry, `make gen`, and the docs openapi
copy and MDX, per CLAUDE.md. New structs go in openapigen's `StructAllow`.

**UI.** A **Tor** button next to Warp / Nord / PIA in the Outbounds tab opens `TorModal`: a searchable list of
countries (flag, name in the panel language, via the existing `countryFlag` / `countryName`), a Test button per
row with its result, an Add / Remove action, an engine card (installed version, install button) and a bridges
box under Advanced. Outbounds-table rows for Tor show the status and link to the modal instead of an Edit
action. Exact visual design comes from the separate design chat; this spec fixes behaviour, not looks.
All new strings go into all 14 locale files (Persian and English real, the other twelve English).

**Security.** SOCKS is bound to 127.0.0.1 only. Bridge lines and Tor data stay owner-only on disk. The UI notes
that a Tor exit can read unencrypted traffic.

## Decisions I made (not Parsa's, flag if wrong)

- Cap of **10 enabled exits** at once (memory); adding an eleventh gives a clear error.
- Bridges are **global**, not per country.
- Tor exits are **panel-local**; a routing rule on a node that names a `tor-` tag must not break that node's
  config (builder to confirm how template rules reach nodes and block or skip them).
- Version bump to 0.5.1; short operator note `dalan/TOR.md`.

## Left to the builder

Which Tor Expert Bundle release to pin (needs x86_64 and, if published for the chosen line, aarch64, with
hashes checked at pin time); exact Tor log lines used for bootstrap and failure reasons; the generalisation of
`engine.Supervisor`; REST route names; test and component layout.

## Tests that must be able to fail

- torrc rendering: exact `ExitNodes {cc}`, `StrictNodes 1`, loopback `SocksPort`, bridge lines present only when
  set.
- Port allocation: never returns a port in use by an inbound, tunnel or another exit; refuses the 11th enabled
  exit.
- Injection: an enabled exit appears as a `socks` outbound `tor-<cc>` in the generated config; a disabled one does
  not.
- Install: a wrong checksum is refused and the old binary stays.
- Status parsing from real Tor log lines; "connected" only after a successful request.
- Test endpoint cleans up its process and directory on success, failure and timeout.
- Frontend: Tor row in the Outbounds table; picker lists many countries and filters; per-row Test shows its three
  outcomes; Tor tags selectable in routing rules.
- Contract tests (`TestRouteRegistryContract`, i18n dead-key test, openapi codegen) stay green.
