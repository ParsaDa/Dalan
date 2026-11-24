package forwarder

import (
	"crypto/ecdh"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

// Rathole carries ports from the entry end to the exit end over one control connection; it only runs
// in reverse, because its server is always the end that exposes the ports.
var ratholeEngine = Engine{
	ID: "rathole", Label: "Rathole", Repo: "rathole-org/rathole", Pinned: "v0.5.0",
	Transports: []Transport{
		{ID: "tcp", Label: "TCP", Reverse: true},
		{ID: "noise", Label: "Noise (encrypted)", Reverse: true},
		{ID: "ws", Label: "WebSocket", Reverse: true},
	},
	Ext:    ".toml",
	Args:   func(config string) []string { return []string{config} },
	Render: renderRathole,
	asset:  ratholeAsset,
	Pins: map[string]string{
		"rathole-x86_64-unknown-linux-gnu.zip":   "3e7d0d0f365120cd3cd351d147d1a12ee960c8068b464d4dd533a3821873b80e",
		"rathole-aarch64-unknown-linux-musl.zip": "fa4a6fc63d86f8f1faa7c103a845e4715ce79a048455c0eec897b27237576564",
	},
}

func ratholeAsset(_, arch string) (Asset, error) {
	switch arch {
	case "amd64":
		return Asset{Name: "rathole-x86_64-unknown-linux-gnu.zip", Member: "rathole"}, nil
	case "arm64":
		return Asset{Name: "rathole-aarch64-unknown-linux-musl.zip", Member: "rathole"}, nil
	}
	return Asset{}, fmt.Errorf("Rathole publishes no Linux build for this processor (%s)", arch)
}

// noiseKeys derives the Noise key pair both ends know from the tunnel secret, so no key has to travel.
func noiseKeys(secret string) (private, public string, err error) {
	seed := sha256.Sum256([]byte("dalan-rathole-noise:" + secret))
	key, err := ecdh.X25519().NewPrivateKey(seed[:])
	if err != nil {
		return "", "", err
	}
	return base64.StdEncoding.EncodeToString(key.Bytes()), base64.StdEncoding.EncodeToString(key.PublicKey().Bytes()), nil
}

func ratholeServiceName(i int) string { return "svc" + strconv.Itoa(i) }

func renderRathole(p Plan) (string, error) {
	var b strings.Builder
	transport := map[string]string{"tcp": "tcp", "noise": "noise", "ws": "websocket"}[p.Transport]
	private, public, err := noiseKeys(p.Secret)
	if err != nil {
		return "", err
	}
	if p.Listens {
		fmt.Fprintf(&b, "[server]\nbind_addr = \"0.0.0.0:%d\"\ndefault_token = %s\nheartbeat_interval = 30\n", p.Port, strconv.Quote(p.Secret))
		fmt.Fprintf(&b, "\n[server.transport]\ntype = %s\n", strconv.Quote(transport))
		if p.Transport == "noise" {
			fmt.Fprintf(&b, "\n[server.transport.noise]\nlocal_private_key = %s\n", strconv.Quote(private))
		}
		if p.Transport == "ws" {
			b.WriteString("\n[server.transport.websocket]\ntls = false\n")
		}
		for i, f := range p.Forwards {
			if f.UDP {
				fmt.Fprintf(&b, "\n[server.services.%s]\ntype = \"udp\"\nbind_addr = %s\n", ratholeServiceName(i), strconv.Quote(f.Listen))
				continue
			}
			fmt.Fprintf(&b, "\n[server.services.%s]\nbind_addr = %s\n", ratholeServiceName(i), strconv.Quote(f.Listen))
		}
		return b.String(), nil
	}
	fmt.Fprintf(&b, "[client]\nremote_addr = %s\ndefault_token = %s\nheartbeat_timeout = 40\n",
		strconv.Quote(fmt.Sprintf("%s:%d", p.Host, p.Port)), strconv.Quote(p.Secret))
	fmt.Fprintf(&b, "\n[client.transport]\ntype = %s\n", strconv.Quote(transport))
	if p.Transport == "noise" {
		fmt.Fprintf(&b, "\n[client.transport.noise]\nremote_public_key = %s\n", strconv.Quote(public))
	}
	if p.Transport == "ws" {
		b.WriteString("\n[client.transport.websocket]\ntls = false\n")
	}
	for i, f := range p.Forwards {
		kind := ""
		if f.UDP {
			kind = "type = \"udp\"\n"
		}
		fmt.Fprintf(&b, "\n[client.services.%s]\n%slocal_addr = %s\n", ratholeServiceName(i), kind, strconv.Quote(f.Target))
	}
	return b.String(), nil
}
