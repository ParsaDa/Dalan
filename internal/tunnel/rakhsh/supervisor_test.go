package rakhsh

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/engine"
)

// standInEngine records its pid and arguments, then waits, so a test sees how Rakhsh was started.
func standInEngine(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "rakhsh")
	script := "#!/bin/sh\necho \"$$ $*\" >> \"$(dirname \"$3\")/starts.log\"\nexec sleep 600\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func startedWith(t *testing.T, dir string, want int) [][]string {
	t.Helper()
	var starts [][]string
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		raw, _ := os.ReadFile(filepath.Join(dir, "starts.log"))
		starts = nil
		for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
			if line != "" {
				starts = append(starts, strings.Fields(line))
			}
		}
		if len(starts) >= want {
			return starts
		}
	}
	t.Fatalf("engine started %d time(s), want %d: %v", len(starts), want, starts)
	return nil
}

func aliveProcess(pid string) bool {
	var n int
	for _, c := range pid {
		n = n*10 + int(c-'0')
	}
	return syscall.Kill(n, 0) == nil
}

func enabledPortal() model.ServerTunnel {
	t := portalTunnel()
	t.Enable = true
	return t
}

func newRakhshSupervisor(t *testing.T) *engine.Supervisor {
	t.Helper()
	s := NewSupervisor(standInEngine(t), t.TempDir())
	s.RestartDelay = 50 * time.Millisecond
	t.Cleanup(s.StopAll)
	return s
}

func TestSupervisorRunsRakhshWithRunAndItsConfigAndLeavesOtherEnginesAlone(t *testing.T) {
	s := newRakhshSupervisor(t)
	backpackTunnel := enabledPortal()
	backpackTunnel.Name, backpackTunnel.Engine = "bp1", model.TunnelEngineBackpack

	if err := s.Reconcile([]model.ServerTunnel{enabledPortal(), backpackTunnel}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	config := filepath.Join(s.Dir, "r1.toml")
	starts := startedWith(t, s.Dir, 1)
	if got := strings.Join(starts[0][1:], " "); got != "run -c "+config {
		t.Fatalf("engine was started as %q, want %q", got, "run -c "+config)
	}
	want, _ := RenderConfig(enabledPortal(), ControlSocket(s.Dir, "r1"))
	if got, err := os.ReadFile(config); err != nil || string(got) != want {
		t.Fatalf("config file = %q (%v), want the rendered config with this tunnel's own control socket", got, err)
	}
	if info, _ := os.Stat(config); info.Mode().Perm() != 0o600 {
		t.Fatalf("config mode = %v, want 0600: it holds the tunnel's secret key", info.Mode().Perm())
	}
	if _, err := os.Stat(filepath.Join(s.Dir, "bp1.toml")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("a BackPack tunnel got a Rakhsh config")
	}
}

func TestSupervisorRestartsOnAChangedConfigAndCleansUpAfterARemovedTunnel(t *testing.T) {
	s := newRakhshSupervisor(t)
	if err := s.Reconcile([]model.ServerTunnel{enabledPortal()}); err != nil {
		t.Fatal(err)
	}
	first := startedWith(t, s.Dir, 1)[0][0]

	changed := enabledPortal()
	changed.Options = map[string]any{"lanes": 8}
	if err := s.Reconcile([]model.ServerTunnel{changed}); err != nil {
		t.Fatal(err)
	}
	second := startedWith(t, s.Dir, 2)[1][0]
	if aliveProcess(first) || !aliveProcess(second) {
		t.Fatalf("after a config change the old process (alive=%v) must be gone and a new one (alive=%v) running", aliveProcess(first), aliveProcess(second))
	}

	socket := ControlSocket(s.Dir, "r1")
	if err := os.WriteFile(socket, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := s.Reconcile(nil); err != nil {
		t.Fatal(err)
	}
	if aliveProcess(second) {
		t.Fatal("the process of a removed tunnel is still running")
	}
	for _, left := range []string{filepath.Join(s.Dir, "r1.toml"), socket} {
		if _, err := os.Stat(left); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("%s was left behind by a removed tunnel", left)
		}
	}
}

func TestSupervisorSaysWhenRakhshIsNotInstalled(t *testing.T) {
	s := NewSupervisor(filepath.Join(t.TempDir(), "rakhsh"), t.TempDir())
	if err := s.Reconcile([]model.ServerTunnel{enabledPortal()}); !errors.Is(err, ErrNotInstalled) {
		t.Fatalf("Reconcile error = %v, want ErrNotInstalled", err)
	}
	if err := s.Reconcile(nil); err != nil {
		t.Fatalf("with no Rakhsh tunnels a missing engine is no error, got %v", err)
	}
}
