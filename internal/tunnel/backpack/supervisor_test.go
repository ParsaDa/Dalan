package backpack

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/engine"
)

// fakeEngine stands in for the BackPack binary: it records its pid and config path, then waits.
func fakeEngine(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "backpack")
	script := "#!/bin/sh\necho \"$$ $2\" >> \"$(dirname \"$2\")/starts.log\"\nexec sleep 600\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake engine: %v", err)
	}
	return path
}

func newSupervisor(t *testing.T) *engine.Supervisor {
	t.Helper()
	s := NewSupervisor(fakeEngine(t), t.TempDir())
	s.RestartDelay = 50 * time.Millisecond
	t.Cleanup(s.StopAll)
	return s
}

func starts(t *testing.T, s *engine.Supervisor, want int) []string {
	t.Helper()
	var lines []string
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		raw, _ := os.ReadFile(filepath.Join(s.Dir, "starts.log"))
		lines = strings.Fields(strings.ReplaceAll(strings.TrimSpace(string(raw)), "\n", " "))
		if len(lines) >= want*2 {
			return lines
		}
	}
	t.Fatalf("engine started %d time(s), want %d: %v", len(lines)/2, want, lines)
	return nil
}

func alive(pid string) bool {
	var n int
	for _, c := range pid {
		n = n*10 + int(c-'0')
	}
	return syscall.Kill(n, 0) == nil
}

func bridge(name, transport string) model.ServerTunnel {
	return model.ServerTunnel{
		Name: name, Role: model.TunnelRoleBridge, Engine: model.TunnelEngineBackpack, Enable: true,
		Address: "203.0.113.10", Port: 8443, UUID: "8d6f3a0e-5c1b-4f0e-9a51-2f6d1c7b9e10", Transport: transport,
	}
}

func TestReconcileRunsOneEnginePerEnabledBackpackTunnel(t *testing.T) {
	s := newSupervisor(t)
	xrayTunnel := bridge("x1", "tcp")
	xrayTunnel.Engine = model.TunnelEngineXray
	disabled := bridge("off", "tcp")
	disabled.Enable = false

	if err := s.Reconcile([]model.ServerTunnel{bridge("de1", "tcp"), xrayTunnel, disabled}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	log := starts(t, s, 1)
	configPath := filepath.Join(s.Dir, "de1.toml")
	if log[1] != configPath {
		t.Fatalf("engine was started with %q, want -c %q", log[1], configPath)
	}
	want, _ := RenderConfig(bridge("de1", "tcp"))
	got, err := os.ReadFile(configPath)
	if err != nil || string(got) != want {
		t.Fatalf("config file = %q (%v), want %q", got, err, want)
	}
	if info, _ := os.Stat(configPath); info.Mode().Perm() != 0o600 {
		t.Fatalf("config file mode = %v, want 0600: it holds the tunnel token", info.Mode().Perm())
	}
	for _, name := range []string{"x1.toml", "off.toml"} {
		if _, err := os.Stat(filepath.Join(s.Dir, name)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("%s exists; only enabled BackPack tunnels get a config", name)
		}
	}
}

// The panel's default engine path is relative ("bin/backpack") while the engine runs inside Dir.
func TestReconcileStartsAnEngineGivenByARelativePath(t *testing.T) {
	folder, err := os.MkdirTemp(".", "engine-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(folder) })
	if err := os.Rename(fakeEngine(t), filepath.Join(folder, "backpack")); err != nil {
		t.Skipf("cannot place the stand-in engine beside the package: %v", err)
	}
	s := NewSupervisor(filepath.Join(folder, "backpack"), t.TempDir())
	s.RestartDelay = 50 * time.Millisecond
	t.Cleanup(s.StopAll)

	if err := s.Reconcile([]model.ServerTunnel{bridge("de1", "tcp")}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	starts(t, s, 1)
}

func TestReconcileRestartsOnlyWhenTheConfigChangedAndStopsRemovedTunnels(t *testing.T) {
	s := newSupervisor(t)
	if err := s.Reconcile([]model.ServerTunnel{bridge("de1", "tcp")}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	first := starts(t, s, 1)[0]

	if err := s.Reconcile([]model.ServerTunnel{bridge("de1", "tcp")}); err != nil {
		t.Fatalf("Reconcile unchanged: %v", err)
	}
	time.Sleep(150 * time.Millisecond)
	if log := starts(t, s, 1); len(log) != 2 || !alive(first) {
		t.Fatalf("an unchanged tunnel was restarted: %v", log)
	}

	if err := s.Reconcile([]model.ServerTunnel{bridge("de1", "wssmux")}); err != nil {
		t.Fatalf("Reconcile changed: %v", err)
	}
	second := starts(t, s, 2)[2]
	if alive(first) || !alive(second) {
		t.Fatalf("after a transport change the old engine (alive=%v) must be gone and a new one (alive=%v) running", alive(first), alive(second))
	}

	if err := s.Reconcile(nil); err != nil {
		t.Fatalf("Reconcile none: %v", err)
	}
	if alive(second) {
		t.Fatal("the engine of a removed tunnel is still running")
	}
	if _, err := os.Stat(filepath.Join(s.Dir, "de1.toml")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("the config of a removed tunnel, which holds its token, was left on disk")
	}
}

func TestACrashedEngineIsStartedAgain(t *testing.T) {
	s := newSupervisor(t)
	if err := s.Reconcile([]model.ServerTunnel{bridge("de1", "tcp")}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	first := starts(t, s, 1)[0]
	var pid int
	for _, c := range first {
		pid = pid*10 + int(c-'0')
	}
	if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
		t.Fatalf("kill engine: %v", err)
	}
	if second := starts(t, s, 2)[2]; second == first || !alive(second) {
		t.Fatalf("after a crash the engine was not started again (pids %s, %s)", first, second)
	}
}

func TestReconcileSaysWhenTheEngineIsNotInstalled(t *testing.T) {
	s := NewSupervisor(filepath.Join(t.TempDir(), "backpack"), t.TempDir())
	err := s.Reconcile([]model.ServerTunnel{bridge("de1", "tcp")})
	if !errors.Is(err, ErrNotInstalled) {
		t.Fatalf("Reconcile error = %v, want ErrNotInstalled", err)
	}
	if err := s.Reconcile(nil); err != nil {
		t.Fatalf("with no BackPack tunnels a missing engine is not an error, got %v", err)
	}
}

func TestAnEnginesOutputReachesThePanelLogAndTheLastLinesAccompanyACrash(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "backpack")
	script := "#!/bin/sh\necho \"listening on 8443\"\necho \"dial failed: refused\" >&2\nexit 3\n"
	if err := os.WriteFile(binary, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake engine: %v", err)
	}
	var mu sync.Mutex
	var lines []string
	s := NewSupervisor(binary, t.TempDir())
	s.RestartDelay = time.Hour
	s.LogLine = func(tunnel, line string) {
		mu.Lock()
		defer mu.Unlock()
		lines = append(lines, tunnel+": "+line)
	}
	t.Cleanup(s.StopAll)
	if err := s.Reconcile([]model.ServerTunnel{bridge("de1", "tcp")}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	want := []string{"de1: listening on 8443", "de1: dial failed: refused"}
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		mu.Lock()
		got := strings.Join(lines, "|")
		mu.Unlock()
		if strings.Contains(got, want[0]) && strings.Contains(got, want[1]) {
			return
		}
	}
	mu.Lock()
	defer mu.Unlock()
	t.Fatalf("engine output reached the log as %q, want both of %q", lines, want)
}
