package service

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/dbtest"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/backpack"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/engine"
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

// fakeBackpack stands in for the engine: it writes the status file BackPack would, then waits.
func fakeBackpack(t *testing.T) *engine.Supervisor {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "backpack")
	script := `#!/bin/sh
name=$(basename "$2" .toml)
printf '{"name":"%s","connected":true,"peer":"198.51.100.7:40000","bytes_in":10,"bytes_out":20}' "$name" > "$(dirname "$2")/$name.metrics.json"
exec sleep 600
`
	if err := os.WriteFile(binary, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake engine: %v", err)
	}
	sup := backpack.NewSupervisor(binary, t.TempDir())
	previous := tunnelEngines
	tunnelEngines = sup
	t.Cleanup(func() {
		sup.StopAll()
		tunnelEngines = previous
	})
	return sup
}

func backpackPortalSpec() tunnel.PortalSpec {
	return tunnel.PortalSpec{
		Name: "bp1", Address: "203.0.113.10", Port: 8443, Engine: model.TunnelEngineBackpack,
		Transport: "tcpmux", ForwardPorts: []string{"443=127.0.0.1:2096"},
	}
}

func waitForStatus(t *testing.T, svc *TunnelService, id int, connected bool) tunnel.Status {
	t.Helper()
	var status tunnel.Status
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(30 * time.Millisecond) {
		var err error
		if status, err = svc.Status(id); err != nil {
			t.Fatalf("Status: %v", err)
		}
		if status.Connected == connected {
			return status
		}
	}
	t.Fatalf("status never became connected=%v: %+v", connected, status)
	return status
}

func TestABackpackTunnelRunsItsEngineAndReportsTheEnginesStatus(t *testing.T) {
	svc := newTunnelDB(t)
	sup := fakeBackpack(t)

	portal, err := svc.CreatePortal(backpackPortalSpec(), nil)
	if err != nil {
		t.Fatalf("CreatePortal: %v", err)
	}
	if tags := inboundTags(t); hasTag(tags, "tunnel-bp1-link") {
		t.Fatalf("a BackPack tunnel was also injected into Xray: %v", tags)
	}
	status := waitForStatus(t, svc, portal.Id, true)
	if status.ExitIP != "198.51.100.7" {
		t.Fatalf("status = %+v, want the connected peer 198.51.100.7 as the exit address", status)
	}

	if err := svc.SetEnable(portal.Id, false); err != nil {
		t.Fatalf("SetEnable: %v", err)
	}
	if _, err := os.Stat(filepath.Join(sup.Dir, "bp1.toml")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("disabling the tunnel left its engine config, and so its engine, in place")
	}
}

func TestABackpackTunnelWithoutTheEngineIsCreatedAndSaysWhatIsMissing(t *testing.T) {
	svc := newTunnelDB(t)
	previous := tunnelEngines
	tunnelEngines = backpack.NewSupervisor(filepath.Join(t.TempDir(), "backpack"), t.TempDir())
	t.Cleanup(func() { tunnelEngines = previous })

	portal, err := svc.CreatePortal(backpackPortalSpec(), nil)
	if err != nil {
		t.Fatalf("CreatePortal without the engine: %v", err)
	}
	status, err := svc.Status(portal.Id)
	if err != nil || status.Connected || !strings.Contains(status.Error, "not installed") {
		t.Fatalf("Status = %+v, %v; want not connected, saying the engine is not installed", status, err)
	}
	if info := svc.EngineInfo(); info.Installed || info.Version != "" || info.Pinned != backpack.PinnedVersion {
		t.Fatalf("EngineInfo = %+v, want not installed and no installed version and the pinned one on offer", info)
	}
}

func TestAManagementTunnelIsNotProbedAndCannotBeDeletedFromTheTunnelList(t *testing.T) {
	svc := newTunnelDB(t)
	portal, err := svc.CreateManagementPortal(tunnel.PortalSpec{
		Name: "node-de1", Address: "203.0.113.10", Port: 8443, Security: model.TunnelSecurityNone,
	})
	if err != nil {
		t.Fatalf("CreateManagementPortal: %v", err)
	}
	if portal.ManagePort < managePortFirst || portal.ManagePort > managePortLast {
		t.Fatalf("management port %d is outside %d-%d", portal.ManagePort, managePortFirst, managePortLast)
	}
	// A probe would be answered by the node's panel and read as a broken tunnel.
	if _, err := svc.Status(portal.Id); err == nil || !strings.Contains(err.Error(), "node's status") {
		t.Fatalf("Status(management tunnel) error = %v, want a pointer to the node's status", err)
	}
	if err := svc.Delete(portal.Id); !errors.Is(err, errManagementTunnel) {
		t.Fatalf("Delete(management tunnel) = %v, want errManagementTunnel", err)
	}
	if !hasTag(inboundTags(t), "tunnel-node-de1-manage") {
		t.Fatalf("the management inbound is missing from the generated Xray config: %v", inboundTags(t))
	}
	if err := svc.DeleteManagement(portal.Id); err != nil {
		t.Fatalf("DeleteManagement: %v", err)
	}
	if hasTag(inboundTags(t), "tunnel-node-de1-manage") {
		t.Fatalf("the management inbound survived DeleteManagement")
	}
}

