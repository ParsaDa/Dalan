package service

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/tor"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel"
)

// useTorLayout points the service at a temporary bundle and state folder and stops any Tor it started.
func useTorLayout(t *testing.T) tor.Layout {
	t.Helper()
	layout := tor.Layout{Root: filepath.Join(t.TempDir(), "state"), BundleDir: filepath.Join(t.TempDir(), "bundle")}
	oldLayout, oldExits := torLayout, torExits
	torLayout, torExits = layout, newTorExits(layout)
	t.Cleanup(func() {
		torExits.StopAll()
		torLayout, torExits = oldLayout, oldExits
	})
	return layout
}

func torBundleServer(t *testing.T) *httptest.Server {
	t.Helper()
	var raw bytes.Buffer
	gz := gzip.NewWriter(&raw)
	tw := tar.NewWriter(gz)
	body := "#!/bin/sh\necho 'Bootstrapped 100% (done): Done'\nexec sleep 600\n"
	if err := tw.WriteHeader(&tar.Header{Name: "tor/tor", Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	_, _ = tw.Write([]byte(body))
	_ = tw.Close()
	_ = gz.Close()
	asset, err := tor.AssetName(tor.PinnedVersion, runtime.GOARCH)
	if err != nil {
		t.Skip(err)
	}
	sum := sha256.Sum256(raw.Bytes())
	sums := hex.EncodeToString(sum[:]) + "  " + asset + "\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch filepath.Base(r.URL.Path) {
		case asset:
			_, _ = w.Write(raw.Bytes())
		case "sha256sums-signed-build.txt":
			_, _ = w.Write([]byte(sums))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if ok() {
			return
		}
	}
	t.Fatalf("timed out waiting for %s", what)
}

func TestEngineInfoBeforeAndAfterInstall(t *testing.T) {
	svc := newTorDB(t)
	useTorLayout(t)
	server := torBundleServer(t)
	oldBase := torReleaseBase
	torReleaseBase = server.URL
	t.Cleanup(func() { torReleaseBase = oldBase })

	info, err := svc.EngineInfo()
	if err != nil || info.Installed || info.Pinned != tor.PinnedVersion || info.Engine != "tor" {
		t.Fatalf("before install: %+v, %v", info, err)
	}
	if err := svc.InstallEngine(); err != nil {
		t.Fatalf("InstallEngine: %v", err)
	}
	info, err = svc.EngineInfo()
	if err != nil || !info.Installed || info.Version != tor.PinnedVersion {
		t.Fatalf("after install: %+v, %v", info, err)
	}
}

func TestInstallStartsTheEnabledExitsAndDisableAndDeleteStopThem(t *testing.T) {
	svc := newTorDB(t)
	layout := useTorLayout(t)
	server := torBundleServer(t)
	oldBase := torReleaseBase
	torReleaseBase = server.URL
	t.Cleanup(func() { torReleaseBase = oldBase })

	de, err := svc.Add("de")
	if err != nil {
		t.Fatal(err)
	}
	nl, err := svc.Add("nl")
	if err != nil {
		t.Fatal(err)
	}
	if torExits.Running("tor-de") {
		t.Fatal("a Tor ran before the bundle was installed")
	}
	if err := svc.InstallEngine(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "both exits to run", func() bool { return torExits.Running("tor-de") && torExits.Running("tor-nl") })
	waitFor(t, "progress to reach 100", func() bool { p, ok := torProgress.Get("de"); return ok && p == 100 })

	if err := svc.SetEnable(de.Id, false); err != nil {
		t.Fatal(err)
	}
	if torExits.Running("tor-de") || !torExits.Running("tor-nl") {
		t.Fatalf("after disabling de: de=%v nl=%v", torExits.Running("tor-de"), torExits.Running("tor-nl"))
	}
	if _, ok := torProgress.Get("de"); ok {
		t.Fatal("the stopped exit still reports progress")
	}

	if err := os.MkdirAll(layout.DataDir("nl"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := svc.Delete(nl.Id); err != nil {
		t.Fatal(err)
	}
	if torExits.Running("tor-nl") {
		t.Fatal("a deleted exit still runs")
	}
	if _, err := os.Stat(layout.DataDir("nl")); err == nil {
		t.Fatal("a deleted exit left its Tor data directory behind")
	}
	if _, ok := torProgress.Get("nl"); ok {
		t.Fatal("a deleted exit still reports progress")
	}
}

func TestReconcileWithoutTorInstalledReportsItAndDoesNotRun(t *testing.T) {
	svc := newTorDB(t)
	useTorLayout(t)
	if _, err := svc.Add("de"); err != nil {
		t.Fatal(err)
	}
	if err := svc.ReconcileExits(); err != tor.ErrNotInstalled {
		t.Fatalf("ReconcileExits = %v, want ErrNotInstalled", err)
	}
}

func TestEnabledExitsReachTheGeneratedXrayConfig(t *testing.T) {
	svc := newTorDB(t)
	useTorLayout(t)
	de, err := svc.Add("de")
	if err != nil {
		t.Fatal(err)
	}
	nl, err := svc.Add("nl")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.SetEnable(nl.Id, false); err != nil {
		t.Fatal(err)
	}
	cfg, err := (&XrayService{}).GetXrayConfig()
	if err != nil {
		t.Fatalf("GetXrayConfig: %v", err)
	}
	outbounds := string(cfg.OutboundConfigs)
	if !strings.Contains(outbounds, `"tor-de"`) || strings.Contains(outbounds, `"tor-nl"`) {
		t.Fatalf("outbounds = %s, want tor-de and no tor-nl", outbounds)
	}
	if !strings.Contains(outbounds, fmt.Sprintf(`"port":%d`, de.Port)) {
		t.Fatalf("outbounds = %s, want the exit's port %d", outbounds, de.Port)
	}
}

func TestOutboundsListsOnlyEnabledExits(t *testing.T) {
	svc := newTorDB(t)
	useTorLayout(t)
	if _, err := svc.Add("de"); err != nil {
		t.Fatal(err)
	}
	nl, err := svc.Add("nl")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.SetEnable(nl.Id, false); err != nil {
		t.Fatal(err)
	}
	got, err := svc.Outbounds()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Tag != "tor-de" || got[0].Country != "de" || !got[0].Enable {
		t.Fatalf("Outbounds = %+v, want only tor-de", got)
	}
}

func TestStatusAllJudgesEachExitAndCachesTheProbe(t *testing.T) {
	svc := newTorDB(t)
	useTorLayout(t)
	server := torBundleServer(t)
	oldBase := torReleaseBase
	torReleaseBase = server.URL
	t.Cleanup(func() { torReleaseBase = oldBase })

	now := time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC)
	probes := 0
	oldProbe, oldNow := torProbe, torNow
	torNow = func() time.Time { return now }
	torProbe = func(addr, url string, timeout time.Duration) tunnel.Status {
		probes++
		return tunnel.Status{Connected: true, LatencyMs: 900, ExitLoc: "DE"}
	}
	t.Cleanup(func() { torProbe, torNow = oldProbe, oldNow })

	if _, err := svc.Add("de"); err != nil {
		t.Fatal(err)
	}
	nl, err := svc.Add("nl")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.SetEnable(nl.Id, false); err != nil {
		t.Fatal(err)
	}
	before, err := svc.StatusAll()
	if err != nil || before["de"].State != tor.StateNotInstalled {
		t.Fatalf("before install: %+v, %v", before["de"], err)
	}

	if err := svc.InstallEngine(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "de to bootstrap", func() bool { p, ok := torProgress.Get("de"); return ok && p == 100 })

	first, err := svc.StatusAll()
	if err != nil {
		t.Fatal(err)
	}
	if first["de"].State != tor.StateConnected || first["de"].LatencyMs != 900 || first["nl"].State != tor.StateStopped {
		t.Fatalf("statuses = %+v", first)
	}
	if _, err := svc.StatusAll(); err != nil || probes != 1 {
		t.Fatalf("a second call inside 30 s probed again: probes = %d, err = %v", probes, err)
	}
	now = now.Add(31 * time.Second)
	if _, err := svc.StatusAll(); err != nil || probes != 2 {
		t.Fatalf("a call after 31 s did not probe again: probes = %d, err = %v", probes, err)
	}
}
