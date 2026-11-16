package backpack

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func releaseArchive(t *testing.T, binary string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range map[string]string{"README.md": "docs", "backpack": binary} {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body))}); err != nil {
			t.Fatalf("tar header: %v", err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatalf("tar body: %v", err)
		}
	}
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

func releaseServer(t *testing.T, archive []byte, sums string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/AminMGMT/BackPack/releases/download/v2.0.0/backpack_linux_amd64.tar.gz":
			_, _ = w.Write(archive)
		case "/AminMGMT/BackPack/releases/download/v2.0.0/SHA256SUMS":
			_, _ = w.Write([]byte(sums))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func sumLine(archive []byte) string {
	sum := sha256.Sum256(archive)
	return hex.EncodeToString(sum[:]) + "  backpack_linux_amd64.tar.gz\n"
}

func TestInstallPutsTheVerifiedEngineInPlace(t *testing.T) {
	archive := releaseArchive(t, "#!/bin/sh\necho engine\n")
	base := releaseServer(t, archive, "0000  other_file.tar.gz\n"+sumLine(archive))
	dest := filepath.Join(t.TempDir(), "bin", "backpack")

	if err := Install(context.Background(), http.DefaultClient, base, "v2.0.0", "amd64", dest); err != nil {
		t.Fatalf("Install: %v", err)
	}
	got, err := os.ReadFile(dest)
	if err != nil || string(got) != "#!/bin/sh\necho engine\n" {
		t.Fatalf("installed file = %q (%v), want the archive's backpack member", got, err)
	}
	if info, _ := os.Stat(dest); info.Mode().Perm()&0o100 == 0 {
		t.Fatalf("installed file mode = %v, want it executable", info.Mode().Perm())
	}
}

func TestInstallRefusesAnArchiveThatDoesNotMatchItsPublishedChecksum(t *testing.T) {
	archive := releaseArchive(t, "tampered")
	cases := []struct{ name, sums, wantErr string }{
		{"wrong checksum", strings.Repeat("0", 64) + "  backpack_linux_amd64.tar.gz\n", "checksum"},
		{"no checksum published for the file", strings.Repeat("0", 64) + "  backpack_linux_arm64.tar.gz\n", "checksum"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dest := filepath.Join(t.TempDir(), "backpack")
			base := releaseServer(t, archive, c.sums)
			err := Install(context.Background(), http.DefaultClient, base, "v2.0.0", "amd64", dest)
			if err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("Install error = %v, want one mentioning the checksum", err)
			}
			if _, statErr := os.Stat(dest); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatal("an unverified engine was left in place")
			}
		})
	}
}

func TestInstallKeepsTheCurrentEngineWhenTheDownloadFails(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "backpack")
	writeFile(t, dest, "current engine")
	base := releaseServer(t, nil, "")

	err := Install(context.Background(), http.DefaultClient, base, "v9.9.9", "amd64", dest)
	if err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("Install error = %v, want one naming status 404", err)
	}
	if got, _ := os.ReadFile(dest); string(got) != "current engine" {
		t.Fatalf("the working engine was replaced by a failed download: %q", got)
	}
}

func TestInstallRemembersWhichReleaseItPutInPlaceAndAFailureKeepsTheOldRecord(t *testing.T) {
	archive := releaseArchive(t, "#!/bin/sh\necho engine\n")
	base := releaseServer(t, archive, sumLine(archive))
	dest := filepath.Join(t.TempDir(), "bin", "backpack")

	if got := InstalledVersion(dest); got != "" {
		t.Fatalf("InstalledVersion before any install = %q, want none", got)
	}
	if err := Install(context.Background(), http.DefaultClient, base, "v2.0.0", "amd64", dest); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if got := InstalledVersion(dest); got != "v2.0.0" {
		t.Fatalf("InstalledVersion = %q, want v2.0.0", got)
	}

	if err := Install(context.Background(), http.DefaultClient, base, "v9.9.9", "amd64", dest); err == nil {
		t.Fatal("installing a release the server does not have succeeded")
	}
	if got := InstalledVersion(dest); got != "v2.0.0" {
		t.Fatalf("InstalledVersion after a failed update = %q, want v2.0.0 kept", got)
	}
}

func TestAnUpdateKeepsTheEngineItReplacedAndRollbackPutsItBack(t *testing.T) {
	old := releaseArchive(t, "old engine")
	oldBase := releaseServer(t, old, sumLine(old))
	dest := filepath.Join(t.TempDir(), "backpack")
	if err := Install(context.Background(), http.DefaultClient, oldBase, "v2.0.0", "amd64", dest); err != nil {
		t.Fatalf("first Install: %v", err)
	}

	next := releaseArchive(t, "new engine")
	rel := Release{Archive: next, Sums: []byte(sumLine(next))}
	if err := Place(rel, "v2.1.0", "amd64", dest); err != nil {
		t.Fatalf("Place: %v", err)
	}
	if got, _ := os.ReadFile(dest); string(got) != "new engine" || InstalledVersion(dest) != "v2.1.0" {
		t.Fatalf("after the update engine = %q version = %q", got, InstalledVersion(dest))
	}

	if err := Rollback(dest); err != nil {
		t.Fatalf("Rollback: %v", err)
	}
	if got, _ := os.ReadFile(dest); string(got) != "old engine" || InstalledVersion(dest) != "v2.0.0" {
		t.Fatalf("after the rollback engine = %q version = %q, want the old one back", got, InstalledVersion(dest))
	}
	if err := Rollback(dest); err == nil {
		t.Fatal("a second Rollback succeeded with nothing left to go back to")
	}
}

func TestPlaceRefusesAReleaseWhoseChecksumDoesNotMatch(t *testing.T) {
	archive := releaseArchive(t, "tampered")
	dest := filepath.Join(t.TempDir(), "backpack")
	writeFile(t, dest, "current engine")

	err := Place(Release{Archive: archive, Sums: []byte("00  backpack_linux_amd64.tar.gz\n")}, "v2.0.0", "amd64", dest)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("Place error = %v, want a checksum mismatch", err)
	}
	if got, _ := os.ReadFile(dest); string(got) != "current engine" {
		t.Fatalf("a refused release replaced the engine: %q", got)
	}
}

func TestLatestVersionReadsTheTagOfTheNewestRelease(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/AminMGMT/BackPack/releases/latest" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"tag_name":"v1.9.0","draft":false}`))
	}))
	t.Cleanup(srv.Close)

	got, err := LatestVersion(context.Background(), http.DefaultClient, srv.URL)
	if err != nil || got != "v1.9.0" {
		t.Fatalf("LatestVersion = %q, %v; want v1.9.0", got, err)
	}
}
