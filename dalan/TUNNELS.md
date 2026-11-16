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

**BackPack** is for links that are filtered hard or lose packets. It is a separate program that Dalan installs and runs for you. Like an Xray tunnel it appears in Xray as an outbound named `tunnel-<name>` that routing rules, balancers and inbound routes can use. It can also open raw ports on the entry server and carry them to the exit server.

### A BackPack tunnel

1. On both servers open **Tunnels**. If a notice says the BackPack engine is not installed, press **Install BackPack**. Dalan downloads the pinned BackPack release, checks it against the checksums published with it, and puts it in the panel's `bin` folder. A server that cannot reach GitHub needs the `backpack` binary copied into that folder by hand.
2. On the **exit** server, create the inbound users will use, for example VLESS on port 2096.
3. On the **entry** server, **New tunnel**, choose **BackPack**:
   - *Transport*: how the tunnel looks on the wire (`wssmux`, `tcpmux`, `quic`, `kcp`, ...).
   - *Fallback transports*: tried in order when the main one is blocked.
   - *Inbounds*: the entry-server inbounds whose traffic leaves through the tunnel.
   - *Forwarded ports* (optional): `443=127.0.0.1:2096` opens 443 on the entry server and delivers it to port 2096 on the exit server as a raw pipe.
4. Copy the setup link and import it on the exit server.

With *Inbounds*, users keep connecting to the entry server and their traffic exits from the exit server. Dalan carries it over a loopback SOCKS relay pair, one port from 63400-64399 on each end, bound to 127.0.0.1 and protected by a password derived from the tunnel token.

With *Forwarded ports*, users connect to the entry server on port 443 and reach the inbound on the exit server; the traffic is not routed, so they get the entry server's address.

Every enabled entry tunnel is listed read-only under **Xray → Outbounds** and can be chosen in **Routing** and **Balancers**.

Dalan keeps one BackPack process per tunnel, restarts it if it exits, and stops it when the tunnel is disabled or deleted. A BackPack tunnel shows **Connected** on both servers, taken from what the engine itself reports; it has no latency figure.

Some BackPack transports need root or raw packets (`pck`, `xdi`) and do not work in containers.

## Status

On the entry server the Tunnels page tests each enabled tunnel once a minute by sending a real request through it. A working tunnel shows **Connected**, the round-trip time, and the country and address the request left from. **Down** means the request did not come back; hover it for the reason.

The exit server cannot test a tunnel by itself, so its rows say so.

