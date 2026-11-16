# Running Dalan in a container (Railway, Orbit, AWS)

The image is `ghcr.io/parsada/dalan:<version>` (the `Dockerfile` in the repo root). It runs the same panel as on a server, with one addition for platforms that expose a single HTTP port: the **front door**.

## The front door

When the platform sets `PORT` (Railway, Orbit and most others do), the container opens that port and sends each request to the right place by its path:

| Path | Goes to |
| --- | --- |
| the path of a WebSocket, HTTPUpgrade or XHTTP inbound with security `none` | that inbound |
| the subscription paths (`/sub/`, JSON, Clash) | the subscription server |
| the panel's base path | the panel |
| `/healthz`, and `/` when the panel has its own base path | `200 ok`, for health checks |

The table is read from the panel's own settings and inbounds, so an inbound added in the panel is reachable within about five seconds, with no environment variable to edit. The panel itself moves to an internal port (`XUI_PORT`, 20530 unless set).

What cannot go through one HTTP port: raw TCP inbounds, REALITY, gRPC and anything with its own TLS. Use WebSocket, HTTPUpgrade or XHTTP with security `none`; the platform ends TLS in front of the container. fail2ban is switched off in this mode because the platform does not allow iptables.

## Settings

| Variable | Meaning |
| --- | --- |
| `PORT` | Set by the platform. The front door listens here. `DALAN_FRONT_PORT` does the same and wins when both are set. |
| `DALAN_ADMIN_USER`, `DALAN_ADMIN_PASSWORD` | Panel login, applied at every start. Both are needed. Without them a login is generated once and printed in the log. |
| `DALAN_BASE_PATH` | Panel base path, for example `/dash/`. |
| `DALAN_ENROL` | A `dalan://enrol/...` link from the main panel. The container joins it at its first start and never uses the link twice. |
| `XUI_DB_FOLDER` | Where the database lives (default `/etc/x-ui`). |

Attach a **volume at `/etc/x-ui`**. Without one the database starts empty at every deploy, and a used enrolment link cannot join again.

## A node abroad on Railway

1. In the main panel (Iran): **Nodes → Add node with a command**, choose the tunnel port and create the command. Copy the `dalan://enrol/...` link from it.
2. In Railway: new service from the image `ghcr.io/parsada/dalan:<version>`, add a volume at `/etc/x-ui`, set `DALAN_ADMIN_USER`, `DALAN_ADMIN_PASSWORD`, `DALAN_BASE_PATH` and `DALAN_ENROL`, generate a public domain.
3. Deploy. The node joins the main panel and reaches it through the tunnel; inbounds pushed from the main panel appear on its domain over WebSocket or XHTTP.

The link lives 30 minutes, so create the command right before deploying. If joining fails the container still starts and tries again at the next start with the same link, until it expires.

## Orbit and AWS

The same image and variables apply. On AWS (ECS, App Runner, Lightsail containers) set the service port to the one the platform passes as `PORT`, or set `DALAN_FRONT_PORT` yourself, and mount persistent storage at `/etc/x-ui`.
