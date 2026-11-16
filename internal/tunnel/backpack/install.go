package backpack

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	// PinnedVersion is the BackPack release Dalan installs; it moves only with a Dalan release.
	PinnedVersion = "v1.8.5"
	// ReleaseBaseURL is where BackPack publishes its releases.
	ReleaseBaseURL = "https://github.com"
	// APIBaseURL is the GitHub API that names the newest release.
	APIBaseURL = "https://api.github.com"

	releaseRepo    = "AminMGMT/BackPack"
	maxArchiveSize = 200 << 20
)

func fetch(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: status %d", url, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxArchiveSize))
}

func publishedChecksum(sums []byte, asset string) (string, bool) {
	for _, line := range strings.Split(string(sums), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == asset {
			return strings.ToLower(fields[0]), true
		}
	}
	return "", false
}

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

// Release is a downloaded BackPack archive with the checksums published beside it.
type Release struct {
	Version string `json:"version"`
	Archive []byte `json:"archive"`
	Sums    []byte `json:"sums"`
}

// ChunkSize keeps each call to a node small enough to finish within one request timeout on a poor link.
const ChunkSize = 512 << 10

// Chunk is one part of a release a node sends in pieces. Pending means the node is still downloading it.
type Chunk struct {
	Pending bool   `json:"pending"`
	Version string `json:"version"`
	Size    int    `json:"size"`
	Data    []byte `json:"data"`
	Sums    []byte `json:"sums"`
}

// ChunkAt returns the part of the archive that starts at offset; the checksums ride on the first part.
func (r Release) ChunkAt(offset int) Chunk {
	if offset < 0 || offset > len(r.Archive) {
		offset = len(r.Archive)
	}
	end := min(offset+ChunkSize, len(r.Archive))
	chunk := Chunk{Version: r.Version, Size: len(r.Archive), Data: r.Archive[offset:end]}
	if offset == 0 {
		chunk.Sums = r.Sums
	}
	return chunk
}

func assetName(arch string) string { return "backpack_linux_" + arch + ".tar.gz" }

// Fetch downloads a BackPack release and its checksums without installing anything.
func Fetch(ctx context.Context, client *http.Client, baseURL, version, arch string) (Release, error) {
	url := fmt.Sprintf("%s/%s/releases/download/%s/", baseURL, releaseRepo, version)
	archive, err := fetch(ctx, client, url+assetName(arch))
	if err != nil {
		return Release{}, err
	}
	sums, err := fetch(ctx, client, url+"SHA256SUMS")
	if err != nil {
		return Release{}, err
	}
	return Release{Version: version, Archive: archive, Sums: sums}, nil
}

// LatestVersion is the tag of the newest BackPack release, read from the GitHub API at apiURL.
func LatestVersion(ctx context.Context, client *http.Client, apiURL string) (string, error) {
	raw, err := fetch(ctx, client, fmt.Sprintf("%s/repos/%s/releases/latest", apiURL, releaseRepo))
	if err != nil {
		return "", err
	}
	var latest struct {
		Tag string `json:"tag_name"`
	}
	if err := json.Unmarshal(raw, &latest); err != nil || latest.Tag == "" {
		return "", errors.New("the release list names no latest BackPack release")
	}
	return latest.Tag, nil
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
	asset := assetName(arch)
	want, ok := publishedChecksum(rel.Sums, asset)
	if !ok {
		return fmt.Errorf("no checksum is published for %s; refusing to install it", asset)
	}
	sum := sha256.Sum256(rel.Archive)
	if got := hex.EncodeToString(sum[:]); got != want {
		return fmt.Errorf("checksum mismatch for %s: got %s, published %s", asset, got, want)
	}
	binary, err := extractEngine(rel.Archive)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	if err := keepPrevious(dest); err != nil {
		return err
	}
	// Written beside dest and renamed, so a running engine never sees a half-written file.
	tmp := dest + ".new"
	if err := os.WriteFile(tmp, binary, 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.WriteFile(versionFile(dest), []byte(version+"\n"), 0o644)
}

func previousFile(dest string) string { return dest + ".previous" }

func keepPrevious(dest string) error {
	if _, err := os.Stat(dest); err != nil {
		return nil
	}
	if err := os.Rename(dest, previousFile(dest)); err != nil {
		return err
	}
	if err := os.Rename(versionFile(dest), previousFile(versionFile(dest))); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// Rollback puts back the engine the last Place replaced.
func Rollback(dest string) error {
	if _, err := os.Stat(previousFile(dest)); err != nil {
		return errors.New("there is no earlier BackPack engine to go back to")
	}
	if err := os.Rename(previousFile(dest), dest); err != nil {
		return err
	}
	if err := os.Rename(previousFile(versionFile(dest)), versionFile(dest)); errors.Is(err, os.ErrNotExist) {
		return os.Remove(versionFile(dest))
	} else if err != nil {
		return err
	}
	return nil
}

func versionFile(dest string) string { return dest + ".version" }

// InstalledVersion is the release Install last put at dest; empty when the engine was placed by hand
// or by an older Dalan.
func InstalledVersion(dest string) string {
	raw, err := os.ReadFile(versionFile(dest))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}
