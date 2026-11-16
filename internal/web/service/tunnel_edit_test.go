package service

import (
	"strconv"
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel"
)

func lastBridge(t *testing.T, fake *fakeTunnelNode) model.ServerTunnel {
	t.Helper()
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if len(fake.links) == 0 {
		t.Fatal("the node was given no setup link")
	}
	bridge, err := tunnel.ParseSetupLink(fake.links[len(fake.links)-1])
	if err != nil {
		t.Fatalf("ParseSetupLink: %v", err)
	}
	return bridge
}

func stored(t *testing.T, id int) model.ServerTunnel {
	t.Helper()
	var row model.ServerTunnel
	if err := database.GetDB().First(&row, id).Error; err != nil {
		t.Fatalf("load tunnel %d: %v", id, err)
	}
	return row
}

func nodeBackpackSpec() tunnel.PortalSpec {
	spec := backpackPortalSpec()
	spec.Options = map[string]any{"nodelay": true}
	return spec
}

func TestEditingAnEntryTunnelKeepsItsStoredSettingsAndSendsTheNodeTheNewLink(t *testing.T) {
	svc := newTunnelDB(t)
	fakeBackpack(t)
	fake, node := newFakeTunnelNode(t)
	portal, err := svc.CreateOnNode(node.Id, nodeBackpackSpec(), nil, "")
	if err != nil {
		t.Fatalf("CreateOnNode: %v", err)
	}

	edited, err := svc.Update(portal.Id, TunnelEdit{
		Address: "203.0.113.11", Port: portal.Port, Transport: "ws", ForwardPorts: []string{"443", "8443"},
		Options: map[string]any{"nodelay": true, "edge_ip": "198.51.100.7"},
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	row := stored(t, portal.Id)
	if edited.Transport != "ws" || row.Transport != "ws" || row.Address != "203.0.113.11" || len(row.ForwardPorts) != 2 ||
		row.Options["edge_ip"] != "198.51.100.7" || row.UUID != portal.UUID || row.NodeId != node.Id {
		t.Fatalf("stored tunnel = %+v, want the edit applied and the token and node kept", row)
	}
	bridge := lastBridge(t, fake)
	if bridge.Transport != "ws" || bridge.Address != "203.0.113.11" || bridge.Options["edge_ip"] != "198.51.100.7" {
		t.Fatalf("the node was sent %+v, want the edited transport, address and options", bridge)
	}
}

func TestAnEditTheNodeRefusesIsUndoneHere(t *testing.T) {
	svc := newTunnelDB(t)
	fakeBackpack(t)
	fake, node := newFakeTunnelNode(t)
	portal, err := svc.CreateOnNode(node.Id, nodeBackpackSpec(), nil, "")
	if err != nil {
		t.Fatalf("CreateOnNode: %v", err)
	}
	fake.mu.Lock()
	fake.fail = true
	fake.mu.Unlock()

	_, err = svc.Update(portal.Id, TunnelEdit{Address: portal.Address, Port: portal.Port, Transport: "ws", ForwardPorts: portal.ForwardPorts})
	if err == nil || !strings.Contains(err.Error(), "node") {
		t.Fatalf("Update error = %v, want one saying the node did not take the change", err)
	}
	if row := stored(t, portal.Id); row.Transport != "tcpmux" {
		t.Fatalf("transport = %q after a refused edit, want tcpmux kept", row.Transport)
	}
}

func TestAnEditWithAnOptionTheEngineCannotReadChangesNothingAnywhere(t *testing.T) {
	svc := newTunnelDB(t)
	fakeBackpack(t)
	fake, node := newFakeTunnelNode(t)
	portal, err := svc.CreateOnNode(node.Id, nodeBackpackSpec(), nil, "")
	if err != nil {
		t.Fatalf("CreateOnNode: %v", err)
	}
	before := len(fake.links)

	_, err = svc.Update(portal.Id, TunnelEdit{
		Address: portal.Address, Port: portal.Port, Transport: "tcpmux", ForwardPorts: portal.ForwardPorts,
		Options: map[string]any{"no_such_option": true},
	})
	if err == nil {
		t.Fatal("Update accepted an option the engine does not read")
	}
	if len(fake.links) != before {
		t.Fatal("a refused edit still reached the node")
	}
}

func TestMovingAnEntryTunnelOntoAnInboundsPortIsRefusedButKeepingItsOwnIsNot(t *testing.T) {
	svc := newTunnelDB(t)
	fakeBackpack(t)
	portal, err := svc.CreatePortal(backpackPortalSpec(), nil)
	if err != nil {
		t.Fatalf("CreatePortal: %v", err)
	}
	if err := database.GetDB().Create(&model.Inbound{Tag: "in-9000", Port: 9000, Protocol: "vless", Enable: true}).Error; err != nil {
		t.Fatalf("create inbound: %v", err)
	}
	same := TunnelEdit{Address: portal.Address, Port: portal.Port, Transport: "tcpmux", ForwardPorts: portal.ForwardPorts}
	if _, err := svc.Update(portal.Id, same); err != nil {
		t.Fatalf("Update with its own port: %v", err)
	}
	moved := same
	moved.Port = 9000
	if _, err := svc.Update(portal.Id, moved); err == nil || !strings.Contains(err.Error(), "already used") {
		t.Fatalf("Update onto an inbound's port error = %v, want 'already used'", err)
	}
}

func TestEditingAnExitTunnelChangesOnlyWhatThisEndOwns(t *testing.T) {
	svc := newTunnelDB(t)
	fakeBackpack(t)
	link, err := func() (string, error) {
		portal, err := tunnel.NewPortal(nodeBackpackSpec())
		if err != nil {
			return "", err
		}
		return tunnel.SetupLink(portal)
	}()
	if err != nil {
		t.Fatalf("setup link: %v", err)
	}
	bridge, err := svc.Import(link, "direct")
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	_, err = svc.Update(bridge.Id, TunnelEdit{Address: "198.51.100.99", Port: 1, Transport: "ws", ExitOutbound: "warp", Options: map[string]any{"nodelay": false}})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	row := stored(t, bridge.Id)
	if row.ExitOutbound != "warp" || row.Options["nodelay"] != false || row.Address != bridge.Address || row.Port != bridge.Port || row.Transport != "tcpmux" {
		t.Fatalf("exit end = %+v, want the exit outbound and options changed and the shared address, port and transport kept", row)
	}
}

func TestImportingAnExistingTunnelsLinkUpdatesItInPlaceAndRefusesAnotherKind(t *testing.T) {
	svc := newTunnelDB(t)
	fakeBackpack(t)
	portal, err := tunnel.NewPortal(nodeBackpackSpec())
	if err != nil {
		t.Fatalf("NewPortal: %v", err)
	}
	first, _ := tunnel.SetupLink(portal)
	bridge, err := svc.Import(first, "warp")
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	portal.Transport = "ws"
	portal.Options = map[string]any{"nodelay": true, "edge_ip": "198.51.100.7"}
	second, _ := tunnel.SetupLink(portal)

	again, err := svc.Import(second, "")
	if err != nil {
		t.Fatalf("Import again: %v", err)
	}
	row := stored(t, bridge.Id)
	if again.Id != bridge.Id || row.Transport != "ws" || row.Options["edge_ip"] != "198.51.100.7" || row.ExitOutbound != "warp" {
		t.Fatalf("tunnel after the second link = %+v, want the same row with the new transport and options and the old exit outbound", row)
	}

	own, err := svc.CreatePortal(tunnel.PortalSpec{Name: "own", Address: "203.0.113.10", Port: 8500, Security: model.TunnelSecurityNone}, nil)
	if err != nil {
		t.Fatalf("CreatePortal: %v", err)
	}
	clash := tunnel.PortalSpec{Name: own.Name, Address: "203.0.113.10", Port: 8501, Security: model.TunnelSecurityNone}
	clashPortal, _ := tunnel.NewPortal(clash)
	clashLink, _ := tunnel.SetupLink(clashPortal)
	if _, err := svc.Import(clashLink, ""); err == nil {
		t.Fatal("Import let a link overwrite an entry end of the same name")
	}
}

func directNodeSpec() tunnel.PortalSpec {
	return tunnel.PortalSpec{
		Name: "gm1", Port: 9000, Engine: model.TunnelEngineBackpack, Mode: model.TunnelModeDirect,
		ForwardPorts: []string{"27015"}, AcceptUDP: true, Options: map[string]any{"carrier": "udp"},
	}
}

func TestADirectTunnelForANodeDialsTheNodesAddressAndTheNodeListens(t *testing.T) {
	svc := newTunnelDB(t)
	fakeBackpack(t)
	fake, node := newFakeTunnelNode(t)
	if err := database.GetDB().Create(&model.Inbound{Tag: "in-9000", Port: 9000, Protocol: "vless", Enable: true}).Error; err != nil {
		t.Fatalf("create inbound: %v", err)
	}

	portal, err := svc.CreateOnNode(node.Id, directNodeSpec(), nil, "")
	if err != nil {
		t.Fatalf("CreateOnNode: %v", err)
	}
	if portal.Mode != model.TunnelModeDirect || portal.Address != node.Address {
		t.Fatalf("entry end = %+v, want a direct tunnel that dials the node's address %s", portal, node.Address)
	}
	bridge := lastBridge(t, fake)
	if bridge.Mode != model.TunnelModeDirect || bridge.Role != model.TunnelRoleBridge || bridge.Port != 9000 {
		t.Fatalf("node end = %+v, want a direct exit end listening on 9000", bridge)
	}
}

func TestADirectTunnelsEntryEndReportsTheLinkInterfaceNotAConnectedPeer(t *testing.T) {
	svc := newTunnelDB(t)
	fakeBackpack(t)
	portal, err := svc.CreatePortal(directNodeSpecWithAddress(), nil)
	if err != nil {
		t.Fatalf("CreatePortal: %v", err)
	}
	previous := directInterfaceUp
	t.Cleanup(func() { directInterfaceUp = previous })

	directInterfaceUp = func(name string) bool { return name == "dl"+itoa(portal.Id) }
	status, err := svc.Status(portal.Id)
	if err != nil || !status.Connected || status.ExitIP != "203.0.113.20" {
		t.Fatalf("Status with the interface up = %+v, %v; want connected with the exit address", status, err)
	}
	directInterfaceUp = func(string) bool { return false }
	status, err = svc.Status(portal.Id)
	if err != nil || status.Connected || !strings.Contains(status.Error, "dl"+itoa(portal.Id)) {
		t.Fatalf("Status with the interface missing = %+v, %v; want not connected, naming the interface", status, err)
	}
}

func directNodeSpecWithAddress() tunnel.PortalSpec {
	spec := directNodeSpec()
	spec.Address = "203.0.113.20"
	return spec
}

func itoa(n int) string { return strconv.Itoa(n) }

func TestTwoDirectTunnelsOnOneServerNeverShareAnAddressPair(t *testing.T) {
	svc := newTunnelDB(t)
	fakeBackpack(t)
	named := func(name string) tunnel.PortalSpec {
		spec := directNodeSpecWithAddress()
		spec.Name = name
		spec.Options = map[string]any{"carrier": "udp", "local_ip": "10.10.5.1/30", "peer_ip": "10.10.5.2"}
		return spec
	}
	if _, err := svc.CreatePortal(named("gm1"), nil); err != nil {
		t.Fatalf("CreatePortal: %v", err)
	}
	if _, err := svc.CreatePortal(named("gm2"), nil); err == nil || !strings.Contains(err.Error(), "10.10.5.1/30") {
		t.Fatalf("second tunnel on the same pair error = %v, want one naming the pair", err)
	}
	auto := directNodeSpecWithAddress()
	auto.Name = "gm3"
	auto.Options = map[string]any{"carrier": "udp"}
	one, err := svc.CreatePortal(auto, nil)
	if err != nil {
		t.Fatalf("CreatePortal with a generated pair: %v", err)
	}
	if one.Options["local_ip"] == "10.10.5.1/30" {
		t.Fatalf("generated pair %v clashes with the first tunnel", one.Options["local_ip"])
	}
}
