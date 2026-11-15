package xrayreverse

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel"
	"github.com/mhsanaei/3x-ui/v3/internal/xray"
	_ "github.com/xtls/xray-core/app/dispatcher"
	_ "github.com/xtls/xray-core/app/log"
	_ "github.com/xtls/xray-core/app/policy"
	_ "github.com/xtls/xray-core/app/proxyman/inbound"
	_ "github.com/xtls/xray-core/app/proxyman/outbound"
	_ "github.com/xtls/xray-core/app/router"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/infra/conf"
	_ "github.com/xtls/xray-core/proxy/blackhole"
	_ "github.com/xtls/xray-core/proxy/freedom"
	_ "github.com/xtls/xray-core/proxy/socks"
	_ "github.com/xtls/xray-core/proxy/vless/inbound"
	_ "github.com/xtls/xray-core/proxy/vless/outbound"
	_ "github.com/xtls/xray-core/transport/internet/tcp"
	"golang.org/x/net/proxy"
)

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func startXray(t *testing.T, cfg *xray.Config) {
	t.Helper()
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	var parsed conf.Config
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("xray rejected config JSON: %v\n%s", err, raw)
	}
	built, err := parsed.Build()
	if err != nil {
		t.Fatalf("xray rejected config: %v\n%s", err, raw)
	}
	instance, err := core.New(built)
	if err != nil {
		t.Fatalf("create xray instance: %v", err)
	}
	if err := instance.Start(); err != nil {
		t.Fatalf("start xray: %v", err)
	}
	t.Cleanup(func() { _ = instance.Close() })
}

// The portal's only ordinary outbound is a blackhole, so a response can only arrive through the bridge.
func TestUserTrafficOnThePortalLeavesFromTheBridge(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "reached through the bridge")
	}))
	t.Cleanup(target.Close)

	socksPort, linkPort := freePort(t), freePort(t)
	portalTunnel, err := tunnel.NewPortal(tunnel.PortalSpec{
		Name: "de1", Address: "127.0.0.1", Port: linkPort, Security: model.TunnelSecurityNone,
	})
	if err != nil {
		t.Fatalf("NewPortal: %v", err)
	}
	portalTunnel.RouteInbounds = []string{"users"}
	link, err := tunnel.SetupLink(portalTunnel)
	if err != nil {
		t.Fatalf("SetupLink: %v", err)
	}
	bridgeTunnel, err := tunnel.ParseSetupLink(link)
	if err != nil {
		t.Fatalf("ParseSetupLink: %v", err)
	}

	portal := &xray.Config{
		InboundConfigs: []xray.InboundConfig{{
			Listen: []byte(`"127.0.0.1"`), Port: socksPort, Protocol: "mixed",
			Settings: []byte(`{"auth":"noauth"}`), Tag: "users",
		}},
		OutboundConfigs: []byte(`[{"tag":"blocked","protocol":"blackhole"}]`),
	}
	if err := Apply(portal, []model.ServerTunnel{portalTunnel}); err != nil {
		t.Fatalf("Apply portal: %v", err)
	}
	// Freedom refuses private targets for tunnelled traffic by default; the test target is on loopback.
	bridge := &xray.Config{OutboundConfigs: []byte(
		`[{"tag":"direct","protocol":"freedom","settings":{"finalRules":[{"action":"allow","ip":["127.0.0.0/8"]}]}}]`)}
	if err := Apply(bridge, []model.ServerTunnel{bridgeTunnel}); err != nil {
		t.Fatalf("Apply bridge: %v", err)
	}
	startXray(t, portal)
	startXray(t, bridge)

	dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("127.0.0.1:%d", socksPort), nil, proxy.Direct)
	if err != nil {
		t.Fatalf("socks dialer: %v", err)
	}
	client := &http.Client{Timeout: 3 * time.Second, Transport: &http.Transport{
		DialContext: func(_ context.Context, network, addr string) (net.Conn, error) { return dialer.Dial(network, addr) },
	}}

	var lastErr error
	for deadline := time.Now().Add(15 * time.Second); time.Now().Before(deadline); time.Sleep(300 * time.Millisecond) {
		resp, err := client.Get(target.URL)
		if err != nil {
			lastErr = err
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if string(body) != "reached through the bridge" {
			t.Fatalf("body = %q, want %q", body, "reached through the bridge")
		}
		return
	}
	t.Fatalf("no response through the tunnel within 15s: %v", lastErr)
}

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
