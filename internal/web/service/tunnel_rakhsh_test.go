package service

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/engine"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/rakhsh"
)

const statusWithLanes = `{"target_lanes":4,"lanes_up":%d,"links":[{"label":"wss 203.0.113.10:7000","rtt_ms":48.5,"loss_pct":0.4}]}`

// fakeRakhsh stands in for the engine: `run` stays up and `status` prints what the test wrote to status.json.
func fakeRakhsh(t *testing.T) (*engine.Supervisor, func(lanesUp int)) {
	t.Helper()
	dir := t.TempDir()
	binary := filepath.Join(dir, "rakhsh")
	script := `#!/bin/sh
if [ "$1" = "status" ]; then exec cat "$(dirname "$0")/status.json"; fi
exec sleep 600
`
	if err := os.WriteFile(binary, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake engine: %v", err)
	}
	sup := rakhsh.NewSupervisor(binary, t.TempDir())
	previous := rakhshEngines
	rakhshEngines = sup
	t.Cleanup(func() {
		sup.StopAll()
		rakhshEngines = previous
	})
	setLanes := func(up int) {
		body := strings.Replace(statusWithLanes, "%d", string(rune('0'+up)), 1)
		if err := os.WriteFile(filepath.Join(dir, "status.json"), []byte(body), 0o644); err != nil {
			t.Fatalf("write status: %v", err)
		}
	}
	setLanes(3)
	return sup, setLanes
}

func rakhshPortalSpec(name string) tunnel.PortalSpec {
	return tunnel.PortalSpec{
		Name: name, Address: "203.0.113.10", Port: 7000, Engine: model.TunnelEngineRakhsh, Transport: "wss",
		ForwardPorts: []string{"443=127.0.0.1:2096"}, Options: map[string]any{"sni": "www.example.com"},
	}
}

