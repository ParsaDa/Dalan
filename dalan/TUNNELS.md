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
| Engine | BackPack (default) or Rakhsh |
| Transport | How the tunnel looks on the wire; `wssmux` for BackPack, `wss` for Rakhsh |
| Inbounds that leave through this tunnel | The inbounds of this server whose users should exit from the other server |

Then press the copy button on the tunnel's row to copy its **setup link**.

**On the exit server**, open **Tunnels → Import link**, paste the link, and confirm. Leave *Exit outbound* empty to leave directly from that server, or name one of its outbounds (for example a WARP outbound).

The engine starts on each server within 30 seconds and the exit server connects.

## What to know

- The setup link holds the tunnel's secret. Treat it like a password. It does not contain the entry server's own private key.
- A transport such as `tcp` is plain and recognisable. Across a filtered network use a disguised one (`wssmux`, `wss`, `quic`).
- The entry end appears in Xray as an outbound named `tunnel-<name>`. Routing rules and balancers can use that tag.
- For traffic that arrives through a tunnel, the exit server refuses private and loopback destinations. Users cannot reach the exit server's own network.
- Deleting a tunnel on one server does not delete it on the other.

## Two engines

The Xray tunnel engine of earlier versions was removed in 0.6.0; see "Upgrading" below.

**BackPack** is the default, for links that are filtered hard or lose packets. It is a separate program that Dalan installs and runs for you. It appears in Xray as an outbound named `tunnel-<name>` that routing rules, balancers and inbound routes can use. It can also open raw ports on the entry server and carry them to the exit server.

**Rakhsh** is a Rust tunnel core that Dalan installs the same way. It carries several lanes at once over one or more links (each link has its own carrier and address), encrypts every record, can resume a broken session and offers many carriers, from `tcp`, `wss` and `quic` to raw ones. See "A Rakhsh tunnel" below.

### A BackPack tunnel

1. On both servers open **Tunnels**. If a notice says the BackPack engine is not installed, press **Install BackPack**. Dalan downloads the pinned BackPack release (1.9.0), checks it against the checksums published with it, and puts it in the panel's `bin` folder. A server that cannot reach GitHub needs the `backpack` binary copied into that folder by hand.
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

### A Rakhsh tunnel

1. On both servers open **Tunnels**. If a notice says the Rakhsh engine is not installed, press **Install Rakhsh** (the pinned release is `v0.1.0`, checked against the checksums published with it). A server that cannot reach GitHub can use **Download through node** or have the `rakhsh` binary copied into the panel's `bin` folder by hand.
2. On the **entry** server, **New tunnel**, choose **Rakhsh**, pick **Reverse** or **Direct**, a *carrier* and optional *forwarded ports*, then open **Advanced options**.
3. Copy the setup link and import it on the exit server. The link carries the Noise keys and the tunnel secret, so treat it like a password and never post it anywhere.

- *Carrier*: `tcp`, `tls`, `ws`, `wss`, `quic`, `quicv2`, `kcp` and the `shabdiz-udp`/`shabdiz-tcp` carriers work as normal sockets. `faketcp`, `icmp`, `spoof`, `l2tcp` and the other raw carriers forge or wrap packets: they need root on **both** servers, do not work in most containers, and need `peer_ip` (the panel makes the field required when you pick one): enter the address of the server that dials, that is the exit server in a reverse tunnel and the entry server in a direct one. The dialling end is given the listening server's address by itself, the exit end swaps `spoof_src_ip` and `spoof_peer_src`, and `bind` stays on the server where you set it.
- *Forwarded ports*: `listen=target`, for example `443=127.0.0.1:2096`; add `/udp` after the target for UDP. A bare `443` is refused. Traffic of whole inbounds uses *Inbounds* instead, as with BackPack.
- *Reverse* (default): the exit server dials the entry server. *Direct*: the entry server dials the exit server; set `l3_addr` to also get a routed layer-3 device (needs root). Only a reverse tunnel can be an Xray outbound.
- *Links*: each entry has its own carrier, address and, if wanted, settings only that link overrides (`sni`, `ws_path`, `edge_ip` and similar). Links are tried in the order given, so a blocked carrier can fall back to the next. `link_policy` chooses failover or spread.
- *Advanced options*: every setting the panel can write, grouped as Rakhsh groups them (encryption and resume, tunnel, links, pacing, raw carriers, layer 3, limits). Settings that only make sense on the entry end (`limit_*`, `quota_*`) are hidden when you edit an exit end.

