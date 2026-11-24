# Dalan 0.6.4 — release notes

Speed test for every tunnel.

## Added

- **Speed test on a tunnel.** An entry tunnel row (and card) on the Tunnels page now has a Speed test button. It measures ping, jitter, download and upload through that tunnel: the test goes into the entry's loopback relay and leaves from the exit server, so the result is the tunnel's own speed to the internet. Each tunnel keeps its last 20 results, shown as the same dial and trend as the dashboard card.
- Three API routes behind it: `POST /panel/api/tunnels/speedtest/start/:id`, `GET /panel/api/tunnels/speedtest/status/:id` and `GET /panel/api/tunnels/speedtest/history/:id`.

## Notes

- Only an enabled reverse entry tunnel (BackPack or Rakhsh) with a relay can be tested. An exit end, a direct tunnel and a management tunnel are refused with a clear message.
- One speed test runs at a time on a panel. Starting the same tunnel again joins the running test; starting another tunnel is refused until it ends.
- The exit server must reach speedtest.net. The dashboard card still measures only this server.
- No migration needs action: the per-tunnel history is one hidden panel setting (`tunnelSpeedtestHistory`) that takes its default on first start.
- Translations for the new text are written in Persian; the other languages show English until translated.
