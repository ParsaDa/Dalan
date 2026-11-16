package rakhsh

import (
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

const amd64Asset = "rakhsh-x86_64-unknown-linux-musl"

func releaseServer(t *testing.T, binary []byte, sums string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/CluvexStudio/rakhsh/releases/download/v0.2.0/" + amd64Asset:
			_, _ = w.Write(binary)
		case "/CluvexStudio/rakhsh/releases/download/v0.2.0/" + amd64Asset + ".sha256":
			_, _ = w.Write([]byte(sums))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func sumLine(binary []byte) string {
	sum := sha256.Sum256(binary)
	return hex.EncodeToString(sum[:]) + "  " + amd64Asset + "\n"
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestInstallPutsTheVerifiedBinaryInPlaceAndRemembersItsRelease(t *testing.T) {
	binary := []byte("#!/bin/sh\necho rakhsh\n")
	base := releaseServer(t, binary, sumLine(binary))
	dest := filepath.Join(t.TempDir(), "bin", "rakhsh")

	if got := InstalledVersion(dest); got != "" {
		t.Fatalf("InstalledVersion before any install = %q, want none", got)
	}
	if err := Install(context.Background(), http.DefaultClient, base, "v0.2.0", "amd64", dest); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if got, err := os.ReadFile(dest); err != nil || string(got) != string(binary) {
		t.Fatalf("installed file = %q (%v), want the release asset itself", got, err)
	}
	if info, _ := os.Stat(dest); info.Mode().Perm()&0o100 == 0 {
		t.Fatalf("installed file mode = %v, want it executable", info.Mode().Perm())
	}
	if got := InstalledVersion(dest); got != "v0.2.0" {
		t.Fatalf("InstalledVersion = %q, want v0.2.0", got)
	}
}

func TestInstallRefusesABinaryThatDoesNotMatchItsPublishedChecksum(t *testing.T) {
	tampered := []byte("tampered")
	for _, c := range []struct{ name, sums string }{
		{"wrong checksum", strings.Repeat("0", 64) + "  " + amd64Asset + "\n"},
		{"checksum for another file", strings.Repeat("0", 64) + "  rakhsh-aarch64-unknown-linux-musl\n"},
		{"empty checksum file", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			dest := filepath.Join(t.TempDir(), "rakhsh")
			writeFile(t, dest, "current engine")
			base := releaseServer(t, tampered, c.sums)
			err := Install(context.Background(), http.DefaultClient, base, "v0.2.0", "amd64", dest)
			if err == nil || !strings.Contains(err.Error(), "checksum") {
				t.Fatalf("Install error = %v, want one mentioning the checksum", err)
			}
			if got, _ := os.ReadFile(dest); string(got) != "current engine" {
				t.Fatalf("an unverified binary replaced the working one: %q", got)
			}
		})
	}
}

func TestInstallKeepsTheCurrentEngineWhenTheDownloadFails(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "rakhsh")
	writeFile(t, dest, "current engine")
	base := releaseServer(t, nil, "")

	err := Install(context.Background(), http.DefaultClient, base, "v9.9.9", "amd64", dest)
	if err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("Install error = %v, want one naming status 404", err)
	}
	if got, _ := os.ReadFile(dest); string(got) != "current engine" {
		t.Fatalf("a failed download replaced the working engine: %q", got)
	}
}

func TestAnUpdateKeepsWhatItReplacedAndRollbackPutsItBack(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "rakhsh")
	oldBinary := []byte("old engine")
	if err := Place(Release{Archive: oldBinary, Sums: []byte(sumLine(oldBinary))}, "v0.1.0", "amd64", dest); err != nil {
		t.Fatalf("first Place: %v", err)
	}
	newBinary := []byte("new engine")
	if err := Place(Release{Archive: newBinary, Sums: []byte(sumLine(newBinary))}, "v0.2.0", "amd64", dest); err != nil {
		t.Fatalf("second Place: %v", err)
	}
	if got, _ := os.ReadFile(dest); string(got) != "new engine" || InstalledVersion(dest) != "v0.2.0" {
		t.Fatalf("after the update engine = %q version = %q", got, InstalledVersion(dest))
	}
	if err := Rollback(dest); err != nil {
		t.Fatalf("Rollback: %v", err)
	}
	if got, _ := os.ReadFile(dest); string(got) != "old engine" || InstalledVersion(dest) != "v0.1.0" {
		t.Fatalf("after the rollback engine = %q version = %q, want the old one", got, InstalledVersion(dest))
	}
	if err := Rollback(dest); err == nil || !strings.Contains(err.Error(), "Rakhsh") {
		t.Fatalf("a second Rollback = %v, want an error that names Rakhsh", err)
	}
}

func TestAssetNamesFollowRakhshsReleaseTargets(t *testing.T) {
	for goarch, want := range map[string]string{
		"amd64":   "rakhsh-x86_64-unknown-linux-musl",
		"arm64":   "rakhsh-aarch64-unknown-linux-musl",
		"arm":     "rakhsh-armv7-unknown-linux-musleabihf",
		"386":     "rakhsh-i686-unknown-linux-musl",
		"riscv64": "rakhsh-riscv64gc-unknown-linux-musl",
	} {
		if got, err := assetName(goarch); err != nil || got != want {
			t.Errorf("assetName(%s) = %q, %v; want %q", goarch, got, err, want)
		}
	}
	if _, err := assetName("mips"); err == nil || !strings.Contains(err.Error(), "mips") {
		t.Errorf("assetName(mips) error = %v, want one naming mips", err)
	}
}

func TestLatestVersionReadsTheTagOfTheNewestRelease(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/CluvexStudio/rakhsh/releases/latest" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"tag_name":"v0.2.0"}`))
	}))
	t.Cleanup(srv.Close)
	if got, err := LatestVersion(context.Background(), http.DefaultClient, srv.URL); err != nil || got != "v0.2.0" {
		t.Fatalf("LatestVersion = %q, %v; want v0.2.0", got, err)
	}
}

func TestInstallErrorsForAnArchitectureRakhshDoesNotBuild(t *testing.T) {
	err := Install(context.Background(), http.DefaultClient, "http://unused.invalid", "v0.2.0", "mips", filepath.Join(t.TempDir(), "rakhsh"))
	if err == nil || !errors.Is(err, errUnsupportedArch) {
		t.Fatalf("error = %v, want errUnsupportedArch", err)
	}
}
