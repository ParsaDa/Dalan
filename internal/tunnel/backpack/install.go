package backpack

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"

	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/engine"
)

const (
	// PinnedVersion is the BackPack release Dalan installs; it moves only with a Dalan release.
	PinnedVersion = "v1.9.0"
	// ReleaseBaseURL is where BackPack publishes its releases.
	ReleaseBaseURL = "https://github.com"
	// APIBaseURL is the GitHub API that names the newest release.
	APIBaseURL = "https://api.github.com"

	releaseRepo    = "AminMGMT/BackPack"
	maxArchiveSize = 200 << 20
)

func extractEngine(archive []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil, errors.New("the release archive holds no backpack binary")
		}
		if err != nil {
			return nil, err
		}
		if filepath.Base(header.Name) == "backpack" && header.Typeflag == tar.TypeReg {
			return io.ReadAll(io.LimitReader(tr, maxArchiveSize))
		}
	}
}

// Release, Chunk and ChunkSize are shared with the other engines, which node updates carry the same way.
type (
	Release = engine.Release
	Chunk   = engine.Chunk
)

const ChunkSize = engine.ChunkSize

func assetName(arch string) string { return "backpack_linux_" + arch + ".tar.gz" }

// Fetch downloads a BackPack release and its checksums without installing anything.
func Fetch(ctx context.Context, client *http.Client, baseURL, version, arch string) (Release, error) {
	url := fmt.Sprintf("%s/%s/releases/download/%s/", baseURL, releaseRepo, version)
	archive, err := engine.Download(ctx, client, url+assetName(arch))
	if err != nil {
		return Release{}, err
	}
	sums, err := engine.Download(ctx, client, url+"SHA256SUMS")
	if err != nil {
		return Release{}, err
	}
	return Release{Version: version, Archive: archive, Sums: sums}, nil
}

// LatestVersion is the tag of the newest BackPack release, read from the GitHub API at apiURL.
func LatestVersion(ctx context.Context, client *http.Client, apiURL string) (string, error) {
	return engine.LatestTag(ctx, client, apiURL, releaseRepo, "BackPack")
}

// RecentVersions lists the newest BackPack releases, newest first, read from the GitHub API at apiURL.
func RecentVersions(ctx context.Context, client *http.Client, apiURL string, limit int) ([]string, error) {
	return engine.RecentTags(ctx, client, apiURL, releaseRepo, "BackPack", limit)
}

// Install downloads a BackPack release, checks it against the checksums published with it,
// and only then replaces the engine at dest.
func Install(ctx context.Context, client *http.Client, baseURL, version, arch, dest string) error {
	rel, err := Fetch(ctx, client, baseURL, version, arch)
	if err != nil {
		return err
	}
	return Place(rel, version, arch, dest)
}

// Place checks rel against its published checksums and replaces the engine at dest with it,
// keeping the engine it replaces so Rollback can bring it back.
func Place(rel Release, version, arch, dest string) error {
	if err := engine.CheckChecksum(rel.Sums, assetName(arch), rel.Archive); err != nil {
		return err
	}
	binary, err := extractEngine(rel.Archive)
	if err != nil {
		return err
	}
	return engine.Replace(dest, binary, version)
}

// Rollback puts back the engine the last Place replaced.
func Rollback(dest string) error { return engine.Rollback(dest, "BackPack") }

// InstalledVersion is the release Install last put at dest; empty when the engine was placed by hand
// or by an older Dalan.
func InstalledVersion(dest string) string { return engine.InstalledVersion(dest) }
