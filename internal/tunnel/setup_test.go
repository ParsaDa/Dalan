package tunnel

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/backpack"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/rakhsh"
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
		{"newer format", encode(`{"v":9,"name":"a","address":"203.0.113.10","port":8443,"uuid":"8d6f3a0e-5c1b-4f0e-9a51-2f6d1c7b9e10","security":"none"}`), "version 9"},
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

func backpackSpec() PortalSpec {
	return PortalSpec{
		Name: "bp1", Address: "203.0.113.10", Port: 8443, Engine: model.TunnelEngineBackpack,
		Transport: "wssmux", FallbackTransports: []string{"quic"},
		ForwardPorts: []string{"443=127.0.0.1:2096"}, AcceptUDP: true,
	}
}

func TestABackpackPortalAndItsLinkCarryTheEngineSettings(t *testing.T) {
	portal, err := NewPortal(backpackSpec())
	if err != nil {
		t.Fatalf("NewPortal: %v", err)
	}
	if portal.Engine != model.TunnelEngineBackpack || portal.Transport != "wssmux" || !portal.AcceptUDP ||
		len(portal.ForwardPorts) != 1 || portal.UUID == "" || portal.RealityPrivateKey != "" {
		t.Fatalf("portal = %+v, want a BackPack portal with a token and no REALITY keys", portal)
	}

	link, err := SetupLink(portal)
	if err != nil {
		t.Fatalf("SetupLink: %v", err)
	}
	bridge, err := ParseSetupLink(link)
	if err != nil {
		t.Fatalf("ParseSetupLink: %v", err)
	}
	if bridge.Engine != model.TunnelEngineBackpack || bridge.Role != model.TunnelRoleBridge ||
		bridge.Transport != "wssmux" || len(bridge.FallbackTransports) != 1 || bridge.FallbackTransports[0] != "quic" ||
		bridge.UUID != portal.UUID || bridge.Address != "203.0.113.10" || bridge.Port != 8443 || len(bridge.ForwardPorts) != 0 {
		t.Fatalf("bridge = %+v, want the dialing end with the same transport, fallback and token and no forwarded ports", bridge)
	}
}

func TestBackpackPortalsNeedAKnownTransportAndEngine(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*PortalSpec)
		wantErr string
	}{
		{"unknown transport", func(s *PortalSpec) { s.Transport = "smoke" }, "transport"},
		{"unknown fallback", func(s *PortalSpec) { s.FallbackTransports = []string{"smoke"} }, "transport"},
		{"unknown engine", func(s *PortalSpec) { s.Engine = "gost" }, "engine"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			spec := backpackSpec()
			c.mutate(&spec)
			if _, err := NewPortal(spec); err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("NewPortal error = %v, want one mentioning %q", err, c.wantErr)
			}
		})
	}
}

// A panel that predates BackPack must refuse its links by version, not misread them as Xray tunnels.
func TestBackpackLinksUseANewerLinkVersionThanXrayLinks(t *testing.T) {
	version := func(link string) string {
		raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(link, "dalan://tunnel/"))
		if err != nil {
			t.Fatalf("decode link: %v", err)
		}
		_, rest, _ := strings.Cut(string(raw), `"v":`)
		v, _, _ := strings.Cut(rest, ",")
		return v
	}
	backpackPortal, err := NewPortal(backpackSpec())
	if err != nil {
		t.Fatalf("NewPortal: %v", err)
	}
	backpackLink, _ := SetupLink(backpackPortal)
	xrayLink, _ := SetupLink(realityPortal(t))
	if version(backpackLink) != "2" || version(xrayLink) != "1" {
		t.Fatalf("link versions: backpack=%s xray=%s, want 2 and 1", version(backpackLink), version(xrayLink))
	}
}

