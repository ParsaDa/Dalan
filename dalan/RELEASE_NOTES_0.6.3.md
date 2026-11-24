# Dalan 0.6.3 — release notes

A faster way to reach the server tools, and a speed test on the dashboard.

## Added

- **Speed test on the dashboard.** The new Speed test card measures this server's own ping, jitter, download and upload against the closest speedtest.net server. A dial follows the live rate while the test runs, and a trend line shows the last 20 results (download and upload). It measures this server only; it never goes through a node or a tunnel. A second click while a test runs joins the running test.
- Three API routes behind it: `POST /panel/api/server/speedtest/start`, `GET /panel/api/server/speedtest/status` and `GET /panel/api/server/speedtest/history`. They stay on the local panel and are not available to node sync tokens.

## Changed

- **The System Tools card is now the first card** of the dashboard (Advanced mode), so Updates, AWG logs, Config, History and Metrics no longer sit below everything else.

## Notes

- The test talks to speedtest.net directly from the server. If the server cannot reach it, the card shows the reason instead of a result. Test servers on private or loopback addresses are never contacted.
- No migration needs action. The history is kept in one hidden panel setting (`speedtestHistory`) that takes its default on first start.
- Translations for the new text are written in Persian; the other languages show English until translated.
