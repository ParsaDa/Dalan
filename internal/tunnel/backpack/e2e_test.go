package backpack

import (
	"fmt"
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

func reservePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// Needs the real engine: DALAN_BACKPACK_E2E_BINARY=/path/to/backpack go test ./internal/tunnel/backpack/
func TestARealEngineCarriesAForwardedPortFromThePortalToTheBridge(t *testing.T) {
	binary := os.Getenv("DALAN_BACKPACK_E2E_BINARY")
	if binary == "" {
		t.Skip("DALAN_BACKPACK_E2E_BINARY is not set")
	}

	backend, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("backend: %v", err)
	}
	t.Cleanup(func() { backend.Close() })
	go func() {
		for {
			conn, err := backend.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				_, _ = io.WriteString(conn, "from the bridge side\n")
			}()
		}
	}()

	tunnelPort, userPort := reservePort(t), reservePort(t)
	portalEnd := model.ServerTunnel{
		Name: "e2e", Role: model.TunnelRolePortal, Engine: model.TunnelEngineBackpack, Enable: true,
		Address: "127.0.0.1", Port: tunnelPort, UUID: "8d6f3a0e-5c1b-4f0e-9a51-2f6d1c7b9e10", Transport: "tcpmux",
		ForwardPorts: []string{fmt.Sprintf("%d=%s", userPort, backend.Addr())},
	}
	bridgeEnd := portalEnd
	bridgeEnd.Role, bridgeEnd.ForwardPorts = model.TunnelRoleBridge, nil

	portal := NewSupervisor(binary, t.TempDir())
	bridge := NewSupervisor(binary, t.TempDir())
	t.Cleanup(portal.StopAll)
	t.Cleanup(bridge.StopAll)
	if err := portal.Reconcile([]model.ServerTunnel{portalEnd}); err != nil {
		t.Fatalf("portal Reconcile: %v", err)
	}
	if err := bridge.Reconcile([]model.ServerTunnel{bridgeEnd}); err != nil {
		t.Fatalf("bridge Reconcile: %v", err)
	}

	var lastErr error
	for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); time.Sleep(300 * time.Millisecond) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", userPort), time.Second)
		if err != nil {
			lastErr = err
			continue
		}
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		buf := make([]byte, 64)
		n, err := conn.Read(buf)
		conn.Close()
		if err != nil {
			lastErr = err
			continue
		}
		if got := string(buf[:n]); got != "from the bridge side\n" {
			t.Fatalf("read %q through the forwarded port, want the backend's greeting", got)
		}
		return
	}
	t.Fatalf("nothing came through the forwarded port within 20s: %v", lastErr)
}
