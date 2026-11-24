package forwarder

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// sing-box has no remote bind, so it only runs direct: the entry dials the exit over a trojan link whose
// transport is the chosen one, and the exit delivers each forward's target.
var singboxEngine = Engine{
	ID: "singbox", Label: "sing-box", Repo: "SagerNet/sing-box", Pinned: "v1.14.4",
	Transports: []ForwardingTransport{
		{ID: "tcp-tls", Label: "TCP over TLS 1.3", Direct: true},
		{ID: "smux-tls", Label: "TLS 1.3 with smux", Direct: true},
		{ID: "yamux-tls", Label: "TLS 1.3 with yamux", Direct: true},
		{ID: "h2mux-tls", Label: "TLS 1.3 with h2mux", Direct: true},
		{ID: "wss", Label: "WebSocket over TLS", Direct: true},
		{ID: "grpc-tls", Label: "gRPC over TLS", Direct: true},
		{ID: "h2-tls", Label: "HTTP/2 over TLS", Direct: true},
		{ID: "httpupgrade-tls", Label: "HTTP upgrade over TLS", Direct: true},
		{ID: "quic-tls", Label: "QUIC", Direct: true},
	},
	Ext:    ".json",
	Args:   func(config string) []string { return []string{"run", "-c", config} },
	Check:  func(config string) []string { return []string{"check", "-c", config} },
	Render: renderSingbox,
	asset:  singboxAsset,
	Pins: map[string]string{
		"sing-box-1.14.4-linux-amd64.tar.gz": "699a1b510aa4396b2176d32d1181920399ebf1e267835da6f3b598d0f7c24968",
		"sing-box-1.14.4-linux-arm64.tar.gz": "436441f76ee2f7fe21b5b2d426d3cb261a480455a704865789ad533e4273ec26",
	},
}

func singboxAsset(version, arch string) (Asset, error) {
	if arch != "amd64" && arch != "arm64" {
		return Asset{}, fmt.Errorf("sing-box publishes no Linux build for this processor (%s)", arch)
	}
	return Asset{Name: "sing-box-" + strings.TrimPrefix(version, "v") + "-linux-" + arch + ".tar.gz", Member: "sing-box"}, nil
}

var errSingboxReverse = errors.New("sing-box runs direct tunnels only")

type object = map[string]any

// sbTransport is the transport block and multiplexer a transport id stands for.
func sbTransport(id string) (transport object, mux string) {
	switch id {
	case "smux-tls":
		return nil, "smux"
	case "yamux-tls":
		return nil, "yamux"
	case "h2mux-tls":
		return nil, "h2mux"
	case "wss":
		return object{"type": "ws", "path": "/ws"}, ""
	case "grpc-tls":
		return object{"type": "grpc", "service_name": "dalan"}, ""
	case "h2-tls":
		return object{"type": "http", "path": "/h2"}, ""
	case "httpupgrade-tls":
		return object{"type": "httpupgrade", "path": "/up"}, ""
	case "quic-tls":
		return object{"type": "quic"}, ""
	}
	return nil, ""
}

func renderSingbox(p Plan) (string, error) {
	if p.Entry == p.Listens {
		return "", errSingboxReverse
	}
	transport, mux := sbTransport(p.Transport)
	var cfg object
	if p.Listens {
		cert, key, err := certFor(p)
		if err != nil {
			return "", err
		}
		in := object{
			"type": "trojan", "tag": "in", "listen": "0.0.0.0", "listen_port": p.Port,
			"users": []object{{"password": p.Secret}},
			"tls":   object{"enabled": true, "min_version": "1.3", "certificate_path": cert, "key_path": key},
		}
		if transport != nil {
			in["transport"] = transport
		}
		if mux != "" {
			in["multiplex"] = object{"enabled": true}
		}
		cfg = object{"log": object{"level": "warn"}, "inbounds": []object{in}, "outbounds": []object{{"type": "direct", "tag": "out"}}}
	} else {
		var inbounds []object
		for i, f := range p.Forwards {
			if f.UDP {
				return "", errors.New("sing-box carries TCP forwards only")
			}
			host, port, _ := strings.Cut(f.Listen, ":")
			target, targetPort, _ := strings.Cut(f.Target, ":")
			inbounds = append(inbounds, object{
				"type": "direct", "tag": fmt.Sprintf("f%d", i), "network": "tcp", "listen": host, "listen_port": portOf(port),
				"override_address": target, "override_port": portOf(targetPort),
			})
		}
		out := object{
			"type": "trojan", "tag": "out", "server": p.Host, "server_port": p.Port, "password": p.Secret,
			"tls": object{"enabled": true, "min_version": "1.3", "insecure": true, "server_name": "localhost"},
		}
		if transport != nil {
			out["transport"] = transport
		}
		if mux != "" {
			out["multiplex"] = object{"enabled": true, "protocol": mux}
		}
		cfg = object{"log": object{"level": "warn"}, "inbounds": inbounds, "outbounds": []object{out}}
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	return string(raw) + "\n", err
}

// portOf reads a port the forward parser already checked.
func portOf(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