The test uses a loopback port per tunnel (62900 + the tunnel's id). If an inbound already uses that port, that tunnel is not tested.

## Nodes that dial in

A server abroad that the main panel cannot reach can still be one of its nodes: the node opens a tunnel to the main panel and is managed through it. The main panel never opens a connection toward the node.

Both servers need Dalan.

1. On the main panel open **Nodes → Add node with a command**.

   | Field | What to enter |
   |---|---|
   | Node name | Short, lower-case: `de1` |
   | This panel's address | The panel as the new node can reach it, with its path: `https://panel.example.com:2053/secret/` |
   | Address | This server's public IP or domain, as the node will dial the tunnel |
   | Port | A free port on this server for this node's tunnel; open it in the firewall |
   | Security, Server name (SNI) | As for any tunnel; REALITY is recommended |

   Instead of a new tunnel you can pick **Tunnel that carries it**: an existing entry tunnel. The node is then managed over that tunnel, with the same address, port and disguise and nothing new to open. On an Xray tunnel the node gets a second client on the tunnel's own port; on a BackPack tunnel (which the node must already have) the panel adds a loopback forward to the node's panel port. A tunnel that carries a node cannot be deleted until that node is removed; if it goes down, the node's management goes down with it.

   **Also make a tunnel for this node's traffic** (on by default for a new tunnel) adds an ordinary entry tunnel named after the node, on the same port and with the same disguise. The node's management rides on it, so one port serves both, and the tunnel can be picked as an exit in Outbounds, Routing and the inbound form as soon as the node has joined. Deleting that tunnel removes the node's end of it too; it cannot be deleted while the node exists.

2. Press **Create command** and copy the command. It is shown once, works once and expires after 30 minutes.
3. On the node, run it:

   ```bash
   dalan enrol 'dalan://enrol/...'
   ```

   Xray restarts on the node and the node appears online in the main panel's node list within a minute.

What to know:

- The node is listed with the address `127.0.0.1` and a port between 61000 and 61999. That port exists only on the main panel's loopback and leads through the tunnel to the node's panel.
- On the node, whatever arrives through this tunnel can reach the node's own panel port and nothing else.
- On a server without Dalan, run the second command of the dialog (`XUI_ENROL='dalan://enrol/…' bash <(curl -Ls …/install.sh)`): it installs Dalan and then joins this panel. If joining fails the install still finishes, and `x-ui enrol '<link>'` can be run again until the command expires.
- The enrolment itself needs the main panel's web port to be reachable from the node, once. Afterwards only the tunnel port is used. The exchange is encrypted with a key inside the command, so it is safe even when the panel is served over plain HTTP.
- The command gives the main panel an API token of the node. Treat the command like a password until it is used.
- A node with its own tunnel needs its own tunnel port on the main panel; a node carried by an existing tunnel needs none.
- Deleting the node in the main panel closes its tunnel there. On the node, the tunnel `node-<name>` stays until it is deleted in the node's Tunnels page or the node joins another main panel.
- Running `dalan enrol` again with a new command moves the node to that main panel; the earlier panel's token stops working.
- If the node's panel is set to listen on one specific IP, clear that setting first: the tunnel delivers to `127.0.0.1`.
- A waiting enrolment can be cancelled in the same dialog.

## Making both ends from one panel

In **Tunnels → New tunnel**, **Create the other end on a node** picks one of the panel's nodes. The panel stores its entry end, then asks the node (through the management channel) to store the dialling end, so nothing is copied or pasted. If the node cannot take its end, nothing is left behind. Deleting the tunnel removes both ends; when the node cannot be reached, the panel offers to remove only its own end.

## Which port passes: Test ports

In **Tunnels → New tunnel**, after choosing a node under **Create the other end on a node** and typing this server's address, **Test ports** tries 443, 8443, 2053, 2083, 2087, 2096, 8080 and 80. This server listens on each port, the node dials it from abroad, and a port counts as open only when the connection really arrived here. Click an open port to use it. A port this server already uses is reported as such. The node needs this Dalan version; the test checks TCP only, not whether a disguise survives deep inspection.

## Pools of spare tunnels

**Tunnels → Tunnel pools → New pool** groups two or more entry tunnels (different ports, transports or even engines) that stand in for each other, in order of preference. Pick the inbounds whose traffic should leave through the pool; the panel adds a balancer `pool-<name>` that points at the tunnel in use, so the same tag can also be chosen in a routing rule.

Every 15 seconds the panel measures each member (a real request through an Xray tunnel, the engine's own connected state for BackPack). Traffic moves to another member only when that member has been the better choice three checks in a row: the tunnel in use is down, or a measured tunnel is at least 50 ms and 30% faster. One bad check changes nothing. The move needs no Xray restart, and the choice is remembered across restarts. A BackPack member reports no latency, so it takes over only from a tunnel that is down. A tunnel that belongs to a pool cannot be deleted before the pool.

## Traffic per tunnel

The **Traffic** column of an entry tunnel shows what has passed through it, up and down, since Xray last started. The panel switches Xray's outbound statistics on for the generated config by itself; nothing needs changing in the Xray settings. BackPack tunnels show nothing there: the engine counts its own bytes.

## Installing and updating the BackPack engine

On the Tunnels page, **Install BackPack** and **Update engine** fetch the newest BackPack release (**Install** the one this Dalan version pins). The release is checked against the checksums published with it before it replaces the engine, and the engine it replaces is kept. After an update, tunnels that were connected must reconnect within about 75 seconds; if they do not, the earlier engine is put back and the update fails with that reason.

A server in Iran usually cannot reach GitHub. Pick **Download through node** first: the node abroad downloads the release and sends it to this panel through the management tunnel in 512 KiB parts, so nothing depends on GitHub being reachable from here. The node needs this Dalan version. The checksums come from the same node, so this protects against a damaged download, not against a node that is itself compromised.

## Engine output

Everything a BackPack process prints goes to the panel log as `BackPack <tunnel>: ...`; when it exits, the last lines are repeated with the restart notice.

## Not built yet

- One shared port for the management tunnels of all nodes.
- Xray tunnels over transports other than REALITY on TCP.