func TestABackpackTunnelGetsARelayPortTheBridgeSharesAndXrayTunnelsDoNot(t *testing.T) {
	spec := backpackSpec()
	spec.ForwardPorts = nil
	portal, err := NewPortal(spec)
	if err != nil {
		t.Fatalf("NewPortal without forwarded ports: %v", err)
	}
	if portal.RelayPort < backpack.RelayPortMin || portal.RelayPort > backpack.RelayPortMax {
		t.Fatalf("RelayPort = %d, want one in %d-%d", portal.RelayPort, backpack.RelayPortMin, backpack.RelayPortMax)
	}
	link, err := SetupLink(portal)
	if err != nil {
		t.Fatalf("SetupLink: %v", err)
	}
	bridge, err := ParseSetupLink(link)
	if err != nil {
		t.Fatalf("ParseSetupLink: %v", err)
	}
	if bridge.RelayPort != portal.RelayPort {
		t.Fatalf("bridge RelayPort = %d, want the portal's %d", bridge.RelayPort, portal.RelayPort)
	}
	if xray := realityPortal(t); xray.RelayPort != 0 {
		t.Fatalf("an Xray tunnel got RelayPort %d, want none", xray.RelayPort)
	}
}

func TestOptionsTravelInTheLinkAndMakeItANewerVersion(t *testing.T) {
	spec := backpackSpec()
	spec.Options = map[string]any{"nodelay": true, "edge_ip": "198.51.100.7", "keepalive_period": float64(20)}
	portal, err := NewPortal(spec)
	if err != nil {
		t.Fatalf("NewPortal: %v", err)
	}
	link, err := SetupLink(portal)
	if err != nil {
		t.Fatalf("SetupLink: %v", err)
	}
	if !strings.Contains(decodeLink(t, link), `"v":3`) {
		t.Fatalf("link = %s, want version 3 because it carries options", decodeLink(t, link))
	}
	bridge, err := ParseSetupLink(link)
	if err != nil {
		t.Fatalf("ParseSetupLink: %v", err)
	}
	if bridge.Options["edge_ip"] != "198.51.100.7" || bridge.Options["nodelay"] != true || bridge.Options["keepalive_period"] != float64(20) {
		t.Fatalf("bridge options = %v, want the portal's", bridge.Options)
	}
}

func decodeLink(t *testing.T, link string) string {
	t.Helper()
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(link, "dalan://tunnel/"))
	if err != nil {
		t.Fatalf("decode link: %v", err)
	}
	return string(raw)
}

func TestNewPortalRefusesOptionsItCannotWriteAndOptionsOnXrayTunnels(t *testing.T) {
	spec := backpackSpec()
	spec.Options = map[string]any{"no_such_option": true}
	if _, err := NewPortal(spec); err == nil {
		t.Fatal("NewPortal accepted an option the engine does not read")
	}
	xray := PortalSpec{
		Name: "x1", Address: "203.0.113.10", Port: 8443, Security: model.TunnelSecurityNone,
		Options: map[string]any{"nodelay": true},
	}
	if _, err := NewPortal(xray); err == nil || !strings.Contains(err.Error(), "BackPack") {
		t.Fatalf("NewPortal(Xray with options) error = %v, want one saying options belong to BackPack", err)
	}
}

func directSpec() PortalSpec {
	return PortalSpec{
		Name: "gm1", Address: "203.0.113.20", Port: 9000, Engine: model.TunnelEngineBackpack, Mode: model.TunnelModeDirect,
		ForwardPorts: []string{"27015"}, AcceptUDP: true, Options: map[string]any{"carrier": "udp", "fec_data": float64(10), "fec_parity": float64(2)},
	}
}

func TestADirectPortalGetsAnAddressPairAndItsBridgeGetsTheMirror(t *testing.T) {
	portal, err := NewPortal(directSpec())
	if err != nil {
		t.Fatalf("NewPortal: %v", err)
	}
	local, _ := portal.Options["local_ip"].(string)
	peer, _ := portal.Options["peer_ip"].(string)
	if portal.Mode != model.TunnelModeDirect || portal.RelayPort != 0 || portal.Transport != "" ||
		!strings.HasPrefix(local, "10.10.") || !strings.HasSuffix(local, ".1/30") || !strings.HasSuffix(peer, ".2") {
		t.Fatalf("portal = %+v, want a direct tunnel with a /30 pair, no relay port and no transport", portal)
	}
	link, err := SetupLink(portal)
	if err != nil {
		t.Fatalf("SetupLink: %v", err)
	}
	bridge, err := ParseSetupLink(link)
	if err != nil {
		t.Fatalf("ParseSetupLink: %v", err)
	}
	prefix := strings.TrimSuffix(local, ".1/30")
	if bridge.Mode != model.TunnelModeDirect || bridge.Role != model.TunnelRoleBridge ||
		bridge.Options["local_ip"] != prefix+".2/30" || bridge.Options["peer_ip"] != prefix+".1" ||
		bridge.Options["carrier"] != "udp" || bridge.Options["fec_data"] != float64(10) || len(bridge.ForwardPorts) != 0 {
		t.Fatalf("bridge = %+v, want the mirrored addresses and the shared carrier and error correction", bridge)
	}
}

