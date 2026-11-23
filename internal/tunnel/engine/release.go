package engine

import (
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
	"regexp"
	"strings"
)

const maxDownload = 200 << 20

// Release is a downloaded engine release with the checksums published beside it.
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

// Download reads one release file, refusing anything but a 200 answer.
func Download(ctx context.Context, client *http.Client, url string) ([]byte, error) {
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
	return io.ReadAll(io.LimitReader(resp.Body, maxDownload))
}

// LatestTag is the tag of the newest release of repo, read from the GitHub API at apiURL.
func LatestTag(ctx context.Context, client *http.Client, apiURL, repo, label string) (string, error) {
	raw, err := Download(ctx, client, fmt.Sprintf("%s/repos/%s/releases/latest", apiURL, repo))
	if err != nil {
		return "", err
	}
	var latest struct {
		Tag string `json:"tag_name"`
	}
	if err := json.Unmarshal(raw, &latest); err != nil || latest.Tag == "" {
		return "", fmt.Errorf("the release list names no latest %s release", label)
	}
	return latest.Tag, nil
}

// VersionPattern is the tag a release must have to be installable; the tag becomes part of a download URL.
var VersionPattern = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)

// releaseListPage is how many releases one list call reads, enough to find a few plain ones past
// pre-releases and drafts.
const releaseListPage = 30

// RecentTags lists up to limit tags of repo, newest first, that name a plain vX.Y.Z release (no drafts,
// pre-releases or other tags), read from the GitHub API at apiURL.
func RecentTags(ctx context.Context, client *http.Client, apiURL, repo, label string, limit int) ([]string, error) {
	raw, err := Download(ctx, client, fmt.Sprintf("%s/repos/%s/releases?per_page=%d", apiURL, repo, releaseListPage))
	if err != nil {
		return nil, err
	}
	var releases []struct {
		Tag        string `json:"tag_name"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
	}
	if err := json.Unmarshal(raw, &releases); err != nil {
		return nil, fmt.Errorf("the %s release list could not be read: %w", label, err)
	}
	var tags []string
	for _, r := range releases {
		if r.Draft || r.Prerelease || !VersionPattern.MatchString(r.Tag) {
			continue
		}
		tags = append(tags, r.Tag)
		if len(tags) == limit {
			break
		}
	}
	if len(tags) == 0 {
		return nil, fmt.Errorf("the release list names no %s release", label)
	}
	return tags, nil
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

// CheckChecksum refuses payload unless the published checksums list asset with payload's SHA-256.
func CheckChecksum(sums []byte, asset string, payload []byte) error {
	want, ok := publishedChecksum(sums, asset)
	if !ok {
		return fmt.Errorf("no checksum is published for %s; refusing to install it", asset)
	}
	sum := sha256.Sum256(payload)
	if got := hex.EncodeToString(sum[:]); got != want {
		return fmt.Errorf("checksum mismatch for %s: got %s, published %s", asset, got, want)
	}
	return nil
}

func previousFile(dest string) string { return dest + ".previous" }
func versionFile(dest string) string  { return dest + ".version" }

// Replace puts binary at dest and records version beside it, keeping the engine it replaces for Rollback.
func Replace(dest string, binary []byte, version string) error {
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

// Rollback puts back the engine the last Replace set aside; label names the engine in the error.
func Rollback(dest, label string) error {
	if _, err := os.Stat(previousFile(dest)); err != nil {
		return fmt.Errorf("there is no earlier %s engine to go back to", label)
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

// InstalledVersion is the release Replace last put at dest; empty when the engine was placed by hand
// or by an older Dalan.
func InstalledVersion(dest string) string {
	raw, err := os.ReadFile(versionFile(dest))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}
