package tor

import (
	"archive/tar"
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

type archiveEntry struct {
	name string
	body string
	mode int64
	link string
}

func buildArchive(t *testing.T, entries []archiveEntry) []byte {
	t.Helper()
	var raw bytes.Buffer
	gz := gzip.NewWriter(&raw)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		hdr := &tar.Header{Name: e.name, Mode: e.mode, Size: int64(len(e.body)), Typeflag: tar.TypeReg}
		if e.link != "" {
			hdr.Typeflag, hdr.Linkname, hdr.Size = tar.TypeSymlink, e.link, 0
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if e.link == "" {
			if _, err := tw.Write([]byte(e.body)); err != nil {
				t.Fatal(err)
			}
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

func goodEntries(marker string) []archiveEntry {
	return []archiveEntry{
		{name: "tor/tor", body: "#!/bin/sh\nexec sleep 600\n# " + marker, mode: 0o755},
		{name: "tor/libevent.so.7", body: "lib", mode: 0o644},
		{name: "tor/libevent.so", link: "libevent.so.7"},
		{name: "tor/pluggable_transports/lyrebird", body: "pt", mode: 0o755},
		{name: "data/geoip", body: "geo", mode: 0o644},
	}
}

func release(t *testing.T, version, arch string, entries []archiveEntry) engine.Release {
	t.Helper()
	archive := buildArchive(t, entries)
	asset, err := AssetName(version, arch)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(archive)
	sums := "0000  tor-browser-linux-x86_64-" + version + ".tar.xz\n" + hex.EncodeToString(sum[:]) + "  " + asset + "\n"
	return engine.Release{Version: version, Archive: archive, Sums: []byte(sums)}
}

func TestAssetName(t *testing.T) {
	cases := []struct {
		arch string
		want string
		fail bool
	}{
		{"amd64", "tor-expert-bundle-linux-x86_64-16.0.1.tar.gz", false},
		{"arm64", "tor-expert-bundle-linux-aarch64-16.0.1.tar.gz", false},
		{"386", "tor-expert-bundle-linux-i686-16.0.1.tar.gz", false},
		{"riscv64", "", true},
	}
	for _, c := range cases {
		t.Run(c.arch, func(t *testing.T) {
			got, err := AssetName("16.0.1", c.arch)
			if c.fail {
				if err == nil || !strings.Contains(err.Error(), "not available") {
					t.Fatalf("err = %v, want one saying the bundle is not available", err)
				}
				return
			}
			if err != nil || got != c.want {
				t.Fatalf("AssetName = %q, %v; want %q", got, err, c.want)
			}
		})
	}
}

func TestPlaceExtractsKeepsModesAndSymlinksAndSetAsidePrevious(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "tor-bundle")
	if err := Place(release(t, "16.0.1", "amd64", goodEntries("one")), "16.0.1", "amd64", dest); err != nil {
		t.Fatalf("first Place: %v", err)
	}
	info, err := os.Stat(filepath.Join(dest, "tor", "tor"))
	if err != nil || info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("tor/tor: %v, mode %v; want an executable file", err, info)
	}
	if target, err := os.Readlink(filepath.Join(dest, "tor", "libevent.so")); err != nil || target != "libevent.so.7" {
		t.Fatalf("symlink = %q, %v", target, err)
	}
	if got := InstalledVersion(dest); got != "16.0.1" {
		t.Fatalf("InstalledVersion = %q, want 16.0.1", got)
	}

	if err := Place(release(t, "16.0.2", "amd64", goodEntries("two")), "16.0.2", "amd64", dest); err != nil {
		t.Fatalf("second Place: %v", err)
	}
	if got := InstalledVersion(dest); got != "16.0.2" {
		t.Fatalf("InstalledVersion after update = %q, want 16.0.2", got)
	}
	if got := InstalledVersion(dest + ".previous"); got != "16.0.1" {
		t.Fatalf("the earlier bundle is %q, want 16.0.1", got)
	}
}

func TestPlaceRefusesAWrongChecksumAndKeepsTheInstalledBundle(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "tor-bundle")
	if err := Place(release(t, "16.0.1", "amd64", goodEntries("one")), "16.0.1", "amd64", dest); err != nil {
		t.Fatal(err)
	}
	tampered := release(t, "16.0.2", "amd64", goodEntries("two"))
	tampered.Archive = append(tampered.Archive, 'x')

	err := Place(tampered, "16.0.2", "amd64", dest)
	if err == nil || !strings.Contains(err.Error(), "tor-expert-bundle-linux-x86_64-16.0.2.tar.gz") {
		t.Fatalf("Place = %v, want a checksum error naming the asset", err)
	}
	if got := InstalledVersion(dest); got != "16.0.1" {
		t.Fatalf("a refused install left version %q, want 16.0.1", got)
	}
	if _, err := os.Stat(dest + ".previous"); err == nil {
		t.Fatal("a refused install set the working bundle aside")
	}
}

func TestPlaceRefusesAnArchiveThatWritesOutsideTheBundle(t *testing.T) {
	cases := map[string][]archiveEntry{
		"parent path":      {{name: "../evil", body: "x", mode: 0o644}},
		"absolute path":    {{name: "/tmp/evil-tor-test", body: "x", mode: 0o644}},
		"escaping symlink": {{name: "tor/link", link: "../../outside"}},
		"absolute symlink": {{name: "tor/link", link: "/etc/passwd"}},
	}
	for name, entries := range cases {
		t.Run(name, func(t *testing.T) {
			base := t.TempDir()
			dest := filepath.Join(base, "tor-bundle")
			if err := Place(release(t, "16.0.1", "amd64", goodEntries("one")), "16.0.1", "amd64", dest); err != nil {
				t.Fatal(err)
			}
			err := Place(release(t, "16.0.2", "amd64", append(goodEntries("two"), entries...)), "16.0.2", "amd64", dest)
			if err == nil {
				t.Fatal("Place accepted it")
			}
			if InstalledVersion(dest) != "16.0.1" {
				t.Fatalf("the installed bundle changed to %q", InstalledVersion(dest))
			}
			if _, statErr := os.Stat(filepath.Join(base, "evil")); statErr == nil {
				t.Fatal("a file was written outside the bundle")
			}
		})
	}
}

func TestRollbackPutsTheEarlierBundleBack(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "tor-bundle")
	if err := Rollback(dest); err == nil {
		t.Fatal("Rollback with nothing to go back to succeeded")
	}
	for _, v := range []string{"16.0.1", "16.0.2"} {
		if err := Place(release(t, v, "amd64", goodEntries(v)), v, "amd64", dest); err != nil {
			t.Fatal(err)
		}
	}
	if err := Rollback(dest); err != nil {
		t.Fatal(err)
	}
	if got := InstalledVersion(dest); got != "16.0.1" {
		t.Fatalf("after Rollback the bundle is %q, want 16.0.1", got)
	}
}

func TestFetchReadsTheArchiveAndItsChecksumFile(t *testing.T) {
	rel := release(t, "16.0.1", "amd64", goodEntries("one"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/torbrowser/16.0.1/tor-expert-bundle-linux-x86_64-16.0.1.tar.gz":
			_, _ = w.Write(rel.Archive)
		case "/torbrowser/16.0.1/sha256sums-signed-build.txt":
			_, _ = w.Write(rel.Sums)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	got, err := Fetch(context.Background(), server.Client(), server.URL, "16.0.1", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != "16.0.1" || !bytes.Equal(got.Archive, rel.Archive) || !bytes.Equal(got.Sums, rel.Sums) {
		t.Fatalf("Fetch returned version %q and the wrong bytes", got.Version)
	}
	if _, err := Fetch(context.Background(), server.Client(), server.URL, "9.9.9", "amd64"); err == nil {
		t.Fatal("Fetch of a release the server does not have succeeded")
	}
}