func TestADirectTunnelNeedsBackpackAndAKnownCarrier(t *testing.T) {
	cases := map[string]func(*PortalSpec){
		"xray engine":     func(s *PortalSpec) { s.Engine = model.TunnelEngineXray; s.Security = model.TunnelSecurityNone },
		"no carrier":      func(s *PortalSpec) { delete(s.Options, "carrier") },
		"unknown carrier": func(s *PortalSpec) { s.Options["carrier"] = "smoke" },
		"reverse option":  func(s *PortalSpec) { s.Options["edge_ip"] = "198.51.100.7" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			spec := directSpec()
			mutate(&spec)
			if _, err := NewPortal(spec); err == nil {
				t.Fatalf("NewPortal accepted a direct tunnel with %s", name)
			}
		})
	}
}

func rakhshSpec() PortalSpec {
	return PortalSpec{
		Name: "rk1", Address: "203.0.113.10", Port: 7000, Engine: model.TunnelEngineRakhsh, Transport: "wss",
		ForwardPorts: []string{"443=127.0.0.1:2096"}, Options: map[string]any{"sni": "www.example.com"},
	}
}

func TestARakhshPortalAndItsBridgePinEachOthersKeys(t *testing.T) {
	for _, mode := range []string{model.TunnelModeReverse, model.TunnelModeDirect} {
		t.Run(mode, func(t *testing.T) {
			spec := rakhshSpec()
			spec.Mode = mode
			portal, err := NewPortal(spec)
			if err != nil {
				t.Fatalf("NewPortal: %v", err)
			}
			if portal.Engine != model.TunnelEngineRakhsh || portal.Mode != mode || portal.Transport != "wss" {
				t.Fatalf("portal = engine %q mode %q carrier %q", portal.Engine, portal.Mode, portal.Transport)
			}
			if portal.RelayPort < backpack.RelayPortMin || portal.RelayPort > backpack.RelayPortMax {
				t.Fatalf("relay port %d is outside the relay range: a Rakhsh tunnel reaches Xray through it in both modes", portal.RelayPort)
			}
			link, err := SetupLink(portal)
			if err != nil {
				t.Fatalf("SetupLink: %v", err)
			}
			if !strings.Contains(decodeLink(t, link), `"v":4`) {
				t.Fatalf("a Rakhsh link must be version 4: %s", decodeLink(t, link))
			}
			bridge, err := ParseSetupLink(link)
			if err != nil {
				t.Fatalf("ParseSetupLink: %v", err)
			}
			portalPublic, _ := rakhsh.PublicKey(portal.RakhshSecret)
			bridgePublic, _ := rakhsh.PublicKey(bridge.RakhshSecret)
			if bridge.RakhshPeerPublic != portalPublic || portal.RakhshPeerPublic != bridgePublic {
				t.Fatalf("keys do not pin each other: bridge pins %q (portal is %q), portal pins %q (bridge is %q)",
					bridge.RakhshPeerPublic, portalPublic, portal.RakhshPeerPublic, bridgePublic)
			}
			if bridge.Engine != model.TunnelEngineRakhsh || bridge.Mode != mode || bridge.RelayPort != portal.RelayPort || bridge.UUID != portal.UUID {
				t.Fatalf("bridge = %+v does not match its portal", bridge)
			}
			text, err := rakhsh.RenderConfig(bridge, "")
			if err != nil {
				t.Fatalf("the parsed bridge does not render: %v", err)
			}
			for _, want := range []string{`peer_public = "` + portalPublic + `"`, `static_secret = "` + bridge.RakhshSecret + `"`} {
				if !strings.Contains(text, want) {
					t.Errorf("bridge config lacks %s:\n%s", want, text)
				}
			}
			again, _ := SetupLink(portal)
			if again != link {
				t.Fatal("the same portal gave a different link the second time, so a re-shown link would not pair with the first")
			}
		})
	}
}

