package tunnelroute

import (
	"encoding/json"
	"strings"
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

func TestApplyKeepsTunnelRulesAheadOfOperatorRulesAndSkipsDisabledTunnels(t *testing.T) {
	routing := `{"rules":[{"type":"field","network":"tcp,udp","outboundTag":"blocked"}]}`
	outbounds := `[{"tag":"direct","protocol":"freedom"}]`
	enabled := backpackEnd(model.TunnelRoleBridge)
	enabled.ExitOutbound = "warp"
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
	if len(got.Rules[0].InboundTag) != 1 || got.Rules[0].InboundTag[0] != "tunnel-bp-de1-relay" {
		t.Fatalf("tunnel rule matches %v, want [tunnel-bp-de1-relay]", got.Rules[0].InboundTag)
	}

	untouched := &xray.Config{RouterConfig: []byte(routing), OutboundConfigs: []byte(outbounds)}
	if err := Apply(untouched, []model.ServerTunnel{disabled}); err != nil {
		t.Fatalf("Apply disabled: %v", err)
	}
	if string(untouched.RouterConfig) != routing || string(untouched.OutboundConfigs) != outbounds {
		t.Fatalf("a disabled tunnel changed the config: %s %s", untouched.RouterConfig, untouched.OutboundConfigs)
	}
}

func TestATunnelOfTheRemovedXrayEngineAddsNothing(t *testing.T) {
	original := `[{"tag":"direct","protocol":"freedom"}]`
	for _, engine := range []string{"xray", ""} {
		xrayRow := backpackEnd(model.TunnelRolePortal)
		xrayRow.Engine = engine
		cfg := &xray.Config{OutboundConfigs: []byte(original)}
		if err := Apply(cfg, []model.ServerTunnel{xrayRow}); err != nil {
			t.Fatalf("Apply with engine %q: %v", engine, err)
		}
		if string(cfg.OutboundConfigs) != original || len(cfg.InboundConfigs) != 0 || len(cfg.RouterConfig) != 0 {
			t.Fatalf("a leftover %q tunnel changed the config: %s", engine, cfg.OutboundConfigs)
		}
	}
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

func poolTunnel(t *testing.T, name string, port int) model.ServerTunnel {
	t.Helper()
	portal, err := tunnel.NewPortal(tunnel.PortalSpec{
		Name: name, Address: "203.0.113.10", Port: port, Engine: model.TunnelEngineBackpack, Transport: "tcpmux",
	})
	if err != nil {
		t.Fatalf("NewPortal: %v", err)
	}
	return portal
}

func routingOf(t *testing.T, cfg *xray.Config) (rules, balancers []map[string]any) {
	t.Helper()
	var routing struct {
		Rules     []map[string]any `json:"rules"`
		Balancers []map[string]any `json:"balancers"`
	}
	if err := json.Unmarshal(cfg.RouterConfig, &routing); err != nil {
		t.Fatalf("routing: %v", err)
	}
	return routing.Rules, routing.Balancers
}

func TestAPoolBecomesABalancerOnItsActiveTunnelAndRoutesItsInbounds(t *testing.T) {
	a, b := poolTunnel(t, "a", 8443), poolTunnel(t, "b", 8444)
	tunnels := []model.ServerTunnel{a, b}
	pool := model.TunnelPool{
		Name: "main", Members: []string{"a", "b"}, RouteInbounds: []string{"in-1"}, Active: "b",
	}
	cfg := &xray.Config{OutboundConfigs: []byte(`[{"tag":"direct","protocol":"freedom"}]`)}
	if err := Apply(cfg, tunnels); err != nil {
		t.Fatal(err)
	}
	if err := ApplyPools(cfg, tunnels, []model.TunnelPool{pool}); err != nil {
		t.Fatalf("ApplyPools: %v", err)
	}
	buildWithXray(t, cfg)

	rules, balancers := routingOf(t, cfg)
	if len(balancers) != 1 || balancers[0]["tag"] != "pool-main" {
		t.Fatalf("balancers = %v, want one named pool-main", balancers)
	}
	selector, _ := balancers[0]["selector"].([]any)
	if len(selector) != 1 || selector[0] != "tunnel-b" {
		t.Fatalf("selector = %v, want only the active tunnel-b", selector)
	}
	var routed bool
	for _, r := range rules {
		if r["balancerTag"] == "pool-main" {
			routed = true
		}
	}
	if !routed {
		t.Fatalf("no rule sends in-1 to the pool: %v", rules)
	}
}

func TestAPoolFallsBackToTheFirstWorkingMemberAndIgnoresTunnelsItCannotUse(t *testing.T) {
	a, b := poolTunnel(t, "a", 8443), poolTunnel(t, "b", 8444)
	a.Enable = false
	tunnels := []model.ServerTunnel{a, b}
	cfg := &xray.Config{OutboundConfigs: []byte(`[{"tag":"direct","protocol":"freedom"}]`)}
	if err := Apply(cfg, tunnels); err != nil {
		t.Fatal(err)
	}
	pool := model.TunnelPool{Name: "main", Members: []string{"a", "gone", "b"}, Active: "a", RouteInbounds: []string{"in-1"}}
	if err := ApplyPools(cfg, tunnels, []model.TunnelPool{pool}); err != nil {
		t.Fatalf("ApplyPools: %v", err)
	}
	buildWithXray(t, cfg)
	_, balancers := routingOf(t, cfg)
	if selector, _ := balancers[0]["selector"].([]any); len(selector) != 1 || selector[0] != "tunnel-b" {
		t.Fatalf("selector = %v, want tunnel-b: a is disabled and gone does not exist", selector)
	}

	empty := &xray.Config{OutboundConfigs: []byte(`[{"tag":"direct","protocol":"freedom"}]`)}
	if err := ApplyPools(empty, nil, []model.TunnelPool{pool}); err != nil {
		t.Fatalf("ApplyPools without tunnels: %v", err)
	}
	if string(empty.RouterConfig) != "" {
		t.Fatalf("a pool with no usable member changed the routing: %s", empty.RouterConfig)
	}
}

func TestAnEntryTunnelSwitchesOutboundStatsOnSoItsTrafficCanBeCounted(t *testing.T) {
	policy := `{"levels":{"0":{"statsUserUplink":true}},"system":{"statsInboundUplink":true,"statsOutboundUplink":false,"statsOutboundDownlink":false}}`
	cfg := &xray.Config{OutboundConfigs: []byte(`[{"tag":"direct","protocol":"freedom"}]`), Policy: []byte(policy)}
	if err := Apply(cfg, []model.ServerTunnel{poolTunnel(t, "a", 8443)}); err != nil {
		t.Fatal(err)
	}
	var got struct {
		Levels map[string]map[string]bool `json:"levels"`
		System map[string]bool            `json:"system"`
	}
	if err := json.Unmarshal(cfg.Policy, &got); err != nil {
		t.Fatalf("policy: %v", err)
	}
	if !got.System["statsOutboundUplink"] || !got.System["statsOutboundDownlink"] || !got.System["statsInboundUplink"] {
		t.Fatalf("system policy = %v, want outbound stats on and the rest kept", got.System)
	}
	if !got.Levels["0"]["statsUserUplink"] {
		t.Fatalf("levels = %v, want the operator's level policy kept", got.Levels)
	}

	exit := poolTunnel(t, "b", 8444)
	exit.Role = model.TunnelRoleBridge
	untouched := &xray.Config{OutboundConfigs: []byte(`[{"tag":"direct","protocol":"freedom"}]`), Policy: []byte(policy)}
	if err := Apply(untouched, []model.ServerTunnel{exit}); err != nil {
		t.Fatal(err)
	}
	if string(untouched.Policy) != policy {
		t.Fatalf("an exit end changed the policy: %s", untouched.Policy)
	}
}

func TestRakhshRelayMakesAnEntryTunnelRoutable(t *testing.T) {
	entry := model.ServerTunnel{Name: "r1", Role: model.TunnelRolePortal, Engine: model.TunnelEngineRakhsh, Enable: true}
	if Routable(entry) {
		t.Fatal("a Rakhsh entry tunnel with no relay port is an outbound")
	}
	entry.RelayPort = 63500
	if !Routable(entry) {
		t.Fatal("a Rakhsh entry tunnel with a relay port is not an outbound")
	}
}

func TestRakhshRelayGetsTheSameXrayWiringAsBackpack(t *testing.T) {
	portal := model.ServerTunnel{
		Id: 7, Name: "r1", Role: model.TunnelRolePortal, Engine: model.TunnelEngineRakhsh, Enable: true,
		RelayPort: 63500, UUID: "token", RouteInbounds: []string{"in-1"},
	}
	cfg := &xray.Config{}
	if err := Apply(cfg, []model.ServerTunnel{portal}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cfg.OutboundConfigs), `"tag":"tunnel-r1"`) {
		t.Fatalf("no outbound tunnel-r1 in %s", cfg.OutboundConfigs)
	}
}
