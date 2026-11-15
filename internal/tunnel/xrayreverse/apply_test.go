package xrayreverse

import (
	"encoding/json"
	"testing"

	"github.com/xtls/xray-core/infra/conf"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel"
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
