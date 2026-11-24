# Dalan 0.7.0 — release notes

More ways to carry a tunnel, and a way to find the best one.

## Added

- **Six forwarding engines.** GOST, Rathole, Backhaul, Kariz, sing-box and Xray can now carry a tunnel next to BackPack and Rakhsh. Each has its own card with install and update, and a tunnel picks the engine and then one of its carriers (TLS, WebSocket, gRPC, HTTP/2, QUIC, KCP, mKCP and more). Both ends share one secret carried in the setup link, so the second server is set up by importing the link.
- **Kernel carriers.** A tunnel can run on a GRE, IPIP or VXLAN link built in the Linux kernel, with the forwarded ports relayed over a private link between the servers. It is behind an **Advanced engines** switch, needs root and the `ip` command on both servers, and asks for the public IPv4 address of the end that dials.
- **Compare methods.** A button on the Tunnels page tries the methods you pick between this server and a node: a temporary tunnel is made on both ends, measured with the speed test, and removed. The methods are ranked by speed or by a balance of speed, ping and jitter, and one click copies the best into the new-tunnel dialog.
- **Per-tunnel speed test** (from 0.6.4, now included in this release). It measures the tunnel itself, between this server and the exit server, with no speedtest.net involved; the exit server answers on a loopback port next to the tunnel's relay port. Compare methods ranks methods by the same measurement.
- New API routes: `GET /panel/api/tunnels/engines`, `GET /panel/api/tunnels/benchmark/status`, `POST /panel/api/tunnels/benchmark/start` and `POST /panel/api/tunnels/benchmark/cancel`.

## Notes

- Kernel carriers fail with a clear message when the panel is not root, is not on Linux, or has no `ip` command. GRE and IPIP are not encrypted or authenticated, and many hosts block them; test with the speed test first. They are not part of Compare methods.
- Open the carrier's protocol in both firewalls: IP protocol 47 (GRE), IP protocol 4 (IPIP) or the tunnel's UDP port (VXLAN).
- A Compare methods run makes tunnels named `dalan-bench-<n>`. If the panel stops during a run, leftovers are removed when it starts again.
- The exit server must also run 0.7.0 or newer to answer the speed test; an older one is reported as not answering.
- No database migration is needed.
- Translations for the new text are written in Persian; the other languages show English until translated.