func TestARakhshTunnelRunsItsEngineAndIsConnectedOnlyWhileALaneIsUp(t *testing.T) {
	svc := newTunnelDB(t)
	sup, setLanes := fakeRakhsh(t)
	portal, err := svc.CreatePortal(rakhshPortalSpec("rk1"), nil)
	if err != nil {
		t.Fatalf("CreatePortal: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(sup.Dir, "rk1.toml"))
	if err != nil {
		t.Fatalf("the engine was given no config: %v", err)
	}
	if !strings.Contains(string(raw), `control_socket = "`+rakhsh.ControlSocket(sup.Dir, "rk1")+`"`) {
		t.Fatalf("config lacks the tunnel's own control socket:\n%s", raw)
	}
	if tags := inboundTags(t); hasTag(tags, "tunnel-rk1-link") {
		t.Fatalf("a Rakhsh tunnel was also injected into Xray: %v", tags)
	}

	status, err := svc.Status(portal.Id)
	if err != nil || !status.Connected || status.LanesUp != 3 || status.TargetLanes != 4 || status.LatencyMs != 49 || status.LossPct != 0.4 {
		t.Fatalf("Status with three lanes = %+v, %v; want connected, 3 of 4 lanes, 49 ms, 0.4%% loss", status, err)
	}
	setLanes(0)
	status, err = svc.Status(portal.Id)
	if err != nil || status.Connected || status.Error != "no lane is up (0 of 4)" {
		t.Fatalf("Status with no lane = %+v, %v; want not connected with the lane count", status, err)
	}
}

func TestARakhshTunnelWithoutTheEngineIsCreatedAndSaysWhatIsMissing(t *testing.T) {
	svc := newTunnelDB(t)
	previous := rakhshEngines
	rakhshEngines = rakhsh.NewSupervisor(filepath.Join(t.TempDir(), "rakhsh"), t.TempDir())
	t.Cleanup(func() { rakhshEngines = previous })

	portal, err := svc.CreatePortal(rakhshPortalSpec("rk1"), nil)
	if err != nil {
		t.Fatalf("CreatePortal without the engine: %v", err)
	}
	status, err := svc.Status(portal.Id)
	if err != nil || status.Connected || status.Error != rakhsh.ErrNotInstalled.Error() {
		t.Fatalf("Status = %+v, %v; want not connected, saying exactly that Rakhsh is not installed", status, err)
	}
	info, err := svc.EngineInfo("rakhsh")
	if err != nil || info.Engine != "rakhsh" || info.Installed || info.Pinned != rakhsh.PinnedVersion {
		t.Fatalf("EngineInfo(rakhsh) = %+v, %v; want not installed with the pinned release on offer", info, err)
	}
	if _, err := svc.EngineInfo("xray"); !errors.Is(err, errEngineHasNoBinary) {
		t.Fatalf("EngineInfo(xray) error = %v, want one saying Xray has no engine to manage", err)
	}
}

func TestTwoRakhshTunnelsGetTheirOwnControlSocketsAndConfigs(t *testing.T) {
	svc := newTunnelDB(t)
	sup, _ := fakeRakhsh(t)
	first, second := rakhshPortalSpec("rk1"), rakhshPortalSpec("rk2")
	second.Port = 7001
	for _, spec := range []tunnel.PortalSpec{first, second} {
		if _, err := svc.CreatePortal(spec, nil); err != nil {
			t.Fatalf("CreatePortal %s: %v", spec.Name, err)
		}
	}
	for _, name := range []string{"rk1", "rk2"} {
		raw, err := os.ReadFile(filepath.Join(sup.Dir, name+".toml"))
		if err != nil || !strings.Contains(string(raw), name+".rakhsh.sock") {
			t.Fatalf("%s config = %q, %v; want its own control socket", name, raw, err)
		}
	}
}

func TestARakhshTunnelWithARelayPortIsARoutableXrayOutbound(t *testing.T) {
	svc := newTunnelDB(t)
	fakeRakhsh(t)
	spec := rakhshPortalSpec("rk1")
	spec.ForwardPorts = nil
	portal, err := svc.CreatePortal(spec, []string{"in-443"})
	if err != nil {
		t.Fatalf("CreatePortal: %v", err)
	}
	if portal.RelayPort == 0 {
		t.Fatal("the Rakhsh portal has no relay port")
	}
	if tags := outboundTags(t); !hasTag(tags, "tunnel-rk1") {
		t.Fatalf("outbounds = %v, want the Rakhsh tunnel as tunnel-rk1", tags)
	}
}

func TestARakhshTunnelNeverShowsItsSecretInTheList(t *testing.T) {
	svc := newTunnelDB(t)
	fakeRakhsh(t)
	portal, err := svc.CreatePortal(rakhshPortalSpec("rk1"), nil)
	if err != nil {
		t.Fatalf("CreatePortal: %v", err)
	}
	list, err := svc.ListWithTraffic()
	if err != nil {
		t.Fatalf("ListWithTraffic: %v", err)
	}
	raw, _ := json.Marshal(list)
	secret := stored(t, portal.Id).RakhshSecret
	if secret == "" || strings.Contains(string(raw), secret) || strings.Contains(string(raw), "rakhshSecret") {
		t.Fatalf("the list carries the portal's secret key or its field name: %s", raw)
	}
}

func TestEditingARakhshEntryTunnelKeepsItsKeysAndSendsTheNodeTheNewLinkFirst(t *testing.T) {
	svc := newTunnelDB(t)
	fakeRakhsh(t)
	fake, node := newFakeTunnelNode(t)
	portal, err := svc.CreateOnNode(node.Id, rakhshPortalSpec("rk1"), nil, "")
	if err != nil {
		t.Fatalf("CreateOnNode: %v", err)
	}
	before := stored(t, portal.Id)

	_, err = svc.Update(portal.Id, TunnelEdit{
		Address: "203.0.113.11", Port: portal.Port, Transport: "quic", ForwardPorts: []string{"443=127.0.0.1:2096", "8443=2097"},
		Options: map[string]any{"sni": "cdn.example.com"},
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	row := stored(t, portal.Id)
	if row.Transport != "quic" || row.Address != "203.0.113.11" || len(row.ForwardPorts) != 2 || row.Options["sni"] != "cdn.example.com" {
		t.Fatalf("stored tunnel = %+v, want the edit applied", row)
	}
	if row.RakhshSecret != before.RakhshSecret || row.RakhshPeerPublic != before.RakhshPeerPublic || row.UUID != before.UUID {
		t.Fatal("editing the entry end changed its keys or token, which would orphan the exit end")
	}
	bridge := lastBridge(t, fake)
	if bridge.Transport != "quic" || bridge.Address != "203.0.113.11" || bridge.Options["sni"] != "cdn.example.com" {
		t.Fatalf("the node was sent %+v, want the edited carrier, address and options", bridge)
	}
}

func TestARakhshEditTheNodeRefusesOrTheEngineCannotReadChangesNothing(t *testing.T) {
	svc := newTunnelDB(t)
	fakeRakhsh(t)
	fake, node := newFakeTunnelNode(t)
	portal, err := svc.CreateOnNode(node.Id, rakhshPortalSpec("rk1"), nil, "")
	if err != nil {
		t.Fatalf("CreateOnNode: %v", err)
	}
	same := TunnelEdit{Address: portal.Address, Port: portal.Port, Transport: "quic", ForwardPorts: portal.ForwardPorts}

	unreadable := same
	unreadable.Options = map[string]any{"no_such_option": true}
	before := len(fake.links)
	if _, err := svc.Update(portal.Id, unreadable); err == nil || len(fake.links) != before {
		t.Fatalf("an option Rakhsh does not read: error %v, links sent %d -> %d; want a refusal that never reached the node", err, before, len(fake.links))
	}

	fake.mu.Lock()
	fake.fail = true
	fake.mu.Unlock()
	if _, err := svc.Update(portal.Id, same); err == nil || !strings.Contains(err.Error(), "node") {
		t.Fatalf("Update error = %v, want one saying the node did not take the change", err)
	}
	if row := stored(t, portal.Id); row.Transport != "wss" {
		t.Fatalf("carrier = %q after a refused edit, want wss kept", row.Transport)
	}
}

func TestEditingARakhshExitTunnelChangesOnlyWhatThisEndOwns(t *testing.T) {
	svc := newTunnelDB(t)
	fakeRakhsh(t)
	portal, err := tunnel.NewPortal(rakhshPortalSpec("rk1"))
	if err != nil {
		t.Fatalf("NewPortal: %v", err)
	}
	link, err := tunnel.SetupLink(portal)
	if err != nil {
		t.Fatalf("SetupLink: %v", err)
	}
	bridge, err := svc.Import(link, "direct")
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	before := stored(t, bridge.Id)
	_, err = svc.Update(bridge.Id, TunnelEdit{Address: "198.51.100.99", Port: 1, Transport: "quic", ExitOutbound: "warp", Options: map[string]any{"sni": "cdn.example.com"}})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	row := stored(t, bridge.Id)
	if row.ExitOutbound != "warp" || row.Options["sni"] != "cdn.example.com" || row.Address != bridge.Address || row.Port != bridge.Port || row.Transport != "wss" {
		t.Fatalf("exit end = %+v, want the exit outbound and options changed and the shared address, port and carrier kept", row)
	}
	if row.RakhshSecret != before.RakhshSecret || row.RakhshPeerPublic != before.RakhshPeerPublic {
		t.Fatal("editing the exit end changed its keys")
	}
}

func TestDeletingARakhshTunnelMadeForANodeRemovesBothEnds(t *testing.T) {
	svc := newTunnelDB(t)
	sup, _ := fakeRakhsh(t)
	fake, node := newFakeTunnelNode(t)
	portal, err := svc.CreateOnNode(node.Id, rakhshPortalSpec("rk1"), nil, "")
	if err != nil {
		t.Fatalf("CreateOnNode: %v", err)
	}
	if len(fake.ends) != 1 {
		t.Fatalf("node holds %d ends, want the exit end of rk1", len(fake.ends))
	}
	if err := svc.Delete(portal.Id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	var left int64
	database.GetDB().Model(&model.ServerTunnel{}).Count(&left)
	if left != 0 || len(fake.ends) != 0 {
		t.Fatalf("after Delete %d local rows and %d node ends remain, want none", left, len(fake.ends))
	}
	if _, err := os.Stat(filepath.Join(sup.Dir, "rk1.toml")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("deleting the tunnel left its engine config, and so its engine, in place")
	}
}

func TestTwoDirectRakhshTunnelsNeverShareALayer3Address(t *testing.T) {
	svc := newTunnelDB(t)
	fakeRakhsh(t)
	layer3 := func(name string, port int, addr string) tunnel.PortalSpec {
		spec := rakhshPortalSpec(name)
		spec.Mode, spec.Port, spec.ForwardPorts = model.TunnelModeDirect, port, nil
		spec.Options = map[string]any{"l3_addr": addr}
		return spec
	}
	if _, err := svc.CreatePortal(layer3("rk1", 7000, "10.20.0.1/24"), nil); err != nil {
		t.Fatalf("CreatePortal: %v", err)
	}
	if _, err := svc.CreatePortal(layer3("rk2", 7001, "10.20.0.2/24"), nil); err == nil || !strings.Contains(err.Error(), "10.20.0.2/24") {
		t.Fatalf("second tunnel on the first one's mirrored address error = %v, want one naming the address", err)
	}
	if _, err := svc.CreatePortal(layer3("rk3", 7002, "10.30.0.1/24"), nil); err != nil {
		t.Fatalf("a tunnel on another network was refused: %v", err)
	}
}

func TestTwoDirectRakhshTunnelsWithoutLayer3AreNotMistakenForClashing(t *testing.T) {
	svc := newTunnelDB(t)
	fakeRakhsh(t)
	for i, name := range []string{"rk1", "rk2"} {
		spec := rakhshPortalSpec(name)
		spec.Mode, spec.Port = model.TunnelModeDirect, 7000+i
		if _, err := svc.CreatePortal(spec, nil); err != nil {
			t.Fatalf("CreatePortal %s: %v", name, err)
		}
	}
}

func TestALocalRakhshEditWithAnOptionTheEngineCannotReadIsRefusedAndStoresNothing(t *testing.T) {
	svc := newTunnelDB(t)
	fakeRakhsh(t)
	portal, err := svc.CreatePortal(rakhshPortalSpec("rk1"), nil)
	if err != nil {
		t.Fatalf("CreatePortal: %v", err)
	}
	_, err = svc.Update(portal.Id, TunnelEdit{
		Address: portal.Address, Port: portal.Port, Transport: "quic", ForwardPorts: portal.ForwardPorts,
		Options: map[string]any{"no_such_option": true},
	})
	if err == nil || !strings.Contains(err.Error(), "no_such_option") {
		t.Fatalf("Update error = %v, want one naming the option Rakhsh does not read", err)
	}
	if row := stored(t, portal.Id); row.Transport != "wss" || row.Options["no_such_option"] != nil {
		t.Fatalf("stored tunnel = %+v after a refused edit, want it untouched", row)
	}
}

func directLayer3Spec(name string, port int, addr string) tunnel.PortalSpec {
	spec := rakhshPortalSpec(name)
	spec.Mode, spec.Port, spec.ForwardPorts = model.TunnelModeDirect, port, nil
	spec.Options = map[string]any{"l3_addr": addr}
	return spec
}

func TestARakhshLayer3AddressMayNotClashWithABackpackDirectPair(t *testing.T) {
	svc := newTunnelDB(t)
	fakeBackpack(t)
	fakeRakhsh(t)
	spec := directNodeSpecWithAddress()
	spec.Name = "gm1"
	spec.Options = map[string]any{"carrier": "udp", "local_ip": "10.10.5.1/30", "peer_ip": "10.10.5.2"}
	if _, err := svc.CreatePortal(spec, nil); err != nil {
		t.Fatalf("CreatePortal backpack: %v", err)
	}
	if _, err := svc.CreatePortal(directLayer3Spec("rk1", 7000, "10.10.5.1/30"), nil); err == nil || !strings.Contains(err.Error(), "10.10.5.1/30") {
		t.Fatalf("a Rakhsh tunnel on the BackPack tunnel's address error = %v, want one naming the address", err)
	}
}

func TestEditingARakhshEntryOntoAnotherTunnelsLayer3AddressIsRefused(t *testing.T) {
	svc := newTunnelDB(t)
	fakeRakhsh(t)
	if _, err := svc.CreatePortal(directLayer3Spec("rk1", 7000, "10.20.0.1/24"), nil); err != nil {
		t.Fatalf("CreatePortal rk1: %v", err)
	}
	second, err := svc.CreatePortal(directLayer3Spec("rk2", 7001, "10.30.0.1/24"), nil)
	if err != nil {
		t.Fatalf("CreatePortal rk2: %v", err)
	}
	edit := TunnelEdit{
		Address: second.Address, Port: second.Port, Transport: second.Transport,
		Options: map[string]any{"l3_addr": "10.20.0.1/24"},
	}
	if _, err := svc.Update(second.Id, edit); err == nil || !strings.Contains(err.Error(), "10.20.0.1/24") {
		t.Fatalf("editing onto a taken layer-3 address error = %v, want one naming the address", err)
	}
	stored, _ := svc.get(second.Id)
	if stored.Options["l3_addr"] != "10.30.0.1/24" {
		t.Fatalf("the refused edit changed the stored address to %v", stored.Options["l3_addr"])
	}
	edit.Options = map[string]any{"l3_addr": "10.30.0.1/24", "sni": "www.example.com"}
	if _, err := svc.Update(second.Id, edit); err != nil {
		t.Fatalf("an edit that keeps the tunnel's own address was refused: %v", err)
	}
}
