package rakhsh

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/engine"
)

const (
	// PinnedVersion is the Rakhsh release Dalan installs; it moves only with a Dalan release.
	PinnedVersion = "v0.1.0"
	// ReleaseBaseURL is where Rakhsh publishes its releases.
	ReleaseBaseURL = "https://github.com"
	// APIBaseURL is the GitHub API that names the newest release.
	APIBaseURL = "https://api.github.com"

	releaseRepo = "CluvexStudio/rakhsh"
)

var errUnsupportedArch = errors.New("Rakhsh publishes no Linux build for this processor")

// Release, Chunk and ChunkSize are the engine package's, so a node receives every engine the same way.
type (
	Release = engine.Release
	Chunk   = engine.Chunk
)

const ChunkSize = engine.ChunkSize

// assetName is the release file for a Go architecture: a bare static binary named for its Rust target.
func assetName(arch string) (string, error) {
	targets := map[string]string{
		"amd64":   "x86_64-unknown-linux-musl",
		"arm64":   "aarch64-unknown-linux-musl",
		"arm":     "armv7-unknown-linux-musleabihf",
		"386":     "i686-unknown-linux-musl",
		"riscv64": "riscv64gc-unknown-linux-musl",
	}
	target, ok := targets[arch]
	if !ok {
		return "", fmt.Errorf("%w (%s)", errUnsupportedArch, arch)
	}
	return "rakhsh-" + target, nil
}

// Fetch downloads a Rakhsh release and its checksum without installing anything; the binary rides in Archive.
func Fetch(ctx context.Context, client *http.Client, baseURL, version, arch string) (Release, error) {
	asset, err := assetName(arch)
	if err != nil {
		return Release{}, err
	}
	url := fmt.Sprintf("%s/%s/releases/download/%s/%s", baseURL, releaseRepo, version, asset)
	binary, err := engine.Download(ctx, client, url)
	if err != nil {
		return Release{}, err
	}
	sums, err := engine.Download(ctx, client, url+".sha256")
	if err != nil {
		return Release{}, err
	}
	return Release{Version: version, Archive: binary, Sums: sums}, nil
}

// LatestVersion is the tag of the newest Rakhsh release, read from the GitHub API at apiURL.
func LatestVersion(ctx context.Context, client *http.Client, apiURL string) (string, error) {
	return engine.LatestTag(ctx, client, apiURL, releaseRepo, "Rakhsh")
}

// Install downloads a Rakhsh release, checks it against its published checksum, and only then replaces dest.
func Install(ctx context.Context, client *http.Client, baseURL, version, arch, dest string) error {
	rel, err := Fetch(ctx, client, baseURL, version, arch)
	if err != nil {
		return err
	}
	return Place(rel, version, arch, dest)
}

// Place checks rel against its published checksum and replaces the engine at dest with it,
// keeping the engine it replaces so Rollback can bring it back.
func Place(rel Release, version, arch, dest string) error {
	asset, err := assetName(arch)
	if err != nil {
		return err
	}
	if err := engine.CheckChecksum(rel.Sums, asset, rel.Archive); err != nil {
		return err
	}
	return engine.Replace(dest, rel.Archive, version)
}

// Rollback puts back the engine the last Place replaced.
func Rollback(dest string) error { return engine.Rollback(dest, "Rakhsh") }

// InstalledVersion is the release Install last put at dest; empty when the engine was placed by hand.
func InstalledVersion(dest string) string { return engine.InstalledVersion(dest) }
