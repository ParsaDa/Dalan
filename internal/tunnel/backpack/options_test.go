package backpack

import (
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

const optionsToken = "8d6f3a0e-5c1b-4f0e-9a51-2f6d1c7b9e10"

func reverseEnd(role string, options map[string]any) model.ServerTunnel {
	return model.ServerTunnel{
		Name: "de1", Role: role, Engine: model.TunnelEngineBackpack, Address: "203.0.113.10", Port: 8443,
		UUID: optionsToken, Transport: "ws", ForwardPorts: []string{"443"}, Options: options,
	}
}

func TestEachEndWritesOnlyTheOptionsOfItsRole(t *testing.T) {
	options := map[string]any{
		"log_level": "debug", "bandwidth_mbps": float64(50), "edge_ip": "198.51.100.7",
		"health_failover": true, "fallback_addrs": []any{"198.51.100.8:443"},
	}
	portal, err := RenderConfig(reverseEnd(model.TunnelRolePortal, options))
	if err != nil {
		t.Fatalf("portal: %v", err)
	}
	bridge, err := RenderConfig(reverseEnd(model.TunnelRoleBridge, options))
	if err != nil {
		t.Fatalf("bridge: %v", err)
	}
	for _, line := range []string{`log_level = "debug"`, "bandwidth_mbps = 50"} {
		if !strings.Contains(portal, line) {
			t.Errorf("portal config lacks %q:\n%s", line, portal)
		}
	}
	for _, line := range []string{`log_level = "debug"`, `edge_ip = "198.51.100.7"`, "health_failover = true", `fallback_addrs = ["198.51.100.8:443"]`} {
		if !strings.Contains(bridge, line) {
			t.Errorf("bridge config lacks %q:\n%s", line, bridge)
		}
	}
	for _, line := range []string{"edge_ip", "health_failover", "fallback_addrs"} {
		if strings.Contains(portal, line) {
			t.Errorf("portal config holds the bridge-only option %q:\n%s", line, portal)
		}
	}
	if strings.Contains(bridge, "bandwidth_mbps") {
		t.Errorf("bridge config holds the portal-only option bandwidth_mbps:\n%s", bridge)
	}
}

func TestRenderConfigRefusesAnOptionThatWouldBreakTheFileOrTheTunnel(t *testing.T) {
	cases := map[string]map[string]any{
		"an option the engine does not read": {"no_such_option": true},
		"a number where a switch belongs":    {"nodelay": float64(1)},
		"a number out of range":              {"keepalive_period": float64(999999)},
		"a fraction where a whole belongs":   {"keepalive_period": 2.5},
		"a choice that is not offered":       {"log_level": "loud"},
		"a host that ends the string":        {"edge_ip": "1.2.3.4\"\ntoken = \"x"},
		"a list item that is no address":     {"fallback_addrs": []any{"no port"}},
		"a proxy that is no URL":             {"proxy": "not a url"},
	}
	for name, options := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := RenderConfig(reverseEnd(model.TunnelRoleBridge, options)); err == nil {
				t.Fatalf("RenderConfig accepted %v", options)
			}
		})
	}
}

func directEnd(role string, options map[string]any) model.ServerTunnel {
	t := model.ServerTunnel{
		Name: "gm1", Role: role, Engine: model.TunnelEngineBackpack, Mode: model.TunnelModeDirect, Id: 7,
		Address: "203.0.113.20", Port: 9000, UUID: optionsToken, ForwardPorts: []string{"27015"}, AcceptUDP: true,
		Options: map[string]any{"carrier": "udp", "local_ip": "10.10.7.1/30", "peer_ip": "10.10.7.2"},
	}
	for k, v := range options {
		t.Options[k] = v
	}
	return t
}

func TestADirectPortalDialsTheExitAndForwardsItsPorts(t *testing.T) {
	got, err := RenderConfig(directEnd(model.TunnelRolePortal, map[string]any{"fec_data": float64(10), "fec_parity": float64(2)}))
	if err != nil {
		t.Fatalf("RenderConfig: %v", err)
	}
	want := `[l3]
mode = "dial"
addr = "203.0.113.20:9000"
token = "8d6f3a0e-5c1b-4f0e-9a51-2f6d1c7b9e10"
iface = "dl7"
carrier = "udp"
local_ip = "10.10.7.1/30"
peer_ip = "10.10.7.2"
ports = ["27015"]
accept_udp = true
fec_data = 10
fec_parity = 2
`
	if got != want {
		t.Fatalf("config =\n%s\nwant\n%s", got, want)
	}
}

func TestADirectBridgeListensAndTakesTheOtherAddress(t *testing.T) {
	bridge := directEnd(model.TunnelRoleBridge, nil)
	bridge.ForwardPorts, bridge.AcceptUDP = nil, false
	bridge.Options["local_ip"], bridge.Options["peer_ip"] = "10.10.7.2/30", "10.10.7.1"
	got, err := RenderConfig(bridge)
	if err != nil {
		t.Fatalf("RenderConfig: %v", err)
	}
	want := `[l3]
mode = "listen"
addr = "0.0.0.0:9000"
token = "8d6f3a0e-5c1b-4f0e-9a51-2f6d1c7b9e10"
iface = "dl7"
carrier = "udp"
local_ip = "10.10.7.2/30"
peer_ip = "10.10.7.1"
`
	if got != want {
		t.Fatalf("config =\n%s\nwant\n%s", got, want)
	}
}

func TestADirectTunnelNeedsACarrierAndTunnelAddresses(t *testing.T) {
	for _, missing := range []string{"carrier", "local_ip", "peer_ip"} {
		t.Run(missing, func(t *testing.T) {
			end := directEnd(model.TunnelRolePortal, nil)
			delete(end.Options, missing)
			if _, err := RenderConfig(end); err == nil {
				t.Fatalf("RenderConfig accepted a direct tunnel without %s", missing)
			}
		})
	}
}

func TestAReverseOnlyOptionIsRefusedOnADirectTunnel(t *testing.T) {
	if _, err := RenderConfig(directEnd(model.TunnelRolePortal, map[string]any{"edge_ip": "198.51.100.7"})); err == nil {
		t.Fatal("RenderConfig accepted a reverse-tunnel option on a direct tunnel")
	}
}
