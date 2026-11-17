package service

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/tor"
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
