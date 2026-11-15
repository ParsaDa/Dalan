package backpack

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
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

// Install downloads a BackPack release, checks it against the checksums published with it,
// and only then replaces the engine at dest.
func Install(ctx context.Context, client *http.Client, baseURL, version, arch, dest string) error {
	asset := "backpack_linux_" + arch + ".tar.gz"
	release := fmt.Sprintf("%s/%s/releases/download/%s/", baseURL, releaseRepo, version)

	archive, err := fetch(ctx, client, release+asset)
	if err != nil {
		return err
	}
	sums, err := fetch(ctx, client, release+"SHA256SUMS")
	if err != nil {
		return err
	}
	want, ok := publishedChecksum(sums, asset)
	if !ok {
		return fmt.Errorf("no checksum is published for %s; refusing to install it", asset)
	}
	sum := sha256.Sum256(archive)
	if got := hex.EncodeToString(sum[:]); got != want {
		return fmt.Errorf("checksum mismatch for %s: got %s, published %s", asset, got, want)
	}
	binary, err := extractEngine(archive)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
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
	return nil
}
