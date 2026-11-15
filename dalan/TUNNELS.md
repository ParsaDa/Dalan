# Tunnels

A tunnel joins two Dalan servers. Users connect to the **entry** server; their traffic leaves from the **exit** server. The exit server opens the connection, so nothing has to be reachable on the exit side and no connection is started from the entry server.


## Set one up

Both servers need Dalan.

**On the entry server** (the one users connect to), open **Tunnels → New tunnel**:

| Field | What to enter |
|---|---|
| Name | Short, lower-case: `de1` |
| Address | This server's public IP or domain, as the exit server will reach it |
| Port | A free port on this server; open it in the firewall |
| Security | REALITY (recommended) |
| Server name (SNI) | A real HTTPS site reachable from this server; the tunnel imitates it |
| Inbounds that leave through this tunnel | The inbounds of this server whose users should exit from the other server |

Then press the copy button on the tunnel's row to copy its **setup link**.

**On the exit server**, open **Tunnels → Import link**, paste the link, and confirm. Leave *Exit outbound* empty to leave directly from that server, or name one of its outbounds (for example a WARP outbound).

Xray restarts on each server within 30 seconds and the exit server connects.

## What to know

- The setup link holds the tunnel's secret. Treat it like a password. It does not contain the entry server's REALITY private key.
- "None" security sends the tunnel unencrypted and recognisable. It exists for links that are already private; do not use it across a filtered network.
- The entry end appears in Xray as an outbound named `tunnel-<name>`. Routing rules and balancers can use that tag.
- For traffic that arrives through a tunnel, the exit server refuses private and loopback destinations. Users cannot reach the exit server's own network.
- Deleting a tunnel on one server does not delete it on the other.

## Two engines

**Xray (built in)** is the default. It needs nothing installed, works everywhere Dalan runs, and routes whole inbounds of the entry server through the tunnel.

**BackPack** is for links that are filtered hard or lose packets. It is a separate program that Dalan installs and runs for you. It works differently: instead of routing inbounds, it opens ports on the entry server and carries them to the exit server.

### A BackPack tunnel

1. On both servers open **Tunnels**. If a notice says the BackPack engine is not installed, press **Install BackPack**. Dalan downloads the pinned BackPack release, checks it against the checksums published with it, and puts it in the panel's `bin` folder. A server that cannot reach GitHub needs the `backpack` binary copied into that folder by hand.
2. On the **exit** server, create the inbound users will use, for example VLESS on port 2096.
3. On the **entry** server, **New tunnel**, choose **BackPack**:
   - *Transport*: how the tunnel looks on the wire (`wssmux`, `tcpmux`, `quic`, `kcp`, ...).
   - *Fallback transports*: tried in order when the main one is blocked.
   - *Forwarded ports*: `443=127.0.0.1:2096` opens 443 on the entry server and delivers it to port 2096 on the exit server.
4. Copy the setup link and import it on the exit server.

Users then connect to the entry server on port 443 and reach the inbound on the exit server.

Dalan keeps one BackPack process per tunnel, restarts it if it exits, and stops it when the tunnel is disabled or deleted. A BackPack tunnel shows **Connected** on both servers, taken from what the engine itself reports; it has no latency figure.

Some BackPack transports need root or raw packets (`pck`, `xdi`) and do not work in containers.

## Status

On the entry server the Tunnels page tests each enabled tunnel once a minute by sending a real request through it. A working tunnel shows **Connected**, the round-trip time, and the country and address the request left from. **Down** means the request did not come back; hover it for the reason.

The exit server cannot test a tunnel by itself, so its rows say so.

The test uses a loopback port per tunnel (62900 + the tunnel's id). If an inbound already uses that port, that tunnel is not tested.

## Not built yet

- Traffic totals per tunnel in the panel.
- Creating both ends from one panel, and tunnels created automatically when a node is added.
- Spare tunnels with automatic switching.
- Xray tunnels over transports other than REALITY on TCP.
- Updating the BackPack engine from the panel after it is installed.
