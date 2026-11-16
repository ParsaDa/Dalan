package xrayreverse

import (
	"encoding/json"
	"testing"

	"github.com/xtls/xray-core/infra/conf"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/backpack"
	"github.com/mhsanaei/3x-ui/v3/internal/xray"
)

func buildWithXray(t *testing.T, cfg *xray.Config) {
	t.Helper()
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	var parsed conf.Config
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("xray rejected config JSON: %v\n%s", err, raw)
	}
	if _, err := parsed.Build(); err != nil {
		t.Fatalf("xray rejected config: %v\n%s", err, raw)
	}
}

func TestRealityTunnelFromASetupLinkIsAcceptedByXray(t *testing.T) {
	portalTunnel, err := tunnel.NewPortal(tunnel.PortalSpec{
		Name: "tr1", Address: "203.0.113.10", Port: 8443,
		Security: model.TunnelSecurityReality, ServerName: "www.example.com",
	})
	if err != nil {
		t.Fatalf("NewPortal: %v", err)
	}
	link, err := tunnel.SetupLink(portalTunnel)
	if err != nil {
		t.Fatalf("SetupLink: %v", err)
	}
	bridgeTunnel, err := tunnel.ParseSetupLink(link)
	if err != nil {
		t.Fatalf("ParseSetupLink: %v", err)
	}

	portal := &xray.Config{OutboundConfigs: []byte(`[{"tag":"direct","protocol":"freedom"}]`)}
	if err := Apply(portal, []model.ServerTunnel{portalTunnel}); err != nil {
		t.Fatalf("Apply portal: %v", err)
	}
	bridge := &xray.Config{OutboundConfigs: []byte(`[{"tag":"direct","protocol":"freedom"}]`)}
	if err := Apply(bridge, []model.ServerTunnel{bridgeTunnel}); err != nil {
		t.Fatalf("Apply bridge: %v", err)
	}
	buildWithXray(t, portal)
	buildWithXray(t, bridge)
}

