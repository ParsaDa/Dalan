package tunnel

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/backpack"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/forwarder"
)

func forwarderSpec(engine, transport, mode string) PortalSpec {
	return PortalSpec{
		Name: "de1", Address: "203.0.113.10", Port: 8443, Engine: engine, Transport: transport, Mode: mode,
		ForwardPorts: []string{"2222=127.0.0.1:22"}, Options: map[string]any{},
	}
}

func linkPayload(t *testing.T, link string) map[string]any {
	t.Helper()
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(link, "dalan://tunnel/"))
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestNewPortalOnAForwardingEngineGeneratesTheSharedSecretAndARelay(t *testing.T) {
	for _, c := range []struct{ engine, transport, mode string }{
		{model.TunnelEngineGost, "wss", ""},
		{model.TunnelEngineGost, "tls", model.TunnelModeDirect},
		{model.TunnelEngineRathole, "tcp", model.TunnelModeReverse},
		{model.TunnelEngineBackhaul, "wssmux", ""},
		{model.TunnelEngineKariz, "tcpmux", model.TunnelModeDirect},
		{model.TunnelEngineSingbox, "tcp-tls", model.TunnelModeDirect},
		{model.TunnelEngineXrayCore, "grpc-tls", model.TunnelModeDirect},
	} {
		t.Run(c.engine+"/"+c.transport+"/"+c.mode, func(t *testing.T) {
			portal, err := NewPortal(forwarderSpec(c.engine, c.transport, c.mode))
			if err != nil {
				t.Fatal(err)
			}
			if portal.Engine != c.engine || portal.Transport != c.transport || portal.Role != model.TunnelRolePortal {
				t.Fatalf("portal = %+v", portal)
			}
			wantMode := c.mode
			if wantMode == "" {
				wantMode = model.TunnelModeReverse
			}
			if portal.Mode != wantMode {
				t.Fatalf("mode = %q, want %q", portal.Mode, wantMode)
			}
			if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(portal.EngineSecret) {
				t.Fatalf("secret %q is not 32 random bytes in hex", portal.EngineSecret)
			}
			if portal.RelayPort < backpack.RelayPortMin || portal.RelayPort > backpack.RelayPortMax {
				t.Fatalf("relay port %d is outside %d-%d", portal.RelayPort, backpack.RelayPortMin, backpack.RelayPortMax)
			}
			if !reflect.DeepEqual(portal.ForwardPorts, []string{"2222=127.0.0.1:22"}) {
				t.Fatalf("forwards = %v", portal.ForwardPorts)
			}
		})
	}
}

func TestNewPortalOnAForwardingEngineRefusesWhatTheEngineCannotRun(t *testing.T) {
	cases := []struct {
		name    string
		spec    PortalSpec
		wantErr string
	}{
		{"no transport", forwarderSpec(model.TunnelEngineGost, "", ""), "transport"},
		{"unknown transport", forwarderSpec(model.TunnelEngineGost, "carrier-pigeon", ""), "transport"},
		{"direct on a reverse-only engine", forwarderSpec(model.TunnelEngineRathole, "tcp", model.TunnelModeDirect), "direct"},
		{"reverse on a direct-only engine", forwarderSpec(model.TunnelEngineSingbox, "tcp-tls", model.TunnelModeReverse), "reverse"},
		{"unknown mode", forwarderSpec(model.TunnelEngineGost, "tls", "sideways"), "mode"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := NewPortal(c.spec); err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("NewPortal error = %v, want one mentioning %q", err, c.wantErr)
			}
		})
	}
	bad := forwarderSpec(model.TunnelEngineGost, "tls", "")
	bad.ForwardPorts = []string{"nonsense"}
	if _, err := NewPortal(bad); err == nil || !strings.Contains(err.Error(), "forwarded port") {
		t.Fatalf("NewPortal error = %v, want the bad forwarded port named", err)
	}
}

