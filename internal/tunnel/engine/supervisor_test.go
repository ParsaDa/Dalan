package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

// recordingEngine writes its arguments and one environment variable to a file in its config's directory, then waits.
func recordingEngine(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "engine")
	script := "#!/bin/sh\necho \"$* ENV=$ENGINE_TEST\" > \"$(dirname \"$2\")/started.log\"\nexec sleep 600\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func startedLog(t *testing.T, dir string) string {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if raw, err := os.ReadFile(filepath.Join(dir, "started.log")); err == nil && len(raw) > 0 {
			return strings.TrimSpace(string(raw))
		}
	}
	t.Fatal("the engine never started")
	return ""
}

func newTestSupervisor(t *testing.T, ext string, env []string) *Supervisor {
	t.Helper()
	s := &Supervisor{
		Name: "Test", Binary: recordingEngine(t), Dir: t.TempDir(), Ext: ext, Env: env,
		Wants:        func(model.ServerTunnel) bool { return true },
		Render:       func(model.ServerTunnel) (string, error) { return "config", nil },
		Args:         func(config string) []string { return []string{"-f", config} },
		NotInstalled: os.ErrNotExist,
	}
	t.Cleanup(s.StopAll)
	return s
}

func TestSupervisorUsesTheConfiguredSuffixAndEnvironment(t *testing.T) {
	s := newTestSupervisor(t, ".torrc", []string{"ENGINE_TEST=hello"})
	if err := s.Reconcile([]model.ServerTunnel{{Name: "a", Enable: true}}); err != nil {
		t.Fatal(err)
	}
	want := "-f " + filepath.Join(s.Dir, "a.torrc") + " ENV=hello"
	if got := startedLog(t, s.Dir); got != want {
		t.Fatalf("engine was started as %q, want %q", got, want)
	}
}

func TestSupervisorKeepsTomlAndTheInheritedEnvironmentByDefault(t *testing.T) {
	t.Setenv("ENGINE_TEST", "inherited")
	s := newTestSupervisor(t, "", nil)
	if err := s.Reconcile([]model.ServerTunnel{{Name: "a", Enable: true}}); err != nil {
		t.Fatal(err)
	}
	want := "-f " + filepath.Join(s.Dir, "a.toml") + " ENV=inherited"
	if got := startedLog(t, s.Dir); got != want {
		t.Fatalf("engine was started as %q, want %q", got, want)
	}
}

func TestRunningFollowsTheDesiredSet(t *testing.T) {
	s := newTestSupervisor(t, "", nil)
	if s.Running("a") {
		t.Fatal("a never started, yet Running says so")
	}
	if err := s.Reconcile([]model.ServerTunnel{{Name: "a", Enable: true}}); err != nil {
		t.Fatal(err)
	}
	if !s.Running("a") {
		t.Fatal("a started, yet Running says not")
	}
	if err := s.Reconcile(nil); err != nil {
		t.Fatal(err)
	}
	if s.Running("a") {
		t.Fatal("a was removed, yet Running still says so")
	}
}
