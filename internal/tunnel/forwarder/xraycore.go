package forwarder

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
)

// Xray here is a standalone xray-core process (not the panel's own Xray): the entry's dokodemo forwards
// dial the exit's inbound, and the exit lets each connection reach only the targets of the forwards.
var xrayCoreEngine = Engine{
	ID: "xraycore", Label: "Xray", Repo: "XTLS/Xray-core", Pinned: "v26.11.9",
	Transports: []ForwardingTransport{
		{ID: "wss", Label: "WebSocket over TLS", Direct: true},
		{ID: "grpc-tls", Label: "gRPC over TLS", Direct: true},
		{ID: "httpupgrade-tls", Label: "HTTP upgrade over TLS", Direct: true},
		{ID: "xhttp-tls", Label: "XHTTP over TLS", Direct: true},
		{ID: "mkcp-chacha", Label: "mKCP, ChaCha20-Poly1305", Direct: true},
		{ID: "mkcp-aes", Label: "mKCP, AES-128-GCM", Direct: true},
	},
	Ext:    ".json",
	Args:   func(config string) []string { return []string{"run", "-config", config} },
	Check:  func(config string) []string { return []string{"run", "-test", "-config", config} },
	Render: renderXrayCore,
	asset:  xrayCoreAsset,
	Pins: map[string]string{
		"Xray-linux-64.zip":        "b651f7e1d9966ee2c1b1ee1bf8b52c72771824d3065dce96bbc8bcc19ed95f82",
		"Xray-linux-arm64-v8a.zip": "64ed6858982a7f5eb85d88b811089063bd4f17561f4c8f44e0f1dc8e6f6430be",
	},
}

func xrayCoreAsset(_, arch string) (Asset, error) {
	name := map[string]string{"amd64": "Xray-linux-64.zip", "arm64": "Xray-linux-arm64-v8a.zip"}[arch]
	if name == "" {
		return Asset{}, fmt.Errorf("Xray publishes no Linux build for this processor (%s)", arch)
	}
	return Asset{Name: name, Member: "xray", SumsName: name + ".dgst"}, nil
}

// uuidOf derives the account id both ends use from the tunnel secret.
func uuidOf(secret string) string {
	h := sha256.Sum256([]byte("dalan-xray-id:" + secret))
	h[6] = h[6]&0x0f | 0x50
	h[8] = h[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", h[0:4], h[4:6], h[6:8], h[8:10], h[10:16])
}

func xrayStream(p Plan) (object, bool, error) {
	kcp := strings.HasPrefix(p.Transport, "mkcp-")
	if kcp {
		return object{"network": "kcp", "kcpSettings": object{
			"mtu": 1350, "tti": 20, "uplinkCapacity": 100, "downlinkCapacity": 100,
			"congestion": true, "readBufferSize": 16, "writeBufferSize": 16, "seed": p.Secret,
		}}, true, nil
	}
	stream := object{"security": "tls"}
	switch p.Transport {
	case "wss":
		stream["network"], stream["wsSettings"] = "ws", object{"path": "/ws"}
	case "grpc-tls":
		stream["network"], stream["grpcSettings"] = "grpc", object{"serviceName": "dalan"}
	case "httpupgrade-tls":
		stream["network"], stream["httpupgradeSettings"] = "httpupgrade", object{"path": "/up"}
	case "xhttp-tls":
		stream["network"], stream["xhttpSettings"] = "xhttp", object{"path": "/x", "mode": "auto"}
	default:
		return nil, false, fmt.Errorf("Xray has no transport %q", p.Transport)
	}
	if p.Listens {
		cert, key, err := derivedFiles(p)
		if err != nil {
			return nil, false, err
		}
		stream["tlsSettings"] = object{"minVersion": "1.3", "certificates": []object{{"certificateFile": cert, "keyFile": key}}}
	} else {
		d, err := DeriveCert(p.Secret)
		if err != nil {
			return nil, false, err
		}
		stream["tlsSettings"] = object{"minVersion": "1.3", "serverName": "localhost", "fingerprint": "unsafe", "pinnedPeerCertSha256": d.Pin}
	}
	return stream, false, nil
}

var errXrayReverse = errors.New("Xray runs direct tunnels only")

func renderXrayCore(p Plan) (string, error) {
	if p.Entry == p.Listens {
		return "", errXrayReverse
	}
	stream, vmess, err := xrayStream(p)
	if err != nil {
		return "", err
	}
	id := uuidOf(p.Secret)
	var cfg object
	if p.Listens {
		settings, protocol := object{"clients": []object{{"id": id}}, "decryption": "none"}, "vless"
		if vmess {
			settings, protocol = object{"clients": []object{{"id": id}}}, "vmess"
		}
		rules := []object{}
		for _, f := range p.Forwards {
			host, port, _ := net.SplitHostPort(f.Target)
			rule := object{"type": "field", "inboundTag": []string{"tunnel"}, "port": port, "outboundTag": "backend"}
			if net.ParseIP(host) != nil {
				rule["ip"] = []string{host}
			} else {
				rule["domain"] = []string{"full:" + host}
			}
			rules = append(rules, rule)
		}
		rules = append(rules, object{"type": "field", "inboundTag": []string{"tunnel"}, "outboundTag": "blocked"})
		cfg = object{
			"log":      object{"loglevel": "warning"},
			"inbounds": []object{{"tag": "tunnel", "listen": "0.0.0.0", "port": p.Port, "protocol": protocol, "settings": settings, "streamSettings": stream}},
			"outbounds": []object{
				{"tag": "backend", "protocol": "freedom", "settings": object{"finalRules": []object{{"action": "allow"}}}},
				{"tag": "blocked", "protocol": "blackhole"},
			},
			"routing": object{"rules": rules},
		}
	} else {
		var inbounds []object
		for i, f := range p.Forwards {
			listenHost, listenPort, _ := net.SplitHostPort(f.Listen)
			targetHost, targetPort, _ := net.SplitHostPort(f.Target)
			network := "tcp"
			if f.UDP {
				network = "tcp,udp"
			}
			inbounds = append(inbounds, object{
				"tag": fmt.Sprintf("f%d", i), "listen": listenHost, "port": portOf(listenPort), "protocol": "dokodemo-door",
				"settings": object{"address": targetHost, "port": portOf(targetPort), "network": network},
			})
		}
		user, protocol := object{"id": id, "encryption": "none"}, "vless"
		if vmess {
			cipher := map[string]string{"mkcp-chacha": "chacha20-poly1305", "mkcp-aes": "aes-128-gcm"}[p.Transport]
			user, protocol = object{"id": id, "security": cipher}, "vmess"
		}
		cfg = object{
			"log":      object{"loglevel": "warning"},
			"inbounds": inbounds,
			"outbounds": []object{{
				"tag": "tunnel", "protocol": protocol, "streamSettings": stream,
				"settings": object{"vnext": []object{{"address": p.Host, "port": p.Port, "users": []object{user}}}},
			}},
		}
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	return string(raw) + "\n", err
}
