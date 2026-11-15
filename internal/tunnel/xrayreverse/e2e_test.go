//go:build !race

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

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel"
	"github.com/mhsanaei/3x-ui/v3/internal/xray"
)

// Excluded under -race: xray-core's VLESS reverse monitor reads its handler while New is
// still writing it (proxy/vless/outbound), a race inside the dependency, not in this package.
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
		probe := tunnel.ProbeVia(fmt.Sprintf("127.0.0.1:%d", ProbePort(portalTunnel)), target.URL, 3*time.Second)
		if !probe.Connected || probe.Error != "" {
			t.Fatalf("probe through the tunnel's own probe port = %+v, want connected", probe)
		}
		return
	}
	t.Fatalf("no response through the tunnel within 15s: %v", lastErr)
}
