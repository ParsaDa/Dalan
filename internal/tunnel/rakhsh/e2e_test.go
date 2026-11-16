package rakhsh

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

const e2eUUID = "8d6f3a0e-5c1b-4f0e-9a51-2f6d1c7b9e10"

// Needs the real engine: DALAN_RAKHSH_E2E_BINARY=/path/to/rakhsh go test ./internal/tunnel/rakhsh/
func e2eBinary(t *testing.T) string {
	t.Helper()
	binary := os.Getenv("DALAN_RAKHSH_E2E_BINARY")
	if binary == "" {
		t.Skip("DALAN_RAKHSH_E2E_BINARY is not set")
	}
	return binary
}

func reservePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func e2ePair(t *testing.T, mode, transport string, forward string, port int) (portal, bridge model.ServerTunnel) {
	t.Helper()
	portalSecret, portalPublic, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("portal keys: %v", err)
	}
	bridgeSecret, bridgePublic, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("bridge keys: %v", err)
	}
	portal = model.ServerTunnel{
		Name: "e2e", Role: model.TunnelRolePortal, Engine: model.TunnelEngineRakhsh, Enable: true, Mode: mode,
		Address: "127.0.0.1", Port: port, UUID: e2eUUID, Transport: transport,
		RakhshSecret: portalSecret, RakhshPeerPublic: bridgePublic,
		ForwardPorts: []string{forward},
		Options:      map[string]any{"sni": "www.example.com"},
	}
	bridge = portal
	bridge.Role, bridge.ForwardPorts = model.TunnelRoleBridge, nil
	bridge.RakhshSecret, bridge.RakhshPeerPublic = bridgeSecret, portalPublic
	return portal, bridge
}

func TestARealEngineAcceptsTheConfigsThePanelRenders(t *testing.T) {
	binary := e2eBinary(t)
	for _, mode := range []string{model.TunnelModeReverse, model.TunnelModeDirect} {
		portal, bridge := e2ePair(t, mode, "tcp", "9001=127.0.0.1:9002", 7000)
		for _, end := range []model.ServerTunnel{portal, bridge} {
			t.Run(mode+"/"+end.Role, func(t *testing.T) {
				dir := t.TempDir()
				text, err := RenderConfig(end, ControlSocket(dir, end.Name))
				if err != nil {
					t.Fatalf("RenderConfig: %v", err)
				}
				path := filepath.Join(dir, "rakhsh.toml")
				if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
					t.Fatal(err)
				}
				out, err := exec.Command(binary, "check", "-c", path).CombinedOutput()
				if err != nil {
					t.Fatalf("rakhsh check rejected the config: %v\n%s\n--- config ---\n%s", err, out, text)
				}
			})
		}
	}
}

func TestARealEngineCarriesAForwardedPortFromThePortalToTheBridge(t *testing.T) {
	binary := e2eBinary(t)

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

	for _, transport := range []string{"tcp", "wss"} {
		t.Run(transport, func(t *testing.T) {
			tunnelPort, userPort := reservePort(t), reservePort(t)
			portalEnd, bridgeEnd := e2ePair(t, model.TunnelModeReverse, transport,
				fmt.Sprintf("%d=%s", userPort, backend.Addr()), tunnelPort)

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
				if err != nil && n == 0 {
					lastErr = err
					continue
				}
				if got := string(buf[:n]); got != "from the bridge side\n" {
					t.Fatalf("forwarded bytes = %q, want the backend's greeting", got)
				}
				return
			}
			t.Fatalf("no bytes crossed the %s tunnel within 20s: %v", transport, lastErr)
		})
	}
}
