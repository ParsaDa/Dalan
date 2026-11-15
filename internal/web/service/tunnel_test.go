package service

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database/dbtest"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel"
)

func newTunnelDB(t *testing.T) *TunnelService {
	t.Helper()
	dbtest.InitDB(t, filepath.Join(t.TempDir(), "x-ui.db"))
	return &TunnelService{}
}

func inboundTags(t *testing.T) []string {
	t.Helper()
	cfg, err := (&XrayService{}).GetXrayConfig()
	if err != nil {
		t.Fatalf("GetXrayConfig: %v", err)
	}
	tags := make([]string, 0, len(cfg.InboundConfigs))
	for _, in := range cfg.InboundConfigs {
		tags = append(tags, in.Tag)
	}
	return tags
}

func hasTag(tags []string, want string) bool {
	for _, tag := range tags {
		if tag == want {
			return true
		}
	}
	return false
}

func TestAPortalTunnelReachesTheGeneratedXrayConfigUntilDisabledOrDeleted(t *testing.T) {
	svc := newTunnelDB(t)
	portal, err := svc.CreatePortal(tunnel.PortalSpec{
		Name: "de1", Address: "203.0.113.10", Port: 8443, Security: model.TunnelSecurityNone,
	}, nil)
	if err != nil {
		t.Fatalf("CreatePortal: %v", err)
	}
	if tags := inboundTags(t); !hasTag(tags, "tunnel-de1-link") {
		t.Fatalf("inbound tags = %v, want tunnel-de1-link among them", tags)
	}

	if err := svc.SetEnable(portal.Id, false); err != nil {
		t.Fatalf("SetEnable: %v", err)
	}
	if tags := inboundTags(t); hasTag(tags, "tunnel-de1-link") {
		t.Fatalf("a disabled tunnel is still in the config: %v", tags)
	}

	if err := svc.SetEnable(portal.Id, true); err != nil {
		t.Fatalf("SetEnable: %v", err)
	}
	if err := svc.Delete(portal.Id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if tags := inboundTags(t); hasTag(tags, "tunnel-de1-link") {
		t.Fatalf("a deleted tunnel is still in the config: %v", tags)
	}
}

func TestImportingAPortalsLinkCreatesTheBridgeEnd(t *testing.T) {
	svc := newTunnelDB(t)
	portal, err := tunnel.NewPortal(tunnel.PortalSpec{
		Name: "ir1", Address: "203.0.113.10", Port: 8443,
		Security: model.TunnelSecurityReality, ServerName: "www.example.com",
	})
	if err != nil {
		t.Fatalf("NewPortal: %v", err)
	}
	link, err := tunnel.SetupLink(portal)
	if err != nil {
		t.Fatalf("SetupLink: %v", err)
	}

	bridge, err := svc.Import(link, "")
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if bridge.Id == 0 || bridge.Role != model.TunnelRoleBridge || bridge.UUID != portal.UUID {
		t.Fatalf("bridge = %+v, want a stored bridge with the portal's uuid", bridge)
	}
	cfg, err := (&XrayService{}).GetXrayConfig()
	if err != nil {
		t.Fatalf("GetXrayConfig: %v", err)
	}
	if !strings.Contains(string(cfg.OutboundConfigs), `"tag":"tunnel-ir1-dial"`) {
		t.Fatalf("outbounds lack the bridge dial outbound: %s", cfg.OutboundConfigs)
	}
}

func TestTunnelsRefuseNamesAndPortsAlreadyTaken(t *testing.T) {
	svc := newTunnelDB(t)
	spec := tunnel.PortalSpec{Name: "de1", Address: "203.0.113.10", Port: 8443, Security: model.TunnelSecurityNone}
	if _, err := svc.CreatePortal(spec, nil); err != nil {
		t.Fatalf("CreatePortal: %v", err)
	}

	if _, err := svc.CreatePortal(spec, nil); err == nil || !strings.Contains(err.Error(), "name") {
		t.Fatalf("duplicate name error = %v, want one mentioning the name", err)
	}
	spec.Name = "de2"
	if _, err := svc.CreatePortal(spec, nil); err == nil || !strings.Contains(err.Error(), "port 8443") {
		t.Fatalf("duplicate port error = %v, want one mentioning port 8443", err)
	}
}

func TestSetupLinkIsServedForPortalsOnly(t *testing.T) {
	svc := newTunnelDB(t)
	portal, err := svc.CreatePortal(tunnel.PortalSpec{
		Name: "de1", Address: "203.0.113.10", Port: 8443, Security: model.TunnelSecurityNone,
	}, nil)
	if err != nil {
		t.Fatalf("CreatePortal: %v", err)
	}
	link, err := svc.SetupLink(portal.Id)
	if err != nil || !strings.HasPrefix(link, "dalan://tunnel/") {
		t.Fatalf("SetupLink = %q, %v; want a dalan://tunnel/ link", link, err)
	}
	if _, err := svc.SetupLink(portal.Id + 100); err == nil {
		t.Fatal("SetupLink for a missing tunnel returned no error")
	}
}

func TestStatusProbesAnEntryTunnelAndRefusesAnExitTunnel(t *testing.T) {
	svc := newTunnelDB(t)
	portal, err := svc.CreatePortal(tunnel.PortalSpec{
		Name: "de1", Address: "203.0.113.10", Port: 8443, Security: model.TunnelSecurityNone,
	}, nil)
	if err != nil {
		t.Fatalf("CreatePortal: %v", err)
	}
	// No Xray runs in this test, so the probe port is closed: that is a status, not a failure.
	status, err := svc.Status(portal.Id)
	if err != nil || status.Connected || status.Error == "" {
		t.Fatalf("Status(portal) = %+v, %v; want a not-connected status carrying the reason", status, err)
	}

	entry, err := tunnel.NewPortal(tunnel.PortalSpec{
		Name: "ir1", Address: "203.0.113.20", Port: 9443, Security: model.TunnelSecurityNone,
	})
	if err != nil {
		t.Fatalf("NewPortal: %v", err)
	}
	link, _ := tunnel.SetupLink(entry)
	bridge, err := svc.Import(link, "")
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if _, err := svc.Status(bridge.Id); err == nil || !strings.Contains(err.Error(), "entry") {
		t.Fatalf("Status(bridge) error = %v, want one saying only the entry end can be tested", err)
	}
}
