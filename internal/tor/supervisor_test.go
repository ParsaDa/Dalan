package tor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

// standInTor records how it was started, prints a bootstrap line like Tor does, then waits.
func standInTor(t *testing.T, layout Layout) {
	t.Helper()
	path := layout.Binary()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\necho \"$* LIB=$LD_LIBRARY_PATH\" >> \"" + filepath.Join(layout.Root, "starts.log") + "\"\n" +
		"echo 'Nov 17 09:30:01.000 [notice] Bootstrapped 45% (loading_descriptors): Loading relay descriptors'\nexec sleep 600\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

func startsLog(t *testing.T, layout Layout, want int) []string {
	t.Helper()
	var lines []string
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		raw, _ := os.ReadFile(filepath.Join(layout.Root, "starts.log"))
		lines = nil
		for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
			if line != "" {
				lines = append(lines, line)
			}
		}
		if len(lines) >= want {
			return lines
		}
	}
	t.Fatalf("tor started %d time(s), want %d: %v", len(lines), want, lines)
	return nil
}

func TestSupervisorStartsOneTorPerEnabledExitWithItsLibrariesAndRecordsProgress(t *testing.T) {
	layout := Layout{Root: t.TempDir(), BundleDir: t.TempDir()}
	standInTor(t, layout)
	var progress Progress
	s := NewSupervisor(layout, func() []string { return nil }, &progress)
	t.Cleanup(s.StopAll)

	exits := []model.TorExit{
		{Id: 1, Country: "de", Port: 62700, Enable: true},
		{Id: 2, Country: "nl", Port: 62701, Enable: false},
	}
	if err := s.Reconcile(ExitRows(exits)); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	starts := startsLog(t, layout, 1)
	want := "-f " + filepath.Join(layout.Root, "tor-de.torrc") + " LIB=" + filepath.Join(layout.BundleDir, "tor")
	if len(starts) != 1 || starts[0] != want {
		t.Fatalf("tor was started as %v, want exactly [%q]", starts, want)
	}
	if !s.Running("tor-de") || s.Running("tor-nl") {
		t.Fatalf("Running: de=%v nl=%v, want true and false", s.Running("tor-de"), s.Running("tor-nl"))
	}
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if pct, ok := progress.Get("de"); ok && pct == 45 {
			return
		}
	}
	t.Fatal("the bootstrap line Tor printed never reached Progress")
}

func TestSupervisorReportsTorMissing(t *testing.T) {
	layout := Layout{Root: t.TempDir(), BundleDir: t.TempDir()}
	s := NewSupervisor(layout, func() []string { return nil }, &Progress{})
	err := s.Reconcile(ExitRows([]model.TorExit{{Country: "de", Port: 62700, Enable: true}}))
	if err != ErrNotInstalled {
		t.Fatalf("Reconcile = %v, want ErrNotInstalled", err)
	}
}