func TestRakhshLinkNeverCarriesPortalSecret(t *testing.T) {
	portal, err := NewPortal(rakhshSpec())
	if err != nil {
		t.Fatal(err)
	}
	link, _ := SetupLink(portal)
	if strings.Contains(decodeLink(t, link), portal.RakhshSecret) {
		t.Fatal("the setup link carries the portal's secret key")
	}
}

func TestNewPortalRefusesARakhshTunnelThatCannotWork(t *testing.T) {
	cases := map[string]func(*PortalSpec){
		"no carrier":      func(s *PortalSpec) { s.Transport = "" },
		"unknown carrier": func(s *PortalSpec) { s.Transport = "smoke" },
		"xray security":   func(s *PortalSpec) { s.Security, s.ServerName = model.TunnelSecurityReality, "www.example.com" },
		"unknown option":  func(s *PortalSpec) { s.Options = map[string]any{"bogus": 1} },
		"unknown mode":    func(s *PortalSpec) { s.Mode = "mesh" },
		"bad forward":     func(s *PortalSpec) { s.ForwardPorts = []string{`443=127.0.0.1:2096"`} },
		"no layer-3 room": func(s *PortalSpec) { s.Options = map[string]any{"l3_addr": "10.77.0.3/30"} },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			spec := rakhshSpec()
			mutate(&spec)
			if _, err := NewPortal(spec); err == nil {
				t.Fatalf("NewPortal accepted a Rakhsh tunnel with %s", name)
			}
		})
	}
}

func TestParseSetupLinkRefusesARakhshLinkItCannotUse(t *testing.T) {
	portal, err := NewPortal(rakhshSpec())
	if err != nil {
		t.Fatal(err)
	}
	good, _ := SetupLink(portal)
	var payload map[string]any
	if err := json.Unmarshal([]byte(decodeLink(t, good)), &payload); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name    string
		mutate  func(map[string]any)
		wantErr string
	}{
		{"xray fields", func(p map[string]any) { p["publicKey"], p["shortId"] = "abc", "def" }, "Xray"},
		{"no bridge secret", func(p map[string]any) { delete(p, "rakhshSecret") }, "secret key"},
		{"no portal public key", func(p map[string]any) { delete(p, "rakhshPeer") }, "peer public key"},
		{"unknown carrier", func(p map[string]any) { p["transport"] = "smoke" }, "carrier"},
		{"relay outside the range", func(p map[string]any) { p["relayPort"] = 80 }, "relay port"},
		{"unknown mode", func(p map[string]any) { p["mode"] = "mesh" }, "mode"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			changed := map[string]any{}
			for k, v := range payload {
				changed[k] = v
			}
			c.mutate(changed)
			raw, _ := json.Marshal(changed)
			link := "dalan://tunnel/" + base64.RawURLEncoding.EncodeToString(raw)
			if _, err := ParseSetupLink(link); err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("ParseSetupLink error = %v, want one mentioning %q", err, c.wantErr)
			}
		})
	}
	future := map[string]any{"v": 9, "name": "a", "address": "203.0.113.10", "port": 8443, "uuid": portal.UUID}
	raw, _ := json.Marshal(future)
	if _, err := ParseSetupLink("dalan://tunnel/" + base64.RawURLEncoding.EncodeToString(raw)); err == nil || !strings.Contains(err.Error(), "versions 1 to 4") {
		t.Fatalf("a version 9 link gave %v, want one that says this panel understands versions 1 to 4", err)
	}
}

func TestARakhshLinkLeavesTheEntryServersBindAddressBehind(t *testing.T) {
	spec := rakhshSpec()
	spec.Options = map[string]any{"sni": "www.example.com", "bind": "192.0.2.5"}
	portal, err := NewPortal(spec)
	if err != nil {
		t.Fatalf("NewPortal: %v", err)
	}
	link, err := SetupLink(portal)
	if err != nil {
		t.Fatalf("SetupLink: %v", err)
	}
	bridge, err := ParseSetupLink(link)
	if err != nil {
		t.Fatalf("ParseSetupLink: %v", err)
	}
	if _, has := bridge.Options["bind"]; has {
		t.Fatalf("bind names an address of the entry server and would fail to start on the exit server: %v", bridge.Options)
	}
	if bridge.Options["sni"] != "www.example.com" {
		t.Fatalf("shared options must still travel: %v", bridge.Options)
	}
	if portal.Options["bind"] != "192.0.2.5" {
		t.Fatalf("the entry end keeps its own bind: %v", portal.Options)
	}
}
