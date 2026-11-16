package rakhsh

import (
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

const (
	portalSecret = "IiIiIiIiIiIiIiIiIiIiIiIiIiIiIiIiIiIiIiIiIiI="
	portalPublic = "MzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzM="
	bridgeSecret = "VVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVVU="
	bridgePublic = "REREREREREREREREREREREREREREREREREREREREREQ="
	// base64(sha256("dalan-rakhsh:" + tunnelUUID)), worked out outside Go.
	tunnelToken = "BqnIJJX2I/Rw0PMD6USAQSzM9LktsLrRZGFyXi+KeCo="
	tunnelUUID  = "8d6f3a0e-5c1b-4f0e-9a51-2f6d1c7b9e10"
	socketPath  = "/run/dalan/r1.rakhsh.sock"
)

func reverseOptions() map[string]any {
	return map[string]any{
		"encryption": "full",
		"resume":     true,
		"lanes":      4,
		"preset":     "balance",
		"sni":        "www.example.com",
		"limit_up":   "100mbit",
		"links": []any{
			map[string]any{"carrier": "quic", "addr": "0.0.0.0:7443", "quic_mask": true, "sni": "www.microsoft.com"},
			map[string]any{"carrier": "ws", "addr": "0.0.0.0:8080", "ws_path": "/ws"},
		},
	}
}

func portalTunnel() model.ServerTunnel {
	return model.ServerTunnel{
		Name: "r1", Role: model.TunnelRolePortal, Engine: model.TunnelEngineRakhsh, Mode: model.TunnelModeReverse,
		Port: 7000, Transport: "wss", UUID: tunnelUUID, RelayPort: 63500,
		RakhshSecret: portalSecret, RakhshPeerPublic: bridgePublic,
		ForwardPorts: []string{"443=127.0.0.1:2096", "51820=127.0.0.1:51820/udp"},
		Options:      reverseOptions(),
	}
}

func bridgeTunnel() model.ServerTunnel {
	t := portalTunnel()
	t.Role = model.TunnelRoleBridge
	t.Address = "203.0.113.10"
	t.RakhshSecret, t.RakhshPeerPublic = bridgeSecret, portalPublic
	t.ForwardPorts = nil
	return t
}

func TestRenderConfigReversePortal(t *testing.T) {
	want := `role = "entry"
dial = "exit"
tunnel_listen = "0.0.0.0:7000"
carrier = "wss"
token = "` + tunnelToken + `"
static_secret = "` + portalSecret + `"
peer_public = "` + bridgePublic + `"
control_socket = "` + socketPath + `"
encryption = "full"
resume = true
lanes = 4
preset = "balance"
sni = "www.example.com"
limit_up = "100mbit"

[[link]]
carrier = "quic"
addr = "0.0.0.0:7443"
sni = "www.microsoft.com"
quic_mask = true

[[link]]
carrier = "ws"
addr = "0.0.0.0:8080"
ws_path = "/ws"

[[forward]]
listen = "0.0.0.0:443"
target = "127.0.0.1:2096"

[[forward]]
listen = "0.0.0.0:51820"
target = "127.0.0.1:51820"
udp = true

[[forward]]
listen = "127.0.0.1:63500"
target = "127.0.0.1:63500"
`
	got, err := RenderConfig(portalTunnel(), socketPath)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("portal config differs\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestRenderConfigReverseBridgeDialsAndHasNoForwardsOrEntryLimits(t *testing.T) {
	want := `role = "exit"
dial = "exit"
tunnel_connect = "203.0.113.10:7000"
carrier = "wss"
token = "` + tunnelToken + `"
static_secret = "` + bridgeSecret + `"
peer_public = "` + portalPublic + `"
control_socket = "` + socketPath + `"
encryption = "full"
resume = true
lanes = 4
preset = "balance"
sni = "www.example.com"

[[link]]
carrier = "quic"
addr = "203.0.113.10:7443"
sni = "www.microsoft.com"
quic_mask = true

[[link]]
carrier = "ws"
addr = "203.0.113.10:8080"
ws_path = "/ws"
`
	got, err := RenderConfig(bridgeTunnel(), socketPath)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("bridge config differs\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestRenderConfigDirectSwapsWhoDials(t *testing.T) {
	portal, bridge := portalTunnel(), bridgeTunnel()
	portal.Mode, bridge.Mode = model.TunnelModeDirect, model.TunnelModeDirect
	portal.Address = "203.0.113.20"
	portal.Options, bridge.Options = nil, nil

	gotPortal, err := RenderConfig(portal, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{`role = "entry"`, `dial = "entry"`, `tunnel_connect = "203.0.113.20:7000"`} {
		if !strings.Contains(gotPortal, line+"\n") {
			t.Errorf("direct portal config lacks %q:\n%s", line, gotPortal)
		}
	}
	if strings.Contains(gotPortal, "tunnel_listen") || strings.Contains(gotPortal, "control_socket") {
		t.Errorf("direct portal must not listen, and an empty socket path writes no control_socket:\n%s", gotPortal)
	}

	gotBridge, err := RenderConfig(bridge, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{`role = "exit"`, `dial = "entry"`, `tunnel_listen = "0.0.0.0:7000"`} {
		if !strings.Contains(gotBridge, line+"\n") {
			t.Errorf("direct bridge config lacks %q:\n%s", line, gotBridge)
		}
	}
	if strings.Contains(gotBridge, "tunnel_connect") {
		t.Errorf("direct bridge must not dial:\n%s", gotBridge)
	}
}

func TestRenderConfigMirrorsTheLayerThreeAddress(t *testing.T) {
	portal, bridge := portalTunnel(), bridgeTunnel()
	portal.ForwardPorts, portal.RelayPort = nil, 0
	opts := map[string]any{"l3_addr": "10.77.0.1/30"}
	portal.Options, bridge.Options = opts, opts

	gotPortal, err := RenderConfig(portal, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotPortal, "l3_addr = \"10.77.0.1/30\"\n") {
		t.Errorf("entry keeps the address it was given:\n%s", gotPortal)
	}
	gotBridge, err := RenderConfig(bridge, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotBridge, "l3_addr = \"10.77.0.2/30\"\n") {
		t.Errorf("exit takes the next address:\n%s", gotBridge)
	}
}

func TestMirrorL3Addr(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"10.77.0.1/30", "10.77.0.2/30"},
		{"10.77.0.2/30", "10.77.0.3/30"},
		{"10.10.7.255/24", ""},
		{"10.77.0.1", "10.77.0.2"},
		{"fd00::1/64", "fd00::2/64"},
		{"fd00::ffff:ffff", "fd00::1:0:0"},
		{"10.77.0.3/30", ""},
	} {
		got, err := MirrorL3Addr(tc.in)
		if tc.want == "" {
			if err == nil || !strings.Contains(err.Error(), "no free address") {
				t.Errorf("%s: want a no-free-address error, got %q, %v", tc.in, got, err)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Errorf("%s: got %q, %v; want %q", tc.in, got, err, tc.want)
		}
	}
}

func TestRenderConfigRefuses(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*model.ServerTunnel)
		want   string
	}{
		{"unknown role", func(t *model.ServerTunnel) { t.Role = "sidecar" }, `tunnel "r1": unknown role "sidecar"`},
		{"unknown mode", func(t *model.ServerTunnel) { t.Mode = "mesh" }, `tunnel "r1": unknown mode "mesh"`},
		{"missing secret", func(t *model.ServerTunnel) { t.RakhshSecret = "" }, `tunnel "r1": the tunnel has no Rakhsh secret key`},
		{"missing peer key", func(t *model.ServerTunnel) { t.RakhshPeerPublic = "" }, `tunnel "r1": the tunnel has no peer public key`},
		{"short peer key", func(t *model.ServerTunnel) { t.RakhshPeerPublic = "AAAA" }, `tunnel "r1": the peer public key is not a 32-byte base64 value`},
		{"missing uuid", func(t *model.ServerTunnel) { t.UUID = "" }, `tunnel "r1": the tunnel has no uuid to derive its token from`},
		{"unknown carrier", func(t *model.ServerTunnel) { t.Transport = "carrierpigeon" }, `tunnel "r1": unknown carrier "carrierpigeon"`},
		{"quote in a forward", func(t *model.ServerTunnel) { t.ForwardPorts = []string{`443=127.0.0.1:2096"`} }, `tunnel "r1": forwarded port "443=127.0.0.1:2096\"" is not valid`},
		{"forward without target", func(t *model.ServerTunnel) { t.ForwardPorts = []string{"443"} }, `tunnel "r1": forwarded port "443" is not valid`},
		{"forward port out of range", func(t *model.ServerTunnel) { t.ForwardPorts = []string{"70000=127.0.0.1:2096"} }, `tunnel "r1": forwarded port "70000=127.0.0.1:2096" is not valid`},
		{"entry with nothing to carry", func(t *model.ServerTunnel) { t.ForwardPorts, t.RelayPort = nil, 0 }, `tunnel "r1": an entry tunnel needs at least one forwarded port`},
		{"option the catalog refuses", func(t *model.ServerTunnel) { t.Options = map[string]any{"bogus": 1} }, `tunnel "r1": option "bogus" is not offered for a reverse tunnel`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tun := portalTunnel()
			tc.change(&tun)
			_, err := RenderConfig(tun, socketPath)
			if err == nil || err.Error() != tc.want {
				t.Fatalf("got %v; want %q", err, tc.want)
			}
		})
	}
}

func TestRenderConfigNeverLetsAValueEndItsString(t *testing.T) {
	tun := portalTunnel()
	tun.Options = map[string]any{"sni": "www.example.com\"\nrole = \"node"}
	if _, err := RenderConfig(tun, socketPath); err == nil {
		t.Fatal("a value that could end its TOML string must be refused")
	}
}

func TestControlSocketLivesInTheTunnelFolder(t *testing.T) {
	if got := ControlSocket("/etc/x-ui/tunnels", "de1"); got != "/etc/x-ui/tunnels/de1.rakhsh.sock" {
		t.Errorf("got %q", got)
	}
}

func TestRelayForwardIsALoopbackPair(t *testing.T) {
	if got := RelayForward(63500); got != "127.0.0.1:63500=127.0.0.1:63500" {
		t.Errorf("got %q", got)
	}
}
