package forwarder

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// GOST runs the exit's relay server and the entry's forwards over one chain; reverse tunnels use its
// remote-bind (rtcp) forwards, direct ones its plain tcp forwards through the relay.
var gostEngine = Engine{
	ID: "gost", Label: "GOST", Repo: "go-gost/gost", Pinned: "v3.3.3",
	Transports: []ForwardingTransport{
		{ID: "tls", Label: "TLS", Reverse: true, Direct: true},
		{ID: "ws", Label: "WebSocket", Reverse: true, Direct: true},
		{ID: "wss", Label: "WebSocket over TLS", Reverse: true, Direct: true},
		{ID: "grpc", Label: "gRPC", Reverse: true, Direct: true},
		{ID: "h2", Label: "HTTP/2", Reverse: true, Direct: true},
	},
	Ext:    ".yml",
	Args:   func(config string) []string { return []string{"-C", config} },
	Render: renderGost,
	asset:  gostAsset,
	Pins: map[string]string{
		"gost_3.3.3_linux_amd64.tar.gz": "f4c1255dc892a7014c0ad6e5f1c3573108f7bea4403f04f21873406ab58aa666",
		"gost_3.3.3_linux_arm64.tar.gz": "4c704c92df6eeea8a3b08d0de4fd3c44343095a20136b3b6cf334f98d2f174ab",
	},
}

func gostAsset(version, arch string) (Asset, error) {
	if arch != "amd64" && arch != "arm64" {
		return Asset{}, fmt.Errorf("GOST publishes no Linux build for this processor (%s)", arch)
	}
	v := strings.TrimPrefix(version, "v")
	return Asset{
		Name: "gost_" + v + "_linux_" + arch + ".tar.gz", Member: "gost",
		SumsName: "gost_" + v + "_checksums.txt",
	}, nil
}

var errGostUDP = errors.New("GOST carries TCP forwards only")

const gostUser = "dalan"

func gostAuth(secret string) string {
	return fmt.Sprintf("{username: %s, password: %s}", strconv.Quote(gostUser), strconv.Quote(secret))
}

func renderGost(p Plan) (string, error) {
	for _, f := range p.Forwards {
		if f.UDP {
			return "", errGostUDP
		}
	}
	var b strings.Builder
	b.WriteString("services:\n")
	if p.Listens {
		fmt.Fprintf(&b, "- name: relay\n  addr: \":%d\"\n  handler:\n    type: relay\n    auth: %s\n    metadata:\n      bind: true\n  listener:\n    type: %s\n",
			p.Port, gostAuth(p.Secret), p.Transport)
		return b.String(), nil
	}
	reverse := !p.Entry
	for i, f := range p.Forwards {
		if reverse {
			fmt.Fprintf(&b, "- name: f%d\n  addr: %s\n  handler:\n    type: rtcp\n  listener:\n    type: rtcp\n    chain: chain-0\n  forwarder:\n    nodes:\n    - name: target\n      addr: %s\n",
				i, strconv.Quote(f.Listen), strconv.Quote(f.Target))
			continue
		}
		fmt.Fprintf(&b, "- name: f%d\n  addr: %s\n  handler:\n    type: tcp\n    chain: chain-0\n  listener:\n    type: tcp\n  forwarder:\n    nodes:\n    - name: target\n      addr: %s\n",
			i, strconv.Quote(f.Listen), strconv.Quote(f.Target))
	}
	fmt.Fprintf(&b, "chains:\n- name: chain-0\n  hops:\n  - name: hop-0\n    nodes:\n    - name: exit\n      addr: %s\n      connector:\n        type: relay\n        auth: %s\n      dialer:\n        type: %s\n        tls:\n          secure: false\n",
		strconv.Quote(fmt.Sprintf("%s:%d", p.Host, p.Port)), gostAuth(p.Secret), p.Transport)
	return b.String(), nil
}
