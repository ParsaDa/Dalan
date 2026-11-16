package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"runtime"
	"sync"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/logger"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/backpack"
)

// engineSettleTimeout is how long an updated engine has to bring its tunnels back; it exceeds the
// engine's 30 second status interval.
var engineSettleTimeout = 75 * time.Second

// engineSource fetches a BackPack release; an empty version means the newest one.
type engineSource func(ctx context.Context, version, arch string) (backpack.Release, error)

func directEngineSource(ctx context.Context, version, arch string) (backpack.Release, error) {
	client := (&SettingService{}).NewProxiedHTTPClient(engineInstallTimeout)
	if version == "" {
		latest, err := backpack.LatestVersion(ctx, client, backpack.APIBaseURL)
		if err != nil {
			return backpack.Release{}, err
		}
		version = latest
	}
	return backpack.Fetch(ctx, client, backpack.ReleaseBaseURL, version, arch)
}

// nodeEngineSource is where this server gets a release it passes on to a main panel; tests replace it.
var nodeEngineSource engineSource = directEngineSource

const engineDownloadKeep = 10 * time.Minute

type engineDownload struct {
	done    chan struct{}
	release backpack.Release
	err     error
	started time.Time
}

var (
	engineDownloadsMu sync.Mutex
	engineDownloads   = map[string]*engineDownload{}
)

// EngineReleaseChunk serves a BackPack release to a main panel in parts. The first call starts the
// download and answers Pending until it is done, so no single call outlasts the link between the panels.
func (s *TunnelService) EngineReleaseChunk(version string, offset int) (backpack.Chunk, error) {
	if err := checkEngineVersion(version); err != nil {
		return backpack.Chunk{}, err
	}
	arch, err := engineArch()
	if err != nil {
		return backpack.Chunk{}, err
	}

	engineDownloadsMu.Lock()
	for key, d := range engineDownloads {
		if time.Since(d.started) > engineDownloadKeep {
			delete(engineDownloads, key)
		}
	}
	d, ok := engineDownloads[version]
	if !ok {
		d = &engineDownload{done: make(chan struct{}), started: time.Now()}
		engineDownloads[version] = d
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), engineInstallTimeout)
			defer cancel()
			d.release, d.err = nodeEngineSource(ctx, version, arch)
			close(d.done)
		}()
	}
	engineDownloadsMu.Unlock()

	select {
	case <-d.done:
	default:
		return backpack.Chunk{Pending: true}, nil
	}
	if d.err != nil {
		engineDownloadsMu.Lock()
		delete(engineDownloads, version)
		engineDownloadsMu.Unlock()
		return backpack.Chunk{}, d.err
	}
	return d.release.ChunkAt(offset), nil
}

var engineVersionPattern = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)

// checkEngineVersion lets an empty version (the newest) or a plain vX.Y.Z tag through; the tag becomes part of a URL.
func checkEngineVersion(version string) error {
	if version != "" && !engineVersionPattern.MatchString(version) {
		return fmt.Errorf("%q is not a BackPack version such as v1.8.5", version)
	}
	return nil
}

// UpdateEngine installs a BackPack release (empty means the newest). With a node it is downloaded there
// and sent over the management channel, for a server that cannot reach GitHub.
func (s *TunnelService) UpdateEngine(ctx context.Context, nodeID int, version string) error {
	if err := checkEngineVersion(version); err != nil {
		return err
	}
	if nodeID == 0 {
		return s.updateEngineFrom(ctx, directEngineSource, version)
	}
	remote, err := s.remoteFor(nodeID)
	if err != nil {
		return err
	}
	return s.updateEngineFrom(ctx, func(ctx context.Context, version, _ string) (backpack.Release, error) {
		return remote.FetchEngineRelease(ctx, version)
	}, version)
}

func engineArch() (string, error) {
	arch := runtime.GOARCH
	switch arch {
	case "amd64", "arm64", "386", "s390x":
		return arch, nil
	}
	return "", fmt.Errorf("BackPack publishes no build for %s", arch)
}

func (s *TunnelService) connectedBackpackTunnels() []string {
	tunnels, err := s.GetAll()
	if err != nil {
		return nil
	}
	var names []string
	for i := range tunnels {
		if tunnels[i].Engine != "backpack" || !tunnels[i].Enable {
			continue
		}
		if snap, err := backpack.ReadSnapshot(tunnelEngines.Dir, tunnels[i].Name); err == nil && snap.Connected {
			names = append(names, tunnels[i].Name)
		}
	}
	return names
}

func (s *TunnelService) waitConnected(names []string, within time.Duration) bool {
	for deadline := time.Now().Add(within); ; time.Sleep(200 * time.Millisecond) {
		all := true
		for _, name := range names {
			if snap, err := backpack.ReadSnapshot(tunnelEngines.Dir, name); err != nil || !snap.Connected {
				all = false
				break
			}
		}
		if all {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
	}
}

// updateEngineFrom installs the release src returns and restarts the tunnels on it. When tunnels that
// were connected do not reconnect in time, the engine it replaced is put back.
func (s *TunnelService) updateEngineFrom(ctx context.Context, src engineSource, version string) error {
	arch, err := engineArch()
	if err != nil {
		return err
	}
	rel, err := src(ctx, version, arch)
	if err != nil {
		return err
	}
	if version == "" {
		version = rel.Version
	}
	if version == "" {
		return errors.New("the release source did not say which BackPack version it sent")
	}
	binary := tunnelEngines.Binary
	if s.EngineInfo().Installed && backpack.InstalledVersion(binary) == version {
		return nil
	}

	connected := s.connectedBackpackTunnels()
	if err := backpack.Place(rel, version, arch, binary); err != nil {
		return err
	}
	tunnelEngines.StopAll()
	reconcileErr := s.ReconcileEngines()
	if len(connected) == 0 || s.waitConnected(connected, engineSettleTimeout) {
		return reconcileErr
	}

	logger.Warningf("BackPack %s did not bring %v back; rolling back", version, connected)
	if err := backpack.Rollback(binary); err != nil {
		return fmt.Errorf("BackPack %s did not reconnect its tunnels and the earlier engine could not be restored: %w", version, err)
	}
	tunnelEngines.StopAll()
	if err := s.ReconcileEngines(); err != nil {
		logger.Warning("BackPack tunnels after rollback:", err)
	}
	return fmt.Errorf("BackPack %s did not reconnect %d tunnel(s) within %s; rolled back to the previous engine", version, len(connected), engineSettleTimeout)
}
