package forwarder

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Backhaul keeps one control connection from the exit end to the entry end and opens the entry's ports
// over it; like Rathole it only runs in reverse.
var backhaulEngine = Engine{
	ID: "backhaul", Label: "Backhaul", Repo: "Musixal/Backhaul", Pinned: "v0.6.5",
	Transports: []ForwardingTransport{
		{ID: "tcp", Label: "TCP", Reverse: true},
		{ID: "tcpmux", Label: "TCP multiplexed", Reverse: true},
		{ID: "ws", Label: "WebSocket", Reverse: true},
		{ID: "wsmux", Label: "WebSocket multiplexed", Reverse: true},
		{ID: "wss", Label: "WebSocket over TLS", Reverse: true},
		{ID: "wssmux", Label: "WebSocket over TLS, multiplexed", Reverse: true},
	},
	Ext:    ".toml",
	Args:   func(config string) []string { return []string{"-c", config} },
	Render: renderBackhaul,
	asset:  backhaulAsset,
	Pins: map[string]string{
		"backhaul_linux_amd64.tar.gz": "665f76036a43dca9fc4d4d44d357f228cbf73926b93dfd115de4625d0bdb92aa",
		"backhaul_linux_arm64.tar.gz": "891f60b0ed88d3bbc9cd3480101075dc86fc4722b3bdd4dc8f2306c210a4eebb",
	},
}

func backhaulAsset(_, arch string) (Asset, error) {
	if arch != "amd64" && arch != "arm64" {
		return Asset{}, fmt.Errorf("Backhaul publishes no Linux build for this processor (%s)", arch)
	}
	return Asset{Name: "backhaul_linux_" + arch + ".tar.gz", Member: "backhaul"}, nil
}

var errBackhaulUDP = errors.New("Backhaul carries UDP only on the tcp transport")

func renderBackhaul(p Plan) (string, error) {
	var b strings.Builder
	if p.Listens {
		fmt.Fprintf(&b, "[server]\nbind_addr = \"0.0.0.0:%d\"\ntransport = %s\ntoken = %s\nlog_level = \"warn\"\nnodelay = true\nkeepalive_period = 75\n",
			p.Port, strconv.Quote(p.Transport), strconv.Quote(p.Secret))
		ports := make([]string, 0, len(p.Forwards))
		udp := false
		for _, f := range p.Forwards {
			ports = append(ports, strconv.Quote(f.Listen+"="+f.Target))
			udp = udp || f.UDP
		}
		fmt.Fprintf(&b, "ports = [%s]\n", strings.Join(ports, ", "))
		if udp {
			if p.Transport != "tcp" {
				return "", errBackhaulUDP
			}
			b.WriteString("accept_udp = true\n")
		}
		if strings.HasPrefix(p.Transport, "wss") {
			cert, key, err := certFor(p)
			if err != nil {
				return "", err
			}
			fmt.Fprintf(&b, "tls_cert = %s\ntls_key = %s\n", strconv.Quote(cert), strconv.Quote(key))
		}
		return b.String(), nil
	}
	fmt.Fprintf(&b, "[client]\nremote_addr = %s\ntransport = %s\ntoken = %s\nlog_level = \"warn\"\nnodelay = true\nkeepalive_period = 75\nretry_interval = 3\n",
		strconv.Quote(fmt.Sprintf("%s:%d", p.Host, p.Port)), strconv.Quote(p.Transport), strconv.Quote(p.Secret))
	return b.String(), nil
}
