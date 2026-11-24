# Dalan 0.7.4 — release notes

Fix for Compare methods on a new tunnel.

## Fixed

- **Compare methods reported "the exit server does not answer the speed test" for every method (`socks connect ... EOF`).** A tunnel that has just been made is connected before its exit end is ready: the exit server's Xray only picks up the new relay on its next restart pass, which can take up to 30 seconds. The test started at once and found nothing there. It now keeps trying for 45 seconds before it reports the exit server as not answering. A run therefore takes longer per method (up to 30 seconds more), and a node that really does not have the link speed test still stops the run after three methods.
