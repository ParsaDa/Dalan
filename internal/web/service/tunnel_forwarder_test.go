package service

import (
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/forwarder"
)

// fakeForwarder stands in for the engine binary of one forwarding engine: it stays up and does nothing else.
func fakeForwarder(t *testing.T, id string, installed bool) string {
	t.Helper()
	e, ok := forwarder.ByID(id)
	if !ok {
		t.Fatalf("no forwarding engine %q", id)
	}
	binary := filepath.Join(t.TempDir(), id)
	if installed {
		if err := os.WriteFile(binary, []byte("#!/bin/sh\nexec sleep 600\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	sup := e.NewSupervisor(binary, t.TempDir())
	previous := forwarderEngines[id]
	forwarderEngines[id] = sup
	t.Cleanup(func() {
		sup.StopAll()
		forwarderEngines[id] = previous
	})
	return sup.Dir
}

func gostPortalSpec(name string) tunnel.PortalSpec {
	return tunnel.PortalSpec{
		Name: name, Address: "203.0.113.10", Port: 8443, Engine: model.TunnelEngineGost, Transport: "wss",
		ForwardPorts: []string{"2222=127.0.0.1:22"},
	}
}

func listenSocksOn(t *testing.T, port int) {
	t.Helper()
	l, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		t.Skipf("relay port %d is taken on this machine: %v", port, err)
	}
	t.Cleanup(func() { l.Close() })
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				buf := make([]byte, 3)
				if _, err := c.Read(buf); err == nil {
					_, _ = c.Write([]byte{5, 2})
				}
			}()
		}
	}()
}

func TestAForwardingEngineTunnelRunsItsEngineAndIsConnectedOnlyWhileTheRelayAnswers(t *testing.T) {
	svc := newTunnelDB(t)
	dir := fakeForwarder(t, model.TunnelEngineGost, true)
	portal, err := svc.CreatePortal(gostPortalSpec("gs1"), nil)
	if err != nil {
		t.Fatalf("CreatePortal: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "gs1.yml"))
	if err != nil {
		t.Fatalf("the engine was given no config: %v", err)
	}
	if !strings.Contains(string(raw), "type: relay") || strings.Contains(string(raw), "tunnel-gs1-link") {
		t.Fatalf("unexpected GOST config:\n%s", raw)
	}
	if tags := inboundTags(t); hasTag(tags, "tunnel-gs1-link") {
		t.Fatalf("a forwarding-engine tunnel was also injected into Xray as a BackPack link: %v", tags)
	}

	status, err := svc.Status(portal.Id)
	if err != nil || status.Connected || !strings.Contains(status.Error, "does not answer") {
		t.Fatalf("Status with a silent relay = %+v, %v; want not connected, saying the exit does not answer", status, err)
	}
	listenSocksOn(t, portal.RelayPort)
	status, err = svc.Status(portal.Id)
	if err != nil || !status.Connected || status.LatencyMs < 1 {
		t.Fatalf("Status with an answering relay = %+v, %v; want connected with a delay", status, err)
	}
}

func TestAForwardingEngineTunnelWithoutTheEngineIsCreatedAndSaysWhatIsMissing(t *testing.T) {
	svc := newTunnelDB(t)
	fakeForwarder(t, model.TunnelEngineGost, false)
	portal, err := svc.CreatePortal(gostPortalSpec("gs1"), nil)
	if err != nil {
		t.Fatalf("CreatePortal without the engine: %v", err)
	}
	status, err := svc.Status(portal.Id)
	if err != nil || status.Connected || !strings.Contains(status.Error, "GOST is not installed") {
		t.Fatalf("Status = %+v, %v; want not connected, saying GOST is not installed", status, err)
	}
	info, err := svc.EngineInfo(model.TunnelEngineGost)
	if err != nil || info.Installed || info.Pinned != "v3.3.3" {
		t.Fatalf("EngineInfo = %+v, %v; want not installed and the pinned release on offer", info, err)
	}
}

func TestImportingAForwardingEngineLinkRunsTheExitEndWithTheSameSecret(t *testing.T) {
	entry := newTunnelDB(t)
	portal, err := entry.CreatePortal(gostPortalSpec("gs1"), nil)
	if err != nil {
		t.Fatal(err)
	}
	link, err := entry.SetupLink(portal.Id)
	if err != nil {
		t.Fatal(err)
	}

	exit := newTunnelDB(t)
	dir := fakeForwarder(t, model.TunnelEngineGost, true)
	bridge, err := exit.Import(link, "")
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if bridge.Role != model.TunnelRoleBridge || bridge.EngineSecret != portal.EngineSecret {
		t.Fatalf("bridge = %+v, want an exit end holding the portal's secret", bridge)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "gs1.yml"))
	if err != nil {
		t.Fatalf("the exit end was given no config: %v", err)
	}
	if !strings.Contains(string(raw), portal.EngineSecret) {
		t.Fatalf("the exit end's config does not carry the shared secret:\n%s", raw)
	}
}

func TestAForwardingEngineTunnelCannotCarryANodesManagement(t *testing.T) {
	svc := newTunnelDB(t)
	fakeForwarder(t, model.TunnelEngineGost, true)
	parent, err := svc.CreatePortal(gostPortalSpec("gs1"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateManagementRider(parent.Id, "node-de1"); err == nil || !strings.Contains(err.Error(), "BackPack or Rakhsh") {
		t.Fatalf("CreateManagementRider error = %v, want one naming BackPack and Rakhsh as the engines that can", err)
	}
}

func TestEditingAForwardingEngineTunnelChecksTheNewTransportAndKeepsTheOldOnRefusal(t *testing.T) {
	svc := newTunnelDB(t)
	fakeForwarder(t, model.TunnelEngineGost, true)
	portal, err := svc.CreatePortal(gostPortalSpec("gs1"), nil)
	if err != nil {
		t.Fatal(err)
	}
	edit := TunnelEdit{Address: portal.Address, Port: portal.Port, Transport: "carrier-pigeon", ForwardPorts: portal.ForwardPorts}
	if _, err := svc.Update(portal.Id, edit); err == nil || !strings.Contains(err.Error(), "transport") {
		t.Fatalf("Update with an unknown transport = %v, want a refusal naming the transport", err)
	}
	edit.Transport, edit.ForwardPorts = "grpc", []string{"2222=127.0.0.1:22", "8080=127.0.0.1:80"}
	updated, err := svc.Update(portal.Id, edit)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Transport != "grpc" || len(updated.ForwardPorts) != 2 || updated.EngineSecret != portal.EngineSecret {
		t.Fatalf("updated = %+v, want the new transport and forwards with the same secret", updated)
	}
}
