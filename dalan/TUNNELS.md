# Tunnels

A tunnel joins two Dalan servers. Users connect to the **entry** server; their traffic leaves from the **exit** server. The exit server opens the connection, so nothing has to be reachable on the exit side and no connection is started from the entry server.

Tunnels run inside Xray, on its VLESS reverse proxy. No extra program is installed.

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

## Not built yet

- Connection status, latency and traffic per tunnel in the panel.
- Creating both ends from one panel, and tunnels created automatically when a node is added.
- Spare tunnels with automatic switching.
- Other engines (BackPack) and transports other than REALITY over TCP.
