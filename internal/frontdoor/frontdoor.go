// Package frontdoor puts one HTTP port in front of the panel, the subscription server and the
// plain WebSocket/XHTTP inbounds, for platforms (Railway, Orbit, AWS) that expose a single port.
package frontdoor

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

// Route sends the requests under Prefix to a local inbound.
type Route struct {
	Prefix string
	Host   string
	Port   int
}

// Table says where each kind of request goes; it is read again for every request.
type Table struct {
	Inbounds  []Route
	SubPaths  []string
	SubPort   int
	PanelPath string
	PanelPort int
}

func under(path, prefix string) bool {
	prefix = strings.TrimSuffix(prefix, "/")
	return prefix == "" || path == prefix || strings.HasPrefix(path, prefix+"/")
}

// Resolve finds the local address that serves path: an inbound first (longest path wins), then the
// subscription server, then the panel.
func (t Table) Resolve(path string) (host string, port int, ok bool) {
	best := -1
	for i, r := range t.Inbounds {
		if under(path, r.Prefix) && (best < 0 || len(r.Prefix) > len(t.Inbounds[best].Prefix)) {
			best = i
		}
	}
	if best >= 0 {
		return t.Inbounds[best].Host, t.Inbounds[best].Port, true
	}
	for _, p := range t.SubPaths {
		if t.SubPort > 0 && under(path, p) {
			return "127.0.0.1", t.SubPort, true
		}
	}
	if t.PanelPort > 0 && under(path, t.PanelPath) {
		return "127.0.0.1", t.PanelPort, true
	}
	return "", 0, false
}

// InboundRoutes lists the enabled inbounds a path can reach: WebSocket, HTTPUpgrade and XHTTP without
// TLS of their own, since the platform ends TLS in front of the port.
func InboundRoutes(inbounds []model.Inbound) []Route {
	var routes []Route
	for _, in := range inbounds {
		if !in.Enable || in.Port <= 0 {
			continue
		}
		var stream struct {
			Network  string                `json:"network"`
			Security string                `json:"security"`
			WS       struct{ Path string } `json:"wsSettings"`
			Upgrade  struct{ Path string } `json:"httpupgradeSettings"`
			XHTTP    struct{ Path string } `json:"xhttpSettings"`
		}
		if json.Unmarshal([]byte(in.StreamSettings), &stream) != nil || stream.Security != "none" {
			continue
		}
		var path string
		switch stream.Network {
		case "ws":
			path = stream.WS.Path
		case "httpupgrade":
			path = stream.Upgrade.Path
		case "xhttp":
			path = stream.XHTTP.Path
		}
		path = strings.TrimSuffix(path, "/")
		if path == "" || !strings.HasPrefix(path, "/") {
			continue
		}
		host := in.Listen
		if host == "" || host == "0.0.0.0" || host == "::" {
			host = "127.0.0.1"
		}
		routes = append(routes, Route{Prefix: path, Host: host, Port: in.Port})
	}
	return routes
}

// New serves requests by looking each path up in the table that tables returns.
func New(tables func() Table) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			_, _ = w.Write([]byte("ok"))
			return
		}
		host, port, ok := tables().Resolve(r.URL.Path)
		if !ok {
			if r.URL.Path == "/" {
				_, _ = w.Write([]byte("ok"))
				return
			}
			http.NotFound(w, r)
			return
		}
		target := &url.URL{Scheme: "http", Host: net.JoinHostPort(host, strconv.Itoa(port))}
		// FlushInterval -1 hands each write on at once, which the streaming transports need.
		(&httputil.ReverseProxy{
			Rewrite:       func(pr *httputil.ProxyRequest) { pr.SetURL(target); pr.Out.Host = pr.In.Host },
			FlushInterval: -1,
		}).ServeHTTP(w, r)
	})
}

// Server returns the HTTP server for addr.
func Server(addr string, tables func() Table) *http.Server {
	return &http.Server{Addr: addr, Handler: New(tables), ReadHeaderTimeout: 30 * time.Second}
}
