# Tor exits

Dalan can send traffic out through the Tor network, choosing the country the traffic leaves from. Tor runs on the Dalan panel server itself. It does not involve nodes or tunnels, and a node never runs it for the main panel.

## What you get

For each country you add, Dalan runs one Tor process with the setting `ExitNodes {cc}` and `StrictNodes 1`, so its traffic can only leave from a relay in that country. Each process listens on a SOCKS5 port on `127.0.0.1` and Dalan adds it to Xray as an outbound named `tor-<cc>` (for example `tor-de`).

The outbound appears in **Xray → Outbounds** with a status chip: **Connected** with a latency figure, **Not connected** with the reason, **Starting**, **Stopped** or **Not installed**. "Connected" means a real request went through that Tor, not just that the process is alive. Routing rules, balancers and inbound routes can name the tag, the same way they name a tunnel.

## Set one up

1. Open **Xray → Outbounds → Tor exits by country**.
2. If the notice says Tor is not installed, press **Install Tor 16.0.1**. Dalan downloads the official Tor Expert Bundle from `dist.torproject.org`, checks it against the checksums published beside it and puts it in the panel's `bin/tor-bundle` folder.
3. Search a country (flag and English name) and press **Test**. Dalan starts a throwaway Tor for that country and reports "Works" with the latency and exit location, or the reason it failed. One test runs at a time and takes up to 90 seconds.
4. Press **Add**. Dalan starts a Tor for the country; it shows in Outbounds within a minute or so, once Tor has built its circuits.
5. Use `tor-<cc>` in **Routing** like any outbound.

Removing a country is refused while a routing rule, a balancer or a panel egress setting still names its tag. Remove that reference first.

## Limits

- **At most 10 exits** are enabled at once; each is a Tor process that uses tens of megabytes.
- **Ports 62700-62789** on `127.0.0.1` are used, one per exit. Nothing is opened to the network.
- **TCP only.** Tor does not carry UDP, so games, calls and QUIC do not go through a Tor exit.
- **Not encrypted at the exit.** The last relay can read traffic that is not itself encrypted (plain HTTP, for example). Use Tor only for what you accept that for.
- **Slow and variable.** Speed depends on the relays in that country. A country with few exit relays may fail to connect; the test says so ("no exit relays reachable in this country").

## Bridges

If this server cannot reach the Tor network directly, open **Advanced** in the Tor window and paste bridges, one per line. The supported types are `obfs4`, `snowflake`, `webtunnel` and `meek_lite`; each line is checked before it is saved. The bridges apply to every exit. Get bridges from `bridges.torproject.org`.

## State on disk

- Bundle: the panel's `bin/tor-bundle` (the previous one is kept beside it as `tor-bundle.previous` for a rollback).
- Per-exit config and Tor state: the database folder, under `tor/` (`tor/data/<cc>`); the state survives a restart, so Tor does not fetch the relay list again each time.

## What is not verified

The Tor bundle's download address, its internal layout and its checksum file were written from the Tor Project's published naming and could not be reached from the build environment. If **Install** answers an error, that is the first suspect. To install by hand, unpack a Tor Expert Bundle for this server's processor so that the folder `tor-bundle` in the panel's `bin` folder holds `tor/tor`, its libraries beside it, and `tor/pluggable_transports/`; then press **Test** again. Also not run on real servers: any country test, the bridge types, and routing through an exit. The aarch64 bundle is assumed to exist for the pinned version.
