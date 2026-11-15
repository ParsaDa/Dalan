package tunnel

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

func realityPortal(t *testing.T) model.ServerTunnel {
	t.Helper()
	portal, err := NewPortal(PortalSpec{
		Name: "tr1", Address: "203.0.113.10", Port: 8443,
		Security: model.TunnelSecurityReality, ServerName: "www.example.com",
	})
	if err != nil {
		t.Fatalf("NewPortal: %v", err)
	}
	return portal
}

func TestSetupLinkCarriesWhatTheBridgeNeedsAndNotThePrivateKey(t *testing.T) {
	portal := realityPortal(t)
	link, err := SetupLink(portal)
	if err != nil {
		t.Fatalf("SetupLink: %v", err)
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(link, "dalan://tunnel/"))
	if err != nil {
		t.Fatalf("link %q is not dalan://tunnel/<base64url>: %v", link, err)
	}
	if portal.RealityPrivateKey == "" || strings.Contains(string(payload), portal.RealityPrivateKey) {
		t.Fatalf("private key must exist on the portal and stay out of the link; payload: %s", payload)
	}

	bridge, err := ParseSetupLink(link)
	if err != nil {
		t.Fatalf("ParseSetupLink: %v", err)
	}
	want := model.ServerTunnel{
		Name: "tr1", Role: model.TunnelRoleBridge, Engine: model.TunnelEngineXray, Enable: true,
		Address: "203.0.113.10", Port: 8443, UUID: portal.UUID, Security: model.TunnelSecurityReality,
		ServerName: "www.example.com", RealityPublicKey: portal.RealityPublicKey, ShortId: portal.ShortId,
	}
	if bridge.Name != want.Name || bridge.Role != want.Role || bridge.Engine != want.Engine ||
		bridge.Enable != want.Enable || bridge.Address != want.Address || bridge.Port != want.Port ||
		bridge.UUID != want.UUID || bridge.Security != want.Security || bridge.ServerName != want.ServerName ||
		bridge.RealityPublicKey != want.RealityPublicKey || bridge.ShortId != want.ShortId ||
		bridge.RealityPrivateKey != "" {
		t.Fatalf("bridge = %+v, want %+v", bridge, want)
	}
}

func TestNewPortalRejectsSpecsThatCannotWork(t *testing.T) {
	base := PortalSpec{Name: "de1", Address: "203.0.113.10", Port: 8443, Security: model.TunnelSecurityNone}
	cases := []struct {
		name    string
		mutate  func(*PortalSpec)
		wantErr string
	}{
		{"name with spaces", func(s *PortalSpec) { s.Name = "my tunnel" }, "name"},
		{"upper-case name", func(s *PortalSpec) { s.Name = "DE1" }, "name"},
		{"empty address", func(s *PortalSpec) { s.Address = "" }, "address"},
		{"port zero", func(s *PortalSpec) { s.Port = 0 }, "port"},
		{"port too large", func(s *PortalSpec) { s.Port = 70000 }, "port"},
		{"unknown security", func(s *PortalSpec) { s.Security = "tls13" }, "security"},
		{"reality without server name", func(s *PortalSpec) { s.Security = model.TunnelSecurityReality }, "server name"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			spec := base
			c.mutate(&spec)
			if _, err := NewPortal(spec); err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("NewPortal error = %v, want one mentioning %q", err, c.wantErr)
			}
		})
	}
}

func TestParseSetupLinkRejectsLinksItCannotUse(t *testing.T) {
	encode := func(json string) string {
		return "dalan://tunnel/" + base64.RawURLEncoding.EncodeToString([]byte(json))
	}
	cases := []struct{ name, link, wantErr string }{
		{"another scheme", "backpack://abc", "not a Dalan tunnel link"},
		{"not base64", "dalan://tunnel/%%%", "not a Dalan tunnel link"},
		{"not JSON", encode("hello"), "not a Dalan tunnel link"},
		{"newer format", encode(`{"v":2,"name":"a","address":"203.0.113.10","port":8443,"uuid":"8d6f3a0e-5c1b-4f0e-9a51-2f6d1c7b9e10","security":"none"}`), "version 2"},
		{"missing uuid", encode(`{"v":1,"name":"a","address":"203.0.113.10","port":8443,"security":"none"}`), "uuid"},
		{"bad name", encode(`{"v":1,"name":"A B","address":"203.0.113.10","port":8443,"uuid":"8d6f3a0e-5c1b-4f0e-9a51-2f6d1c7b9e10","security":"none"}`), "name"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := ParseSetupLink(c.link); err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("ParseSetupLink error = %v, want one mentioning %q", err, c.wantErr)
			}
		})
	}
}

func TestSetupLinkIsOnlyForPortals(t *testing.T) {
	bridge := model.ServerTunnel{Name: "a", Role: model.TunnelRoleBridge}
	if _, err := SetupLink(bridge); err == nil || !strings.Contains(err.Error(), "portal") {
		t.Fatalf("SetupLink(bridge) error = %v, want one mentioning the portal", err)
	}
}
