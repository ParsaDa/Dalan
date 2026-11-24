package forwarder

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

// DALAN_ENGINE_BINS names a folder holding the engines' own binaries (rathole, backhaul, gost, kariz,
// sing-box, xray); without it these tests are skipped, like the other real-engine tests.
func engineBinary(t *testing.T, name string) string {
	t.Helper()
	dir := os.Getenv("DALAN_ENGINE_BINS")
	if dir == "" {
		t.Skip("DALAN_ENGINE_BINS is not set")
	}
	path := filepath.Join(dir, name)
	if _, err := os.Stat(path); err != nil {
		t.Skipf("no %s binary in %s", name, dir)
	}
	return path
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func echoServer(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				line, err := bufio.NewReader(c).ReadString('\n')
				if err == nil {
					_, _ = c.Write([]byte("echo:" + line))
				}
			}()
		}
	}()
	return l.Addr().(*net.TCPAddr).Port
}

// runEnd starts one end of a tunnel as the engine would and stops it with the test.
func runEnd(t *testing.T, e Engine, binary string, tunnel model.ServerTunnel) {
	t.Helper()
	dir := t.TempDir()
	plan, err := e.PlanFor(tunnel, dir)
	if err != nil {
		t.Fatal(err)
	}
	text, err := e.Render(plan)
	if err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(dir, tunnel.Role+e.Ext)
	if err := os.WriteFile(config, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, binary, e.Args(config)...)
	cmd.Dir = dir
	out := &lockedBuffer{}
	cmd.Stdout, cmd.Stderr = out, out
	if err := cmd.Start(); err != nil {
		cancel()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cancel()
		_ = cmd.Wait()
		if t.Failed() {
			t.Logf("%s %s output:\n%s\nconfig:\n%s", e.ID, tunnel.Role, out.String(), text)
		}
	})
}

// carries says whether a line sent to listen comes back from the echo server behind the tunnel.
func carries(t *testing.T, listen string) bool {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		c, err := net.DialTimeout("tcp", listen, time.Second)
		if err == nil {
			_ = c.SetDeadline(time.Now().Add(3 * time.Second))
			if _, err := c.Write([]byte("ping\n")); err == nil {
				if got, err := bufio.NewReader(c).ReadString('\n'); err == nil && got == "echo:ping\n" {
					_ = c.Close()
					return true
				}
			}
			_ = c.Close()
		}
		time.Sleep(300 * time.Millisecond)
	}
	return false
}

// pairOf builds the two ends of one tunnel on loopback; the entry end opens a forward to an echo server
// the exit end reaches.
func pairOf(t *testing.T, e Engine, transport, mode string) (portal, bridge model.ServerTunnel, listen string) {
	t.Helper()
	port, entryPort, echo := freePort(t), freePort(t), echoServer(t)
	forward := fmt.Sprintf("127.0.0.1:%d=127.0.0.1:%d", entryPort, echo)
	base := model.ServerTunnel{
		Name: "t1", Engine: e.ID, Enable: true, Address: "127.0.0.1", Port: port, Mode: mode,
		Transport: transport, EngineSecret: "a-shared-secret-of-thirty-two-chars", UUID: "8d6f3a0e-5c1b-4f0e-9a51-2f6d1c7b9e10",
		ForwardPorts: []string{forward},
	}
	portal, bridge = base, base
	portal.Role, bridge.Role = model.TunnelRolePortal, model.TunnelRoleBridge
	return portal, bridge, "127.0.0.1:" + strconv.Itoa(entryPort)
}

func TestRatholeCarriesTrafficOnItsTransports(t *testing.T) {
	binary := engineBinary(t, "rathole")
	for _, transport := range []string{"tcp", "noise", "ws"} {
		t.Run(transport, func(t *testing.T) {
			portal, bridge, listen := pairOf(t, ratholeEngine, transport, model.TunnelModeReverse)
			runEnd(t, ratholeEngine, binary, portal)
			runEnd(t, ratholeEngine, binary, bridge)
			if !carries(t, listen) {
				t.Fatalf("no traffic crossed the %s tunnel", transport)
			}
		})
	}
}

func TestBackhaulCarriesTrafficOnItsTransports(t *testing.T) {
	binary := engineBinary(t, "backhaul")
	for _, tr := range backhaulEngine.Transports {
		t.Run(tr.ID, func(t *testing.T) {
			portal, bridge, listen := pairOf(t, backhaulEngine, tr.ID, model.TunnelModeReverse)
			runEnd(t, backhaulEngine, binary, portal)
			runEnd(t, backhaulEngine, binary, bridge)
			if !carries(t, listen) {
				t.Fatalf("no traffic crossed the %s tunnel", tr.ID)
			}
		})
	}
}

func TestGostCarriesTrafficInBothDirectionsOnEveryTransport(t *testing.T) {
	binary := engineBinary(t, "gost")
	for _, tr := range gostEngine.Transports {
		for _, mode := range []string{model.TunnelModeReverse, model.TunnelModeDirect} {
			t.Run(tr.ID+"/"+mode, func(t *testing.T) {
				portal, bridge, listen := pairOf(t, gostEngine, tr.ID, mode)
				runEnd(t, gostEngine, binary, portal)
				runEnd(t, gostEngine, binary, bridge)
				if !carries(t, listen) {
					t.Fatalf("no traffic crossed the %s %s tunnel", tr.ID, mode)
				}
			})
		}
	}
}

func TestKarizCarriesTrafficInBothDirectionsOnEveryTransport(t *testing.T) {
	binary := engineBinary(t, "kariz")
	for _, tr := range karizEngine.Transports {
		for _, mode := range []string{model.TunnelModeReverse, model.TunnelModeDirect} {
			t.Run(tr.ID+"/"+mode, func(t *testing.T) {
				portal, bridge, listen := pairOf(t, karizEngine, tr.ID, mode)
				runEnd(t, karizEngine, binary, portal)
				runEnd(t, karizEngine, binary, bridge)
				if !carries(t, listen) {
					t.Fatalf("no traffic crossed the %s %s tunnel", tr.ID, mode)
				}
			})
		}
	}
}

func TestSingboxCarriesTrafficOnEveryTransport(t *testing.T) {
	binary := engineBinary(t, "sing-box")
	for _, tr := range singboxEngine.Transports {
		t.Run(tr.ID, func(t *testing.T) {
			portal, bridge, listen := pairOf(t, singboxEngine, tr.ID, model.TunnelModeDirect)
			runEnd(t, singboxEngine, binary, bridge)
			runEnd(t, singboxEngine, binary, portal)
			if !carries(t, listen) {
				t.Fatalf("no traffic crossed the %s tunnel", tr.ID)
			}
		})
	}
}

func TestXrayCarriesTrafficOnEveryTransport(t *testing.T) {
	binary := engineBinary(t, "xray")
	for _, tr := range xrayCoreEngine.Transports {
		t.Run(tr.ID, func(t *testing.T) {
			portal, bridge, listen := pairOf(t, xrayCoreEngine, tr.ID, model.TunnelModeDirect)
			runEnd(t, xrayCoreEngine, binary, bridge)
			runEnd(t, xrayCoreEngine, binary, portal)
			if !carries(t, listen) {
				t.Fatalf("no traffic crossed the %s tunnel", tr.ID)
			}
		})
	}
}
