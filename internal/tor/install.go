package tor

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/engine"
)

const (
	// PinnedVersion is the Tor Expert Bundle release Dalan installs; it moves only with a Dalan release.
	PinnedVersion = "16.0.1"
	// ReleaseBaseURL is where the Tor Project publishes the bundle.
	ReleaseBaseURL = "https://dist.torproject.org"

	checksumFile  = "sha256sums-signed-build.txt"
	versionFile   = ".dalan-version"
	extractLimit  = 600 << 20
	extractedMode = 0o755
)

var errUnsupportedArch = errors.New("the Tor bundle is not available for this processor")

// AssetName is the bundle archive of a release for a Go architecture.
func AssetName(version, goarch string) (string, error) {
	arch, ok := map[string]string{"amd64": "x86_64", "arm64": "aarch64", "386": "i686"}[goarch]
	if !ok {
		return "", fmt.Errorf("%w (%s)", errUnsupportedArch, goarch)
	}
	return fmt.Sprintf("tor-expert-bundle-linux-%s-%s.tar.gz", arch, version), nil
}

// Fetch downloads a bundle release and the checksum file published beside it, without installing anything.
func Fetch(ctx context.Context, client *http.Client, baseURL, version, goarch string) (engine.Release, error) {
	asset, err := AssetName(version, goarch)
	if err != nil {
		return engine.Release{}, err
	}
	dir := fmt.Sprintf("%s/torbrowser/%s/", baseURL, version)
	archive, err := engine.Download(ctx, client, dir+asset)
	if err != nil {
		return engine.Release{}, err
	}
	sums, err := engine.Download(ctx, client, dir+checksumFile)
	if err != nil {
		return engine.Release{}, err
	}
	return engine.Release{Version: version, Archive: archive, Sums: sums}, nil
}

func previousDir(dest string) string { return dest + ".previous" }

// Place checks rel against its published checksum, unpacks it beside dest and swaps it in, keeping the
// bundle it replaces for Rollback. Nothing at dest changes unless the whole archive was accepted.
func Place(rel engine.Release, version, goarch, dest string) error {
	asset, err := AssetName(version, goarch)
	if err != nil {
		return err
	}
	if err := engine.CheckChecksum(rel.Sums, asset, rel.Archive); err != nil {
		return err
	}
	staging := dest + ".new"
	if err := os.RemoveAll(staging); err != nil {
		return err
	}
	if err := unpack(rel.Archive, staging); err != nil {
		_ = os.RemoveAll(staging)
		return err
	}
	if err := os.WriteFile(filepath.Join(staging, versionFile), []byte(version+"\n"), 0o644); err != nil {
		_ = os.RemoveAll(staging)
		return err
	}
	if _, err := os.Stat(dest); err == nil {
		if err := os.RemoveAll(previousDir(dest)); err != nil {
			return err
		}
		if err := os.Rename(dest, previousDir(dest)); err != nil {
			return err
		}
	}
	if err := os.Rename(staging, dest); err != nil {
		_ = os.Rename(previousDir(dest), dest)
		return err
	}
	return nil
}

// Rollback puts back the bundle the last Place set aside.
func Rollback(dest string) error {
	if _, err := os.Stat(previousDir(dest)); err != nil {
		return errors.New("there is no earlier Tor bundle to go back to")
	}
	if err := os.RemoveAll(dest); err != nil {
		return err
	}
	return os.Rename(previousDir(dest), dest)
}

// InstalledVersion is the release Place last put at dest; empty when the bundle was placed by hand.
func InstalledVersion(dest string) string {
	raw, err := os.ReadFile(filepath.Join(dest, versionFile))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}

func unpack(archive []byte, root string) error {
	if err := os.MkdirAll(root, extractedMode); err != nil {
		return err
	}
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return fmt.Errorf("the Tor bundle is not a gzip archive: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	var written int64
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		target, err := insideRoot(root, hdr.Name)
		if err != nil {
			return err
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			err = os.MkdirAll(target, extractedMode)
		case tar.TypeReg:
			written += hdr.Size
			if written > extractLimit {
				return errors.New("the Tor bundle unpacks to more than it should")
			}
			err = writeFile(target, tr, hdr.FileInfo().Mode().Perm())
		case tar.TypeSymlink:
			err = linkInside(root, target, hdr.Linkname)
		default:
			err = fmt.Errorf("the Tor bundle holds %q, which is neither a file, a folder nor a link", hdr.Name)
		}
		if err != nil {
			return err
		}
	}
}

// insideRoot resolves an archive path under root and refuses one that leaves it.
func insideRoot(root, name string) (string, error) {
	if filepath.IsAbs(name) {
		return "", fmt.Errorf("the Tor bundle holds the absolute path %q", name)
	}
	target := filepath.Join(root, name)
	if !within(root, target) {
		return "", fmt.Errorf("the Tor bundle holds %q, which leaves the install folder", name)
	}
	return target, nil
}

func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func writeFile(target string, r io.Reader, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(target), extractedMode); err != nil {
		return err
	}
	f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode|0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, r); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func linkInside(root, target, linkname string) error {
	if filepath.IsAbs(linkname) || !within(root, filepath.Join(filepath.Dir(target), linkname)) {
		return fmt.Errorf("the Tor bundle holds a link to %q, which leaves the install folder", linkname)
	}
	if err := os.MkdirAll(filepath.Dir(target), extractedMode); err != nil {
		return err
	}
	return os.Symlink(linkname, target)
}
