package forwarder

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"regexp"
	"runtime"
	"strings"

	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/engine"
)

const (
	// ReleaseBaseURL is where the engines publish their releases; APIBaseURL names the newest one.
	ReleaseBaseURL = "https://github.com"
	APIBaseURL     = "https://api.github.com"

	maxBinary = 200 << 20
)

// Asset is the release file of one engine for one processor. Member is the binary inside an archive;
// empty means the file itself is the binary.
type Asset struct {
	Name, Member string
	// SumsName is a checksum file the engine publishes under another name than the usual ones.
	SumsName string
}

// Arch is the processor Dalan installs for on this server.
func Arch() string { return runtime.GOARCH }

// Fetch downloads a release of the engine and the checksums published beside it, when there are any.
func (e Engine) Fetch(ctx context.Context, client *http.Client, baseURL, version, arch string) (engine.Release, error) {
	a, err := e.asset(version, arch)
	if err != nil {
		return engine.Release{}, err
	}
	base := fmt.Sprintf("%s/%s/releases/download/%s", baseURL, e.Repo, version)
	archive, err := engine.Download(ctx, client, base+"/"+a.Name)
	if err != nil {
		return engine.Release{}, err
	}
	rel := engine.Release{Version: version, Archive: archive}
	names := []string{a.Name + ".sha256", a.Name + ".dgst", "checksums.txt", "SHA256SUMS"}
	if a.SumsName != "" {
		names = append([]string{a.SumsName}, names...)
	}
	for _, name := range names {
		body, err := engine.Download(ctx, client, base+"/"+name)
		if err != nil {
			continue
		}
		if sums := normalizeSums(a.Name, body); len(sums) > 0 {
			rel.Sums = sums
			break
		}
	}
	return rel, nil
}

var hashPattern = regexp.MustCompile(`\b[a-fA-F0-9]{64}\b`)

// normalizeSums turns a checksum file into "hash  file" lines; a file that holds one bare hash is read as
// the hash of asset.
func normalizeSums(asset string, body []byte) []byte {
	text := strings.TrimSpace(string(body))
	if text == "" {
		return nil
	}
	if !strings.ContainsAny(text, "\n \t") && hashPattern.MatchString(text) {
		return []byte(strings.ToLower(text) + "  " + asset + "\n")
	}
	var out strings.Builder
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && hashPattern.MatchString(fields[0]) {
			out.WriteString(strings.ToLower(fields[0]) + "  " + strings.TrimPrefix(fields[1], "*") + "\n")
		}
	}
	return []byte(out.String())
}

// Place checks rel against the pinned sum of the engine's release or, for another version, the checksum
// the release publishes, and only then replaces the engine at dest with the binary inside it.
func (e Engine) Place(rel engine.Release, version, arch, dest string) error {
	a, err := e.asset(version, arch)
	if err != nil {
		return err
	}
	if pin, pinned := e.Pins[a.Name]; pinned && version == e.Pinned {
		sum := sha256.Sum256(rel.Archive)
		if got := hex.EncodeToString(sum[:]); got != pin {
			return fmt.Errorf("checksum mismatch for %s: got %s, pinned %s", a.Name, got, pin)
		}
	} else if err := engine.CheckChecksum(rel.Sums, a.Name, rel.Archive); err != nil {
		return err
	}
	binary, err := extract(a, rel.Archive)
	if err != nil {
		return fmt.Errorf("%s: %w", e.Label, err)
	}
	return engine.Replace(dest, binary, version)
}

var errAmbiguousMember = errors.New("the release archive must hold exactly one engine binary")

func extract(a Asset, archive []byte) ([]byte, error) {
	switch {
	case a.Member == "":
		return archive, nil
	case strings.HasSuffix(a.Name, ".zip"):
		return fromZip(a.Member, archive)
	default:
		return fromTarGz(a.Member, archive)
	}
}

func fromTarGz(member string, archive []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(gz)
	var found []byte
	count := 0
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if h.Typeflag != tar.TypeReg || path.Base(h.Name) != member {
			continue
		}
		count++
		if found, err = io.ReadAll(io.LimitReader(tr, maxBinary)); err != nil {
			return nil, err
		}
	}
	if count != 1 {
		return nil, fmt.Errorf("%w (found %d named %s)", errAmbiguousMember, count, member)
	}
	return found, nil
}

func fromZip(member string, archive []byte) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, err
	}
	var file *zip.File
	count := 0
	for _, f := range zr.File {
		if !f.FileInfo().IsDir() && path.Base(f.Name) == member {
			file = f
			count++
		}
	}
	if count != 1 {
		return nil, fmt.Errorf("%w (found %d named %s)", errAmbiguousMember, count, member)
	}
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(io.LimitReader(rc, maxBinary))
}

// LatestVersion is the tag of the newest release of the engine.
func (e Engine) LatestVersion(ctx context.Context, client *http.Client, apiURL string) (string, error) {
	return engine.LatestTag(ctx, client, apiURL, e.Repo, e.Label)
}

// RecentVersions lists the engine's newest plain releases, newest first.
func (e Engine) RecentVersions(ctx context.Context, client *http.Client, apiURL string, limit int) ([]string, error) {
	return engine.RecentTags(ctx, client, apiURL, e.Repo, e.Label, limit)
}

// Rollback puts back the engine the last Place replaced.
func (e Engine) Rollback(dest string) error { return engine.Rollback(dest, e.Label) }

// InstalledVersion is the release Place last put at dest; empty when the engine was placed by hand.
func (e Engine) InstalledVersion(dest string) string { return engine.InstalledVersion(dest) }