func TestApplyKeepsTunnelRulesAheadOfOperatorRulesAndSkipsDisabledTunnels(t *testing.T) {
	routing := `{"rules":[{"type":"field","network":"tcp,udp","outboundTag":"blocked"}]}`
	outbounds := `[{"tag":"direct","protocol":"freedom"}]`
	enabled := model.ServerTunnel{
		Name: "a", Role: model.TunnelRoleBridge, Enable: true, Security: model.TunnelSecurityNone,
		Address: "203.0.113.10", Port: 8443, UUID: "8d6f3a0e-5c1b-4f0e-9a51-2f6d1c7b9e10", ExitOutbound: "warp",
	}
	disabled := enabled
	disabled.Name, disabled.Enable = "b", false

	cfg := &xray.Config{RouterConfig: []byte(routing), OutboundConfigs: []byte(outbounds)}
	if err := Apply(cfg, []model.ServerTunnel{enabled, disabled}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	var got struct {
		Rules []struct {
			InboundTag  []string `json:"inboundTag"`
			OutboundTag string   `json:"outboundTag"`
		} `json:"rules"`
	}
	if err := json.Unmarshal(cfg.RouterConfig, &got); err != nil {
		t.Fatalf("routing: %v", err)
	}
	if len(got.Rules) != 2 || got.Rules[0].OutboundTag != "warp" || got.Rules[1].OutboundTag != "blocked" {
		t.Fatalf("rules = %+v, want the tunnel rule (to warp) first, then the operator rule", got.Rules)
	}
	if len(got.Rules[0].InboundTag) != 1 || got.Rules[0].InboundTag[0] != "tunnel-a-exit" {
		t.Fatalf("tunnel rule matches %v, want [tunnel-a-exit]", got.Rules[0].InboundTag)
	}

	untouched := &xray.Config{RouterConfig: []byte(routing), OutboundConfigs: []byte(outbounds)}
	if err := Apply(untouched, []model.ServerTunnel{disabled}); err != nil {
		t.Fatalf("Apply disabled: %v", err)
	}
	if string(untouched.RouterConfig) != routing || string(untouched.OutboundConfigs) != outbounds {
		t.Fatalf("a disabled tunnel changed the config: %s %s", untouched.RouterConfig, untouched.OutboundConfigs)
	}
}

func TestManagementPortalListensOnLoopbackAndHasNoProbe(t *testing.T) {
	portalTunnel := model.ServerTunnel{
		Id: 7, Name: "node-de1", Role: model.TunnelRolePortal, Enable: true, Security: model.TunnelSecurityNone,
		Address: "203.0.113.10", Port: 8443, UUID: "8d6f3a0e-5c1b-4f0e-9a51-2f6d1c7b9e10", ManagePort: 41000,
	}
	cfg := &xray.Config{OutboundConfigs: []byte(`[{"tag":"direct","protocol":"freedom"}]`)}
	if err := Apply(cfg, []model.ServerTunnel{portalTunnel}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	var manage *xray.InboundConfig
	for i := range cfg.InboundConfigs {
		in := &cfg.InboundConfigs[i]
		if in.Port == ProbePort(portalTunnel) {
			t.Fatalf("management tunnel got a probe inbound %q; a probe would be answered by the node's panel", in.Tag)
		}
		if in.Port == 41000 {
			manage = in
		}
	}
	if manage == nil {
		t.Fatalf("no inbound on the management port 41000")
	}
	if string(manage.Listen) != `"127.0.0.1"` {
		t.Fatalf("management inbound listens on %s, want loopback only", manage.Listen)
	}
	buildWithXray(t, cfg)
}

func backpackEnd(role string) model.ServerTunnel {
	return model.ServerTunnel{
		Id: 3, Name: "bp-de1", Role: role, Engine: model.TunnelEngineBackpack, Enable: true,
		Address: "203.0.113.10", Port: 8443, UUID: "8d6f3a0e-5c1b-4f0e-9a51-2f6d1c7b9e10",
		Transport: "tcpmux", RelayPort: 63500,
	}
}

func parsedOutbounds(t *testing.T, cfg *xray.Config) []map[string]any {
	t.Helper()
	var out []map[string]any
	if err := json.Unmarshal(cfg.OutboundConfigs, &out); err != nil {
		t.Fatalf("outbounds: %v", err)
	}
	return out
}

func parsedRules(t *testing.T, cfg *xray.Config) []map[string]any {
	t.Helper()
	var routing struct {
		Rules []map[string]any `json:"rules"`
	}
	if err := json.Unmarshal(cfg.RouterConfig, &routing); err != nil {
		t.Fatalf("routing: %v", err)
	}
	return routing.Rules
}

func TestABackpackEntryEndBecomesATunnelOutboundThroughItsLoopbackRelay(t *testing.T) {
	end := backpackEnd(model.TunnelRolePortal)
	end.RouteInbounds = []string{"in-443"}
	cfg := &xray.Config{OutboundConfigs: []byte(`[{"tag":"direct","protocol":"freedom"}]`)}
	if err := Apply(cfg, []model.ServerTunnel{end}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	var found map[string]any
	for _, o := range parsedOutbounds(t, cfg) {
		if o["tag"] == PortalOutboundTag("bp-de1") {
			found = o
		}
	}
	if found == nil || found["protocol"] != "socks" {
		t.Fatalf("outbounds = %v, want a socks outbound tagged %q", parsedOutbounds(t, cfg), PortalOutboundTag("bp-de1"))
	}
	server := found["settings"].(map[string]any)["servers"].([]any)[0].(map[string]any)
	user := server["users"].([]any)[0].(map[string]any)
	wantUser, wantPass := backpack.RelayCredentials(end)
	if server["address"] != "127.0.0.1" || server["port"] != float64(63500) || user["user"] != wantUser || user["pass"] != wantPass {
		t.Fatalf("relay server = %v, want 127.0.0.1:63500 with the tunnel's credentials", server)
	}
	if rules := parsedRules(t, cfg); len(rules) == 0 || rules[0]["outboundTag"] != PortalOutboundTag("bp-de1") {
		t.Fatalf("rules = %v, want the inbound routed into the tunnel first", rules)
	}
	buildWithXray(t, cfg)
}

func TestABackpackExitEndAnswersOnLoopbackWithPasswordAndLeavesFromTheExitOutbound(t *testing.T) {
	end := backpackEnd(model.TunnelRoleBridge)
	cfg := &xray.Config{OutboundConfigs: []byte(`[{"tag":"direct","protocol":"freedom"}]`)}
	if err := Apply(cfg, []model.ServerTunnel{end}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	var relay *xray.InboundConfig
	for i := range cfg.InboundConfigs {
		if cfg.InboundConfigs[i].Port == 63500 {
			relay = &cfg.InboundConfigs[i]
		}
	}
	if relay == nil || relay.Protocol != "socks" || string(relay.Listen) != `"127.0.0.1"` {
		t.Fatalf("inbounds = %+v, want a socks inbound on 127.0.0.1:63500", cfg.InboundConfigs)
	}
	var settings struct {
		Auth     string `json:"auth"`
		Accounts []struct{ User, Pass string }
	}
	if err := json.Unmarshal(relay.Settings, &settings); err != nil {
		t.Fatalf("relay settings: %v", err)
	}
	wantUser, wantPass := backpack.RelayCredentials(end)
	if settings.Auth != "password" || len(settings.Accounts) != 1 || settings.Accounts[0].User != wantUser || settings.Accounts[0].Pass != wantPass {
		t.Fatalf("relay settings = %+v, want password auth with the tunnel's credentials", settings)
	}
	if rules := parsedRules(t, cfg); len(rules) != 1 || rules[0]["outboundTag"] != "direct" {
		t.Fatalf("rules = %v, want the relay's traffic to leave through direct", rules)
	}
	buildWithXray(t, cfg)
}

func TestABackpackTunnelWithoutARelayPortAndADisabledOneChangeNothing(t *testing.T) {
	legacy := backpackEnd(model.TunnelRolePortal)
	legacy.RelayPort = 0
	off := backpackEnd(model.TunnelRoleBridge)
	off.Enable = false
	original := `[{"tag":"direct","protocol":"freedom"}]`
	cfg := &xray.Config{OutboundConfigs: []byte(original)}
	if err := Apply(cfg, []model.ServerTunnel{legacy, off}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if string(cfg.OutboundConfigs) != original || len(cfg.InboundConfigs) != 0 || len(cfg.RouterConfig) != 0 {
		t.Fatalf("config changed for tunnels with nothing to inject: outbounds %s, inbounds %d, routing %s",
			cfg.OutboundConfigs, len(cfg.InboundConfigs), cfg.RouterConfig)
	}
}
