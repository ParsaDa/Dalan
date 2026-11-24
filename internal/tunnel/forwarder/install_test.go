package forwarder

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/engine"
)

func tarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var raw bytes.Buffer
	gz := gzip.NewWriter(&raw)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return raw.Bytes()
}

func zipOf(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var raw bytes.Buffer
	zw := zip.NewWriter(&raw)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return raw.Bytes()
}

func sumOf(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

func testEngine(asset Asset, pins map[string]string) Engine {
	return Engine{
		ID: "demo", Label: "Demo", Repo: "o/demo", Pinned: "v1.0.0", Pins: pins,
		asset: func(string, string) (Asset, error) { return asset, nil },
	}
}

func TestPlaceExtractsTheBinaryFromATarballAndKeepsTheRestOut(t *testing.T) {
	archive := tarGz(t, map[string]string{"demo-1/README.md": "docs", "demo-1/demo": "ELF-binary", "demo-1/lib.so": "lib"})
	e := testEngine(Asset{Name: "demo.tar.gz", Member: "demo"}, map[string]string{"demo.tar.gz": sumOf(archive)})
	dest := filepath.Join(t.TempDir(), "bin", "demo")
	if err := e.Place(engine.Release{Version: "v1.0.0", Archive: archive}, "v1.0.0", "amd64", dest); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dest)
	if err != nil || string(got) != "ELF-binary" {
		t.Fatalf("installed file = %q, %v", got, err)
	}
	if v := e.InstalledVersion(dest); v != "v1.0.0" {
		t.Fatalf("installed version = %q", v)
	}
}

func TestPlaceReadsAZipAndARawBinary(t *testing.T) {
	zipped := zipOf(t, map[string]string{"x/demo": "from-zip", "geoip.dat": "data"})
	e := testEngine(Asset{Name: "demo.zip", Member: "demo"}, map[string]string{"demo.zip": sumOf(zipped)})
	dest := filepath.Join(t.TempDir(), "demo")
	if err := e.Place(engine.Release{Version: "v1.0.0", Archive: zipped}, "v1.0.0", "amd64", dest); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(dest); string(got) != "from-zip" {
		t.Fatalf("zip member = %q", got)
	}
	raw := []byte("bare-binary")
	e = testEngine(Asset{Name: "Launcher"}, map[string]string{"Launcher": sumOf(raw)})
	dest = filepath.Join(t.TempDir(), "launcher")
	if err := e.Place(engine.Release{Version: "v1.0.0", Archive: raw}, "v1.0.0", "amd64", dest); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(dest); string(got) != "bare-binary" {
		t.Fatalf("raw asset = %q", got)
	}
}

func TestPlaceRefusesAFileThatDiffersFromThePin(t *testing.T) {
	archive := tarGz(t, map[string]string{"demo": "tampered"})
	e := testEngine(Asset{Name: "demo.tar.gz", Member: "demo"}, map[string]string{"demo.tar.gz": sumOf([]byte("the real one"))})
	dest := filepath.Join(t.TempDir(), "demo")
	err := e.Place(engine.Release{Version: "v1.0.0", Archive: archive}, "v1.0.0", "amd64", dest)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("err = %v, want a checksum mismatch", err)
	}
	if _, statErr := os.Stat(dest); statErr == nil {
		t.Fatal("a file that failed the pin was installed")
	}
}

func TestPlaceTakesAnotherVersionOnlyOnItsPublishedChecksum(t *testing.T) {
	archive := tarGz(t, map[string]string{"demo": "newer"})
	e := testEngine(Asset{Name: "demo.tar.gz", Member: "demo"}, map[string]string{"demo.tar.gz": sumOf([]byte("pinned"))})
	dest := filepath.Join(t.TempDir(), "demo")
	if err := e.Place(engine.Release{Version: "v1.1.0", Archive: archive}, "v1.1.0", "amd64", dest); err == nil ||
		!strings.Contains(err.Error(), "no checksum is published") {
		t.Fatalf("err = %v, want a refusal without a published checksum", err)
	}
	sums := []byte(sumOf(archive) + "  demo.tar.gz\n")
	if err := e.Place(engine.Release{Version: "v1.1.0", Archive: archive, Sums: sums}, "v1.1.0", "amd64", dest); err != nil {
		t.Fatalf("published checksum was refused: %v", err)
	}
}

func TestPlaceRefusesAnArchiveWithTwoBinariesOfThatName(t *testing.T) {
	archive := tarGz(t, map[string]string{"a/demo": "one", "b/demo": "two"})
	e := testEngine(Asset{Name: "demo.tar.gz", Member: "demo"}, map[string]string{"demo.tar.gz": sumOf(archive)})
	err := e.Place(engine.Release{Version: "v1.0.0", Archive: archive}, "v1.0.0", "amd64", filepath.Join(t.TempDir(), "demo"))
	if err == nil || !strings.Contains(err.Error(), "exactly one") {
		t.Fatalf("err = %v, want a refusal of an ambiguous archive", err)
	}
}

func TestFetchReadsTheAssetAndFindsPublishedChecksums(t *testing.T) {
	archive := []byte("archive-bytes")
	var asked []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = append(asked, r.URL.Path)
		switch r.URL.Path {
		case "/o/demo/releases/download/v1.0.0/demo.tar.gz":
			_, _ = w.Write(archive)
		case "/o/demo/releases/download/v1.0.0/checksums.txt":
			_, _ = w.Write([]byte(sumOf(archive) + "  demo.tar.gz\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	e := testEngine(Asset{Name: "demo.tar.gz", Member: "demo"}, nil)
	rel, err := e.Fetch(context.Background(), srv.Client(), srv.URL, "v1.0.0", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	if string(rel.Archive) != "archive-bytes" || !strings.Contains(string(rel.Sums), "demo.tar.gz") {
		t.Fatalf("release = %q / %q (asked %v)", rel.Archive, rel.Sums, asked)
	}
}
