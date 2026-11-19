package tor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/tunnel"
)

// fakeTor installs a stand-in tor in a layout; body is the shell it runs after recording its pid.
func fakeTor(t *testing.T, body string) Layout {
	t.Helper()
	layout := Layout{Root: t.TempDir(), BundleDir: t.TempDir()}
	path := layout.Binary()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	pidFile := filepath.Join(t.TempDir(), "pids")
	t.Setenv("FAKE_PID_FILE", pidFile)
	script := "#!/bin/sh\necho $$ >> \"$FAKE_PID_FILE\"\n" + body + "\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return layout
}

func pids(t *testing.T) []int {
	t.Helper()
	raw, _ := os.ReadFile(os.Getenv("FAKE_PID_FILE"))
	var out []int
	for _, field := range strings.Fields(string(raw)) {
		n, err := strconv.Atoi(field)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, n)
	}
	return out
}

func waitPids(t *testing.T, want int) []int {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		if got := pids(t); len(got) >= want {
			return got
		}
	}
	t.Fatalf("the stand-in tor started %d time(s), want %d", len(pids(t)), want)
	return nil
}

func assertGone(t *testing.T, tmp string) {
	t.Helper()
	for _, pid := range pids(t) {
		deadline := time.Now().Add(5 * time.Second)
		for syscall.Kill(pid, 0) == nil {
			if time.Now().After(deadline) {
				t.Fatalf("tor process %d is still running", pid)
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	entries, _ := os.ReadDir(tmp)
	if len(entries) != 0 {
		t.Fatalf("the trial left %d entries in its temporary folder", len(entries))
	}
}

func connected(addr, url string, timeout time.Duration) tunnel.Status {
	return tunnel.Status{Connected: true, LatencyMs: 1200, ExitIP: "203.0.113.9", ExitLoc: "NL"}
}

func TestTrialSucceedsAndCleansUp(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	layout := fakeTor(t, "echo 'Bootstrapped 100% (done): Done'\nexec sleep 600")
	var probed string
	probe := func(addr, url string, timeout time.Duration) tunnel.Status {
		probed = addr
		return connected(addr, url, timeout)
	}

	got, err := Trial(context.Background(), "nl", nil, layout, probe)
	if err != nil {
		t.Fatal(err)
	}
	if !got.OK || got.LatencyMs != 1200 || got.ExitLoc != "NL" || got.ExitIP != "203.0.113.9" {
		t.Fatalf("result = %+v", got)
	}
	if !strings.HasPrefix(probed, "127.0.0.1:") {
		t.Fatalf("probed %q, want a loopback address", probed)
	}
	assertGone(t, tmp)
}

func TestTrialSaysWhenACountryHasNoExitRelays(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	layout := fakeTor(t, "echo '[warn] Failed to find node for hop #1 of our path. Discarding this circuit.'\necho 'Bootstrapped 100% (done): Done'\nexec sleep 600")
	probe := func(string, string, time.Duration) tunnel.Status {
		return tunnel.Status{Error: "socks connect: general failure"}
	}

	got, err := Trial(context.Background(), "aq", nil, layout, probe)
	if err != nil {
		t.Fatal(err)
	}
	if got.OK || !strings.Contains(got.Reason, "no exit relays") || got.Error != "socks connect: general failure" {
		t.Fatalf("result = %+v, want a failure whose reason names missing exit relays", got)
	}
	assertGone(t, tmp)
}

func TestTrialTimesOutWhenTorNeverFinishesConnecting(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	layout := fakeTor(t, "echo 'Bootstrapped 5% (conn): Connecting to a relay'\nexec sleep 600")
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	got, err := Trial(ctx, "de", nil, layout, connected)
	if err != nil {
		t.Fatal(err)
	}
	if got.OK || !strings.Contains(got.Reason, "did not finish connecting") || !strings.Contains(got.Reason, "5%") {
		t.Fatalf("result = %+v, want a timeout reason that says how far Tor got", got)
	}
	assertGone(t, tmp)
}

func TestTrialReportsTorStoppingByItself(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	layout := fakeTor(t, "echo '[err] Reading config failed--see warnings above.'\nexit 1")

	got, err := Trial(context.Background(), "de", nil, layout, connected)
	if err != nil {
		t.Fatal(err)
	}
	if got.OK || !strings.Contains(got.Reason, "stopped") || !strings.Contains(got.Reason, "Reading config failed") {
		t.Fatalf("result = %+v, want a reason carrying what Tor said", got)
	}
	assertGone(t, tmp)
}

func TestTrialReasonAlwaysCarriesTheLastWordsOfATorThatStopsAtOnce(t *testing.T) {
	layout := fakeTor(t, "echo '[err] Reading config failed--see warnings above.'\nexit 1")
	for i := 0; i < 300; i++ {
		got, err := Trial(context.Background(), "de", nil, layout, connected)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(got.Reason, "Reading config failed") {
			t.Fatalf("run %d: reason %q lost what Tor said", i, got.Reason)
		}
	}
}

func TestSecondTrialIsBusyAndLeavesNoStrayProcess(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	layout := fakeTor(t, "exec sleep 600")
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = Trial(ctx, "de", nil, layout, connected)
	}()
	waitPids(t, 1)

	if _, err := Trial(context.Background(), "nl", nil, layout, connected); !errors.Is(err, ErrTrialBusy) {
		t.Fatalf("second Trial = %v, want ErrTrialBusy", err)
	}
	if got := pids(t); len(got) != 1 {
		t.Fatalf("the refused trial started a process: %v", got)
	}
	cancel()
	wg.Wait()
	assertGone(t, tmp)
	if _, err := Trial(context.Background(), "nl", nil, fakeTor(t, "echo 'Bootstrapped 100% (done): Done'\nexec sleep 600"), connected); err != nil {
		t.Fatalf("a trial after the first finished: %v", err)
	}
}

func TestTrialRefusesWhatItCannotRun(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	missing := Layout{Root: t.TempDir(), BundleDir: t.TempDir()}
	if _, err := Trial(context.Background(), "de", nil, missing, connected); !errors.Is(err, ErrNotInstalled) {
		t.Fatalf("without a bundle: %v, want ErrNotInstalled", err)
	}
	layout := fakeTor(t, "exec sleep 600")
	if _, err := Trial(context.Background(), "d1", nil, layout, connected); err == nil {
		t.Fatal("a bad country was accepted")
	}
	if entries, _ := os.ReadDir(tmp); len(entries) != 0 {
		t.Fatalf("a refused trial left %d entries behind", len(entries))
	}
	if got := pids(t); len(got) != 0 {
		t.Fatalf("a refused trial started tor: %v", got)
	}
}