func TestSetupLinkCarriesWhatTheExitEndOfAForwardingEngineNeeds(t *testing.T) {
	spec := forwarderSpec(model.TunnelEngineKariz, "wss", model.TunnelModeDirect)
	spec.Options = map[string]any{"profile": "ultraspeed"}
	portal, err := NewPortal(spec)
	if err != nil {
		t.Fatal(err)
	}
	link, err := SetupLink(portal)
	if err != nil {
		t.Fatal(err)
	}
	if v := linkPayload(t, link)["v"]; v != float64(5) {
		t.Fatalf("link version = %v, want 5", v)
	}
	bridge, err := ParseSetupLink(link)
	if err != nil {
		t.Fatal(err)
	}
	same := func(name string, got, want any) {
		t.Helper()
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s = %v, want %v", name, got, want)
		}
	}
	same("role", bridge.Role, model.TunnelRoleBridge)
	same("engine", bridge.Engine, portal.Engine)
	same("transport", bridge.Transport, portal.Transport)
	same("mode", bridge.Mode, portal.Mode)
	same("secret", bridge.EngineSecret, portal.EngineSecret)
	same("relay port", bridge.RelayPort, portal.RelayPort)
	same("forwards", bridge.ForwardPorts, portal.ForwardPorts)
	same("options", bridge.Options, portal.Options)
	same("address", bridge.Address, portal.Address)
	same("port", bridge.Port, portal.Port)
}

func TestParseSetupLinkRefusesAForwardingLinkThatCannotRun(t *testing.T) {
	good, err := NewPortal(forwarderSpec(model.TunnelEngineGost, "wss", ""))
	if err != nil {
		t.Fatal(err)
	}
	link, err := SetupLink(good)
	if err != nil {
		t.Fatal(err)
	}
	reencode := func(edit func(map[string]any)) string {
		p := linkPayload(t, link)
		edit(p)
		raw, _ := json.Marshal(p)
		return "dalan://tunnel/" + base64.RawURLEncoding.EncodeToString(raw)
	}
	cases := []struct {
		name string
		edit func(map[string]any)
		want string
	}{
		{"no secret", func(p map[string]any) { delete(p, "engineSecret") }, "secret"},
		{"unknown engine", func(p map[string]any) { p["engine"] = "nope" }, "engine"},
		{"unknown transport", func(p map[string]any) { p["transport"] = "carrier-pigeon" }, "transport"},
		{"relay port outside the range", func(p map[string]any) { p["relayPort"] = 80 }, "relay port"},
		{"newer link", func(p map[string]any) { p["v"] = 6 }, "version 6"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := ParseSetupLink(reencode(c.edit)); err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("ParseSetupLink error = %v, want one mentioning %q", err, c.want)
			}
		})
	}
}

func carrierSpec() PortalSpec {
	spec := forwarderSpec(model.TunnelEngineCarrier, "gre", "")
	spec.Options = map[string]any{"dialerIp": "198.51.100.7"}
	return spec
}

func carrierProbe(t *testing.T, err error) {
	t.Helper()
	saved := forwarder.CarrierProbe
	t.Cleanup(func() { forwarder.CarrierProbe = saved })
	forwarder.CarrierProbe = func() error { return err }
}

func TestAKernelCarrierTunnelIsRefusedOnAServerThatCannotMakeInterfaces(t *testing.T) {
	carrierProbe(t, errors.New("kernel carriers need the panel to run as root"))
	if _, err := NewPortal(carrierSpec()); err == nil || !strings.Contains(err.Error(), "root") {
		t.Fatalf("NewPortal = %v, want the missing root named", err)
	}

	carrierProbe(t, nil)
	portal, err := NewPortal(carrierSpec())
	if err != nil {
		t.Fatalf("NewPortal on a server that can run carriers: %v", err)
	}
	link, err := SetupLink(portal)
	if err != nil {
		t.Fatal(err)
	}

	carrierProbe(t, errors.New("kernel carriers need the ip command (iproute2)"))
	if _, err := ParseSetupLink(link); err == nil || !strings.Contains(err.Error(), "ip command") {
		t.Fatalf("ParseSetupLink = %v, want the exit server's missing ip command named", err)
	}
}