A Rakhsh tunnel shows **Connected** with how many of its lanes are up (`3 of 4 lanes`) and the loss the engine measures, both read from the engine's own status socket. When no lane is up the **Down** tag shows the engine's reason on hover.

What it does not do: Dalan does not switch carriers by measured packet loss (the engine's own link policy is all there is), the engine's router sections (`rules`, `fleet`, the OpenVPN helper) are not exposed, and Dalan has not yet run Rakhsh on real servers or across a filtered link. Its own tests ran a real pair on loopback over `tcp` and `wss` only.

### Reverse and direct

Every tunnel is labelled **Reverse** or **Direct** in the Tunnels page and in the Outbounds table.

- **Reverse** (the default): the exit server dials the entry server, so the entry server never needs to reach the exit server. It passes filtering that only blocks inbound connections, works with CDN transports (`ws`, `wss`) and carries Xray traffic.
- **Direct** (BackPack layer 3, BackPack 1.9.0 and newer): the entry server dials the exit server and they exchange IP packets through an encrypted GRE link. It is for games and anything that needs the lowest ping, because the packets are not wrapped in a TCP stream. Choose **BackPack**, then **Direct**, then a *carrier* (`udp`, `pck`, `quic`, `sni` or `xdi`) and the ports to forward (`27015`, or `27015=10.10.7.2:27015`, with *Accept UDP* for UDP games). The panel picks a private `/30` in `10.10.0.0/16` for the link and shows nothing about it; the two ends get mirrored addresses.

A direct tunnel needs root and `/dev/net/tun` on **both** servers, so it does not work in most containers. Its carriers are UDP, raw packets or QUIC, which a CDN does not pass: for a CDN use a reverse tunnel with `ws`/`wss`. A direct tunnel is **not** an Xray outbound (there is no loopback relay in it), so it is not offered in Routing; it carries only the forwarded ports. It shows **Connected** when its link interface (`dl<id>`) is up; the exit server must be reachable on the tunnel's port.

### Options

**Advanced options** in the form lists every BackPack setting the panel can write, grouped as BackPack groups them: logging, limits, performance (`nodelay`, `keepalive_period`, ...), multiplexing, failover, CDN (`edge_ip` and friends, for `ws`/`wss` behind a CDN), and for direct tunnels the carrier and link-quality settings (`mtu`, `fec_data`, `fec_parity`, `bandwidth_mbps`, ...). A field left empty is not written, so the engine keeps its default. The setting travels in the setup link, and each end writes the ones that apply to its role. A few BackPack keys are not offered: the Naive and Xray helper sections, the monitor page and spoofed source addresses (`spoof` carrier).

### Editing

The pencil in the tunnel list changes a tunnel without deleting it. On an entry end you can change the address, port, transport, forwarded ports, options and routed inbounds; the name, secrets, engine and mode stay. When this panel made the other end on a node, the change is sent to the node first and nothing changes if the node refuses. On an exit end only the exit outbound and the options can change; to change a shared setting of a tunnel made by pasting a link, edit the entry end and import its new link on the exit server, which then updates that end instead of failing. A tunnel that carries a node's management cannot have its address, port or transport changed.

## Forwarding engines

Besides BackPack and Rakhsh, 0.7.0 can run a tunnel on six more engines. Each is a program that Dalan installs and supervises, one process per tunnel, and each carries the ports you list from the entry server to a target through the exit server over a loopback relay, so the speed test, the status and the setup links work the same as for a Rakhsh tunnel.

| Engine | Pinned | Carriers | Directions |
|---|---|---|---|
| GOST | v3.3.3 | `tls`, `ws`, `wss`, `grpc`, `h2` | reverse and direct |
| Rathole | v0.5.0 | `tcp`, `noise`, `ws` | reverse |
| Backhaul | v0.6.5 | `tcp`, `tcpmux`, `ws`, `wsmux`, `wss`, `wssmux` | reverse |
| Kariz | v2.8.1 | `tcp`, `tcpmux`, `ws`, `wss`, `quic`, `kcp`, `paqet` | reverse and direct |
| sing-box | v1.14.4 | `tcp-tls`, `smux-tls`, `yamux-tls`, `h2mux-tls`, `wss`, `grpc-tls`, `h2-tls`, `httpupgrade-tls`, `quic-tls` | direct |
| Xray | v26.11.9 | `wss`, `grpc-tls`, `httpupgrade-tls`, `xhttp-tls`, `mkcp-chacha`, `mkcp-aes` | direct |

Each engine has its own card on the Tunnels page with its install and update button (the same checksum check, download route and node download as for the engines above; see "Installing and updating the engines"). A tunnel is made from **New tunnel** by choosing the engine, then one of its carriers. Engine settings, such as Kariz's `profile`, appear in the same dialog.

Both ends share one secret that Dalan generates and carries in the setup link (version 5), so the second server is set up by importing the link, exactly as for Rakhsh. The relay of a forwarding tunnel listens only on loopback, in ports 63400-64399.

`paqet` (Kariz) needs root and an IPv4 address. A tunnel that fails to start shows the engine's own output in the panel log as `<Engine> <tunnel>: ...`.

## Kernel carriers

The **Kernel carriers** engine builds the link in the Linux kernel instead of running a proxy program: a **GRE**, **IPIP** or **VXLAN** interface between the two servers, with a private /30 link in 10.222.0.0/16 chosen from the relay port, and a TCP relay on that link carrying the forwarded ports. There is nothing to download: the panel's own binary runs it (`x-ui carrier <config>`).

It is behind the **Advanced engines** switch on the Tunnels page, because it changes the server's network. It stays visible while any carrier tunnel exists.

- Both servers need **root**, **Linux** and the `ip` command (iproute2). A tunnel is refused with that reason when they are missing, when it is made and when its link is imported.
- Fill in **Public IPv4 address of the end that dials**: the exit's address in a reverse tunnel, the entry's address in a direct one. A tunnel cannot be made without it.
- Open the protocol in the firewall of both servers: IP protocol 47 for GRE, IP protocol 4 for IPIP, and the tunnel's port (UDP) for VXLAN.
- GRE and IPIP have no encryption or authentication of their own, and many hosting networks drop them or cannot NAT them. Prefer a server with a public IPv4 address and test with **Speed test** before relying on it. A Xray relay is protected by a SOCKS login; a carrier link is not.
- Kernel carriers are not part of **Compare methods**, because each needs root and the other end's address.

## Compare methods

**Compare methods** (a button on the Tunnels page, shown when a node is connected) finds out which method works best between this server and a node. For every method you pick it makes a temporary tunnel named `dalan-bench-<n>` on both servers, waits until it connects, runs the speed test through it, and removes it. The result is a table ranked by a score; **Use this method** copies the best row into the new-tunnel dialog.

- Choose the node, this server's address as the node reaches it (reverse methods need it), the direction (reverse, direct or both) and the engines. At most 60 methods run in one run, and a run can take several minutes.
- **Rank by** *Speed* weighs download 70% and upload 30%. *Balanced* weighs throughput 50%, ping 30% and jitter 20%. A method that fails to connect or measure scores 0 and shows the reason.
- One run goes at a time on a panel, and **Stop** cancels it after the current method.
- A temporary tunnel that cannot be removed is listed as *Left behind*. Tunnels named `dalan-bench-*` that a crash left are deleted when the panel starts.
- The routes behind it: `POST /panel/api/tunnels/benchmark/start`, `GET /panel/api/tunnels/benchmark/status` and `POST /panel/api/tunnels/benchmark/cancel`.

## Status

The Tunnels page shows the state each engine reports for every enabled tunnel, on both servers: **Connected** (BackPack has no latency figure; Rakhsh shows how many lanes are up) or **Down**, with the engine's reason on hover. A node's management tunnel is not tested on its own; its state is the node's status in the node list.

## Speed test

The speed test button (the lightning bolt) on an entry tunnel's row measures that tunnel. The test connects through the tunnel's loopback relay, so its connections leave from the exit server: ping, jitter, download and upload are the speed of this tunnel to the internet, against the speedtest.net server closest to the exit server. Run it whenever you like; the last 20 results of each tunnel are kept.

- Only an enabled reverse entry tunnel (BackPack or Rakhsh) can be tested. An exit end, a direct tunnel (it has no relay) and a management tunnel are refused.
- One speed test runs at a time on a panel. Starting the same tunnel again joins the running test; starting another tunnel waits until it ends.
- The exit server must reach speedtest.net. If it cannot, the dialog shows the reason.
- The Speed test card on the dashboard measures this server itself and does not go through any tunnel.

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
   | Engine | BackPack (default) or Rakhsh |
   | Transport | `wssmux` for BackPack, `wss` for Rakhsh unless you pick another |

   **How should it connect?** A new tunnel for this node (the default) makes an entry tunnel named after the node, on its own port. Its management rides on that tunnel as a loopback forward to the node's panel port, so one port serves both, and the tunnel can be picked as an exit in Outbounds, Routing and the inbound form as soon as the node has joined. Deleting that tunnel removes the node's end of it too; it cannot be deleted while the node exists.

   Instead you can pick an existing entry tunnel (the one users ride is marked). The node is then managed over that tunnel, with the same engine, address, port and disguise and nothing new to open; the node must already have its end of that tunnel. A tunnel that carries a node cannot be deleted until that node is removed; if it goes down, the node's management goes down with it.

2. Press **Create command** and copy the command. It is shown once, works once and expires after 30 minutes.
3. On the node, run it:

   ```bash
   dalan enrol 'dalan://enrol/...'
   ```

   The node starts its tunnel and appears online in the main panel's node list within a minute.

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

The **Traffic** column of an entry tunnel shows what has passed through it, up and down, since Xray last started. The panel switches Xray's outbound statistics on for the generated config by itself; nothing needs changing in the Xray settings. BackPack direct tunnels show nothing there; a BackPack reverse tunnel shows what its outbound carried.

## Installing and updating the engines

On the Tunnels page, **Install BackPack**, **Install Rakhsh** and each engine's **Update engine** fetch the newest release of that engine (**Install** the one this Dalan version pins: BackPack 1.9.0, Rakhsh 0.1.0). The page shows one bar per engine that a tunnel on it uses. The release is checked against the checksums published with it before it replaces the engine, and the engine it replaces is kept. After an update, tunnels that were connected must reconnect within about 75 seconds; if they do not, the earlier engine is put back and the update fails with that reason.

## Download route

Every install and update button on this server (Tunnels, and the Tor window) has a **Download route** menu beside it. It decides how the release travels from the release server to this panel:

- **As the panel is set** (default): through the panel's own outbound setting, direct when none is set.
- **Direct from this server**: no proxy, whatever the panel setting says.
- **An outbound of the panel**: any outbound of the running Xray, listed by its tag. Tunnels (`tunnel-<name>`) and Tor exits (`tor-<cc>`) are in the list, so a server in Iran can download through a tunnel to a server abroad. The panel starts a short-lived copy of that outbound for the download and stops it afterwards.
- **A proxy address or share link**: type `socks5://host:port` or `http://host:port`, or paste a `vless://`, `vmess://`, `trojan://` or `ss://` link. A share link runs in a short-lived core for the download only; it is not saved anywhere.

The install button waits until a link is typed. **Download through node** (Tunnels only) is a separate way and hides the menu, because the node downloads as its own panel is set.

The fresh-install step `x-ui install-engines` reads the environment variable `XUI_ENGINE_PROXY` (a proxy address or share link) for a server that cannot reach GitHub: `XUI_ENGINE_PROXY=socks5://127.0.0.1:1080 x-ui install-engines`.

A temporary core needs Xray to be installed on the panel server; an outbound that is only in the saved template is used when Xray is not running.

A server in Iran usually cannot reach GitHub. Pick **Download through node** (or a route above) first: the node abroad downloads the release and sends it to this panel through the management tunnel in 512 KiB parts, so nothing depends on GitHub being reachable from here. The node needs this Dalan version. The checksums come from the same node, so this protects against a damaged download, not against a node that is itself compromised.

## Engine output

Everything a BackPack or Rakhsh process prints goes to the panel log as `BackPack <tunnel>: ...` or `Rakhsh <tunnel>: ...`; when it exits, the last lines are repeated with the restart notice.

## Not built yet

- Loss-aware switching between carriers or tunnels (planned for a later version).
- Rakhsh's router sections.
- A direct tunnel as an Xray outbound, and a direct tunnel through a CDN (neither is possible with a layer-3 link).
- BackPack's `spoof` carrier and its Naive and Xray helper sections.
- One shared port for the management tunnels of all nodes.

## Upgrading from a version with Xray tunnels

Version 0.6.0 removes the Xray tunnel engine. On the first start after the upgrade Dalan deletes every Xray tunnel, the management tunnels that rode on them and their places in spare pools; traffic that went through them stops until you make a BackPack or Rakhsh tunnel. A node that joined through an Xray tunnel stays in the node list with a notice, **must be enrolled again**. Press **Enrol again**, run the new command on the node, and the same node record (its inbounds and history) is pointed at the new tunnel. The node's engine binary has to be installed first (`x-ui install-engines`).
