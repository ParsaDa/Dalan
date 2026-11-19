package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/tor"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/backpack"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/engine"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/rakhsh"
)

// emptyEngineSlots points both external engines at binaries that are not installed yet.
func emptyEngineSlots(t *testing.T) {
	t.Helper()
	oldBackpack, oldRakhsh := tunnelEngines, rakhshEngines
	tunnelEngines = backpack.NewSupervisor(filepath.Join(t.TempDir(), "backpack"), t.TempDir())
	rakhshEngines = rakhsh.NewSupervisor(filepath.Join(t.TempDir(), "rakhsh"), t.TempDir())
	t.Cleanup(func() { tunnelEngines, rakhshEngines = oldBackpack, oldRakhsh })
}

func TestMissingEnginesAreInstalledAndOneFailureDoesNotStopTheRest(t *testing.T) {
	emptyEngineSlots(t)
	layout := useTorLayout(t)
	server := torBundleServer(t)
	oldBase := torReleaseBase
	torReleaseBase = server.URL
	t.Cleanup(func() { torReleaseBase = oldBase })

	rel := engineRelease(t, true)
	asked := map[string]int{}
	src := func(_ context.Context, id, version, _ string) (engine.Release, error) {
		asked[id]++
		if id == "rakhsh" {
			return engine.Release{}, errors.New("no route to the release server")
		}
		return rel, nil
	}
	got := installMissingEngines(context.Background(), src, func(ctx context.Context) (engine.Release, error) {
		return tor.Fetch(ctx, server.Client(), server.URL, tor.PinnedVersion, runtime.GOARCH)
	})

	if len(got) != 3 || got[0].Label != "BackPack" || got[1].Label != "Rakhsh" || got[2].Label != "Tor" {
		t.Fatalf("results = %+v, want BackPack, Rakhsh, Tor in that order", got)
	}
	if !got[0].Installed || got[0].Err != nil || got[0].Version != backpack.PinnedVersion {
		t.Fatalf("BackPack = %+v, want installed at %s", got[0], backpack.PinnedVersion)
	}
	if got[1].Installed || got[1].Err == nil {
		t.Fatalf("Rakhsh = %+v, want not installed with the reason", got[1])
	}
	if !got[2].Installed || got[2].Err != nil || tor.InstalledVersion(layout.BundleDir) != tor.PinnedVersion {
		t.Fatalf("Tor = %+v (bundle %q), want installed at %s", got[2], tor.InstalledVersion(layout.BundleDir), tor.PinnedVersion)
	}
	if _, err := os.Stat(tunnelEngines.Binary); err != nil {
		t.Fatalf("the BackPack binary is not in place: %v", err)
	}
	if _, err := os.Stat(rakhshEngines.Binary); err == nil {
		t.Fatal("a failed Rakhsh install left a binary behind")
	}

	asked = map[string]int{}
	again := installMissingEngines(context.Background(), src, func(context.Context) (engine.Release, error) {
		t.Fatal("Tor was downloaded again although it is installed")
		return engine.Release{}, nil
	})
	if asked["backpack"] != 0 || !again[0].Installed || again[0].Err != nil || !again[2].Installed {
		t.Fatalf("second run asked %v and returned %+v; installed engines must be left alone", asked, again)
	}
	if asked["rakhsh"] != 1 || again[1].Err == nil {
		t.Fatalf("the engine that failed was not tried again: asked %v, %+v", asked, again[1])
	}
}

func TestAnEngineThatHangsOnTheNetworkDoesNotHoldUpTheOthers(t *testing.T) {
	emptyEngineSlots(t)
	useTorLayout(t)
	old := preinstallTimeout
	preinstallTimeout = 100 * time.Millisecond
	t.Cleanup(func() { preinstallTimeout = old })

	hang := func(ctx context.Context, _, _, _ string) (engine.Release, error) {
		<-ctx.Done()
		return engine.Release{}, ctx.Err()
	}
	start := time.Now()
	got := installMissingEngines(context.Background(), hang, func(ctx context.Context) (engine.Release, error) {
		<-ctx.Done()
		return engine.Release{}, ctx.Err()
	})
	if took := time.Since(start); took > 3*time.Second {
		t.Fatalf("three hanging downloads took %v, want about 3 x the per-engine limit", took)
	}
	for _, r := range got {
		if r.Installed || !errors.Is(r.Err, context.DeadlineExceeded) {
			t.Fatalf("%s = %+v, want not installed because it ran out of time", r.Label, r)
		}
	}
}

func TestAnUnusableRouteLeavesEveryEngineNotInstalledWithTheReason(t *testing.T) {
	emptyEngineSlots(t)
	useTorLayout(t)
	got := InstallMissingEngines(context.Background(), EngineRoute{Via: "carrier-pigeon"})
	if len(got) != 3 {
		t.Fatalf("got %d results, want one for BackPack, Rakhsh and Tor", len(got))
	}
	for _, r := range got {
		if r.Installed || !errors.Is(r.Err, errUnknownRoute) {
			t.Errorf("%s = %+v, want not installed because of the route", r.Label, r)
		}
	}
}
