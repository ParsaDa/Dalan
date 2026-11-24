package forwarder

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/carrier"
)

func carrierPortal() model.ServerTunnel {
	return model.ServerTunnel{
		Name: "ir1", Role: model.TunnelRolePortal, Engine: model.TunnelEngineCarrier, Mode: model.TunnelModeReverse,
		Address: "203.0.113.10", Port: 8443, Transport: carrier.GRE, EngineSecret: "s3cret", RelayPort: 63500, Enable: true,
		Options: map[string]any{"dialerIp": "198.51.100.7"},
	}
}

func renderedCarrier(t *testing.T, tun model.ServerTunnel) carrier.Config {
	t.Helper()
	p, err := carrierEngine.PlanFor(tun, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	text, err := carrierEngine.Render(p)
	if err != nil {
		t.Fatal(err)
	}
	var c carrier.Config
	if err := json.Unmarshal([]byte(text), &c); err != nil {
		t.Fatalf("the carrier file is not JSON: %v\n%s", err, text)
	}
	return c
}

func TestCarrierEntryOpensTheRelayPortAndDialsTheExitsLinkAddress(t *testing.T) {
	got := renderedCarrier(t, carrierPortal())
	want := carrier.Config{
		Kind: carrier.GRE, Interface: "dlc63500", Local: "203.0.113.10", Remote: "198.51.100.7",
		Address: "10.222.224.49", VNI: 63500, Port: 8443,
		Relays: []carrier.Relay{{Listen: "127.0.0.1:63500", Dial: "10.222.224.50:8443"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("entry = %+v\nwant   %+v", got, want)
	}
}

func TestCarrierExitListensOnItsLinkAddressAndDeliversToItsOwnTarget(t *testing.T) {
	bridge := carrierPortal()
	bridge.Role = model.TunnelRoleBridge
	bridge.ForwardPorts = []string{"443=127.0.0.1:2096"}
	got := renderedCarrier(t, bridge)
	want := carrier.Config{
		Kind: carrier.GRE, Interface: "dlc63500", Local: "198.51.100.7", Remote: "203.0.113.10",
		Address: "10.222.224.50", VNI: 63500, Port: 8443,
		Relays: []carrier.Relay{
			{Listen: "10.222.224.50:8443", Dial: "127.0.0.1:63500"},
			{Listen: "10.222.224.50:8444", Dial: "127.0.0.1:2096"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("exit = %+v\nwant  %+v", got, want)
	}
}

func TestCarrierEntryGivesEachForwardedPortItsOwnLinkPort(t *testing.T) {
	portal := carrierPortal()
	portal.ForwardPorts = []string{"443=127.0.0.1:2096"}
	got := renderedCarrier(t, portal).Relays
	want := []carrier.Relay{
		{Listen: "127.0.0.1:63500", Dial: "10.222.224.50:8443"},
		{Listen: "0.0.0.0:443", Dial: "10.222.224.50:8444"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("relays = %+v, want %+v", got, want)
	}
}

func TestInADirectCarrierTunnelTheEntryIsTheDialingEnd(t *testing.T) {
	portal := carrierPortal()
	portal.Mode = model.TunnelModeDirect
	portal.Address = "198.51.100.7"
	portal.Options = map[string]any{"dialerIp": "203.0.113.10"}
	entry := renderedCarrier(t, portal)
	if entry.Local != "203.0.113.10" || entry.Remote != "198.51.100.7" {
		t.Fatalf("entry public addresses = %s -> %s, want itself first and the exit second", entry.Local, entry.Remote)
	}
	portal.Role = model.TunnelRoleBridge
	exit := renderedCarrier(t, portal)
	if exit.Local != "198.51.100.7" || exit.Remote != "203.0.113.10" {
		t.Fatalf("exit public addresses = %s -> %s", exit.Local, exit.Remote)
	}
}

func TestVXLANCarrierUsesTheRelayPortAsItsIDAndTheTunnelPortAsItsUDPPort(t *testing.T) {
	portal := carrierPortal()
	portal.Transport = carrier.VXLAN
	got := renderedCarrier(t, portal)
	if got.Kind != carrier.VXLAN || got.VNI != 63500 || got.Port != 8443 {
		t.Fatalf("vxlan config = %+v", got)
	}
}

func TestACarrierTunnelThatCannotBeBuiltIsRefusedWhenItIsMade(t *testing.T) {
	cases := map[string]func(*model.ServerTunnel) string{
		"no dialer address": func(t *model.ServerTunnel) string { t.Options = nil; return "public IP" },
		"hostname instead of an address": func(t *model.ServerTunnel) string {
			t.Address = "node.example.com"
			return "IPv4"
		},
		"no relay port": func(t *model.ServerTunnel) string { t.RelayPort = 0; return "relay port" },
		"udp forward": func(t *model.ServerTunnel) string {
			t.ForwardPorts = []string{"53=127.0.0.1:53/udp"}
			return "TCP"
		},
		"port beyond the link ports": func(t *model.ServerTunnel) string {
			t.Port = 65535
			t.ForwardPorts = []string{"443=127.0.0.1:2096"}
			return "port"
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			tun := carrierPortal()
			want := mutate(&tun)
			err := Validate(tun)
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("Validate = %v, want an error that mentions %q", err, want)
			}
		})
	}
	if err := Validate(carrierPortal()); err != nil {
		t.Fatalf("a good carrier tunnel was refused: %v", err)
	}
}

func TestTheDialerAddressMustBeAnIPv4Address(t *testing.T) {
	tun := carrierPortal()
	tun.Options = map[string]any{"dialerIp": "not-an-ip"}
	if err := Validate(tun); err == nil || !strings.Contains(err.Error(), "dialerIp") {
		t.Fatalf("Validate = %v, want the option named", err)
	}
	tun.Options = map[string]any{"dialerIp": "2001:db8::1"}
	if err := Validate(tun); err == nil {
		t.Fatal("an IPv6 address was accepted")
	}
}

func TestCarrierRunsThePanelItselfWithTheCarrierCommand(t *testing.T) {
	sup := carrierEngine.NewSupervisor("/usr/local/x-ui/x-ui", t.TempDir())
	if sup.Ext != ".json" || !reflect.DeepEqual(sup.Args("/x/ir1.json"), []string{"carrier", "/x/ir1.json"}) {
		t.Fatalf("ext %q args %v", sup.Ext, sup.Args("/x/ir1.json"))
	}
	if !carrierEngine.Self {
		t.Fatal("the carrier is the panel's own binary and has nothing to download")
	}
}

func TestCatalogOffersTheCarrierWithItsAddressSetting(t *testing.T) {
	var found *ForwardingEngine
	for _, e := range Catalog() {
		if e.ID == model.TunnelEngineCarrier {
			found = &e
		}
	}
	if found == nil {
		t.Fatal("the carrier is missing from the catalog")
	}
	ids := []string{}
	for _, tr := range found.Transports {
		if !tr.Reverse || !tr.Direct {
			t.Fatalf("transport %+v must run in both directions", tr)
		}
		ids = append(ids, tr.ID)
	}
	if !reflect.DeepEqual(ids, []string{"gre", "ipip", "vxlan"}) {
		t.Fatalf("transports = %v", ids)
	}
	raw, err := json.Marshal(found.Options)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"key":"dialerIp"`) || !strings.Contains(string(raw), `"kind":"ip"`) || !strings.Contains(string(raw), `"choices":[]`) {
		t.Fatalf("options JSON = %s", raw)
	}
}

func TestPreflightAsksTheCarrierEngineWhetherThisServerCanRunIt(t *testing.T) {
	saved := CarrierProbe
	t.Cleanup(func() { CarrierProbe = saved })
	CarrierProbe = func() error { return errors.New("kernel carriers need the panel to run as root") }

	if err := Preflight(carrierPortal()); err == nil || !strings.Contains(err.Error(), "root") {
		t.Fatalf("Preflight(carrier) = %v, want the probe's error", err)
	}
	if err := Preflight(gostPortal()); err != nil {
		t.Fatalf("Preflight(gost) = %v, an engine with nothing to probe must pass", err)
	}
}
