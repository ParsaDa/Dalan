package forwarder

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Kariz is a Rust tunnel with seven transports, UDP, and both directions; the entry owns the forwards.
var karizEngine = Engine{
	ID: "kariz", Label: "Kariz", Repo: "Erfan-XRay/Kariz", Pinned: "v2.8.1",
	Transports: []Transport{
		{ID: "tcp", Label: "TCP", Reverse: true, Direct: true},
		{ID: "tcpmux", Label: "TCP multiplexed", Reverse: true, Direct: true},
		{ID: "ws", Label: "WebSocket", Reverse: true, Direct: true},
		{ID: "wss", Label: "WebSocket over TLS", Reverse: true, Direct: true},
		{ID: "quic", Label: "QUIC", Reverse: true, Direct: true},
		{ID: "kcp", Label: "KCP", Reverse: true, Direct: true},
		{ID: "paqet", Label: "Paqet (KCP in raw TCP, IPv4, root)", Reverse: true, Direct: true},
	},
	Ext:    ".toml",
	Args:   func(config string) []string { return []string{"run", "-c", config} },
	Check:  func(config string) []string { return []string{"check", "-c", config} },
	Render: renderKariz,
	asset:  karizAsset,
	Pins: map[string]string{
		"kariz-v2.8.1-x86_64-linux.tar.gz":  "9239e6c24d826a2b6ccedff4ad51dd1b895a2d3bdcab75bd8072010bfcb20057",
		"kariz-v2.8.1-aarch64-linux.tar.gz": "ca169377ab754bd3d6cbdff7c751d71d300e8d5bbab87289d9743305670fde1a",
	},
}

func karizAsset(version, arch string) (Asset, error) {
	cpu := map[string]string{"amd64": "x86_64", "arm64": "aarch64"}[arch]
	if cpu == "" {
		return Asset{}, fmt.Errorf("Kariz publishes no build for this processor (%s)", arch)
	}
	return Asset{Name: "kariz-" + version + "-" + cpu + "-linux.tar.gz", Member: "kariz"}, nil
}

var karizProfiles = []string{"balanced", "ultraspeed", "gaming"}

var errKarizUDP = errors.New("Kariz carries UDP forwards only on a multiplexed transport: tcpmux, ws, wss, quic, kcp or paqet")

func renderKariz(p Plan) (string, error) {
	profile := "balanced"
	if v, ok := p.Options["profile"].(string); ok && v != "" {
		if !slices.Contains(karizProfiles, v) {
			return "", fmt.Errorf("Kariz has no profile %q", v)
		}
		profile = v
	}
	role, mode := "exit", "reverse"
	if p.Entry {
		role = "entry"
	}
	// Reverse: the exit dials the entry. Direct: the entry dials the exit.
	if p.Entry != p.Listens {
		mode = "direct"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "role = %q\nmode = %q\nprofile = %q\n\n[tunnel]\ntransport = %q\ntoken = %q\n", role, mode, profile, p.Transport, p.Secret)
	if p.Listens {
		fmt.Fprintf(&b, "listen = %q\n", fmt.Sprintf("0.0.0.0:%d", p.Port))
	} else {
		fmt.Fprintf(&b, "remote = %q\n", fmt.Sprintf("%s:%d", p.Host, p.Port))
	}
	if p.Transport == "ws" || p.Transport == "wss" {
		b.WriteString("\n[tunnel.ws]\npath = \"/ws\"\n")
	}
	if p.Transport == "wss" {
		b.WriteString("\n[tunnel.tls]\n")
		if p.Listens {
			cert, key, err := certFor(p)
			if err != nil {
				return "", err
			}
			fmt.Fprintf(&b, "cert = %s\nkey = %s\n", strconv.Quote(cert), strconv.Quote(key))
		} else {
			b.WriteString("insecure = true\n")
		}
	}
	if p.Entry {
		for _, f := range p.Forwards {
			protocol := "tcp"
			if f.UDP {
				if p.Transport == "tcp" {
					return "", errKarizUDP
				}
				protocol = "udp"
			}
			fmt.Fprintf(&b, "\n[[forward]]\nlisten = %s\ntarget = %s\nprotocol = %q\n", strconv.Quote(f.Listen), strconv.Quote(f.Target), protocol)
		}
	}
	b.WriteString("\n[log]\nlevel = \"warn\"\n")
	return b.String(), nil
}