func outboundTags(t *testing.T) []string {
	t.Helper()
	cfg, err := (&XrayService{}).GetXrayConfig()
	if err != nil {
		t.Fatalf("GetXrayConfig: %v", err)
	}
	var outbounds []struct {
		Tag string `json:"tag"`
	}
	if err := json.Unmarshal(cfg.OutboundConfigs, &outbounds); err != nil {
		t.Fatalf("outbounds: %v", err)
	}
	tags := make([]string, 0, len(outbounds))
	for _, o := range outbounds {
		tags = append(tags, o.Tag)
	}
	return tags
}

func TestABackpackTunnelIsARoutableXrayOutboundAndNeedsNoForwardedPort(t *testing.T) {
	svc := newTunnelDB(t)
	fakeBackpack(t)
	spec := backpackPortalSpec()
	spec.ForwardPorts = nil
	portal, err := svc.CreatePortal(spec, []string{"in-443"})
	if err != nil {
		t.Fatalf("CreatePortal: %v", err)
	}
	if portal.RelayPort == 0 {
		t.Fatal("the BackPack portal has no relay port")
	}
	if tags := outboundTags(t); !hasTag(tags, "tunnel-bp1") {
		t.Fatalf("outbounds = %v, want the BackPack tunnel as tunnel-bp1", tags)
	}
}

func TestARelayPortAlreadyUsedByATunnelOrAnInboundIsRefused(t *testing.T) {
	svc := newTunnelDB(t)
	fakeBackpack(t)
	portal, err := svc.CreatePortal(backpackPortalSpec(), nil)
	if err != nil {
		t.Fatalf("CreatePortal: %v", err)
	}
	link, err := svc.SetupLink(portal.Id)
	if err != nil {
		t.Fatalf("SetupLink: %v", err)
	}
	clash, err := tunnel.ParseSetupLink(link)
	if err != nil {
		t.Fatalf("ParseSetupLink: %v", err)
	}
	clash.Name = "bp2"
	if err := svc.create(&clash); err == nil || !strings.Contains(err.Error(), "already used") {
		t.Fatalf("creating a tunnel whose relay port belongs to tunnel bp1: err = %v, want 'already used'", err)
	}

	taken := &model.Inbound{UserId: 1, Tag: "in-taken", Enable: true, Listen: "0.0.0.0", Port: 63555, Protocol: model.VLESS, Settings: `{"clients":[]}`}
	if err := database.GetDB().Create(taken).Error; err != nil {
		t.Fatalf("create inbound: %v", err)
	}
	clash.Name, clash.RelayPort = "bp3", 63555
	if err := svc.create(&clash); err == nil || !strings.Contains(err.Error(), "already used") {
		t.Fatalf("creating a tunnel whose relay port is an inbound's port: err = %v, want 'already used'", err)
	}
}

func TestRoutableOutboundsListOnlyEnabledEntryTunnelsRoutingCanUse(t *testing.T) {
	svc := newTunnelDB(t)
	fakeBackpack(t)
	xrayPortal, err := svc.CreatePortal(tunnel.PortalSpec{
		Name: "x1", Address: "203.0.113.10", Port: 8443, Security: model.TunnelSecurityNone,
	}, nil)
	if err != nil {
		t.Fatalf("CreatePortal xray: %v", err)
	}
	bpSpec := backpackPortalSpec()
	bpSpec.Port = 8444
	if _, err := svc.CreatePortal(bpSpec, nil); err != nil {
		t.Fatalf("CreatePortal backpack: %v", err)
	}
	if _, err := svc.CreateManagementPortal(tunnel.PortalSpec{
		Name: "node-de1", Address: "203.0.113.10", Port: 8445, Security: model.TunnelSecurityNone,
	}); err != nil {
		t.Fatalf("CreateManagementPortal: %v", err)
	}
	off, err := svc.CreatePortal(tunnel.PortalSpec{
		Name: "off1", Address: "203.0.113.10", Port: 8446, Security: model.TunnelSecurityNone,
	}, nil)
	if err != nil {
		t.Fatalf("CreatePortal off: %v", err)
	}
	if err := svc.SetEnable(off.Id, false); err != nil {
		t.Fatalf("SetEnable: %v", err)
	}
	bridge := model.ServerTunnel{
		Name: "exit1", Role: model.TunnelRoleBridge, Engine: model.TunnelEngineXray, Enable: true,
		Address: "203.0.113.10", Port: 8447, UUID: xrayPortal.UUID, Security: model.TunnelSecurityNone,
	}
	if err := svc.create(&bridge); err != nil {
		t.Fatalf("create bridge: %v", err)
	}

	got, err := svc.RoutableOutbounds()
	if err != nil {
		t.Fatalf("RoutableOutbounds: %v", err)
	}
	var tags []string
	for _, o := range got {
		tags = append(tags, o.Tag+"/"+o.Engine)
	}
	if want := "tunnel-x1/xray,tunnel-bp1/backpack"; strings.Join(tags, ",") != want {
		t.Fatalf("routable outbounds = %v, want %s", tags, want)
	}
}
