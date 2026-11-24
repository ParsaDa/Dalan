package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/config"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/logger"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/backpack"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/engine"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/forwarder"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/rakhsh"
)

// engineSettleTimeout is how long an updated engine has to bring its tunnels back; it exceeds the
// engine's 30 second status interval.
var engineSettleTimeout = 75 * time.Second

// externalEngine is what the panel needs to run, install and watch an engine that runs as its own process.
type externalEngine struct {
	id, label  string
	supervisor func() *engine.Supervisor
	pinned     string
	arch       func() (string, error)
	latest     func(ctx context.Context, client *http.Client) (string, error)
	recent     func(ctx context.Context, client *http.Client, limit int) ([]string, error)
	fetch      func(ctx context.Context, client *http.Client, version, arch string) (engine.Release, error)
	place      func(rel engine.Release, version, arch, dest string) error
	rollback   func(dest string) error
	installed  func(dest string) string
	// connected says whether the engine currently reports this tunnel as connected.
	connected func(dir string, t model.ServerTunnel) bool
	// preinstalled engines are put in place by the install script; the others are installed from the panel.
	preinstalled bool
}

func backpackEngine() externalEngine {
	return externalEngine{
		id: model.TunnelEngineBackpack, label: "BackPack", preinstalled: true,
		supervisor: func() *engine.Supervisor { return tunnelEngines },
		pinned:     backpack.PinnedVersion,
		arch:       backpackArch,
		latest: func(ctx context.Context, client *http.Client) (string, error) {
			return backpack.LatestVersion(ctx, client, backpack.APIBaseURL)
		},
		recent: func(ctx context.Context, client *http.Client, limit int) ([]string, error) {
			return backpack.RecentVersions(ctx, client, backpack.APIBaseURL, limit)
		},
		fetch: func(ctx context.Context, client *http.Client, version, arch string) (engine.Release, error) {
			return backpack.Fetch(ctx, client, backpack.ReleaseBaseURL, version, arch)
		},
		place:     backpack.Place,
		rollback:  backpack.Rollback,
		installed: backpack.InstalledVersion,
		connected: func(dir string, t model.ServerTunnel) bool {
			snap, err := backpack.ReadSnapshot(dir, t.Name)
			return err == nil && snap.Connected
		},
	}
}

func rakhshEngine() externalEngine {
	return externalEngine{
		id: model.TunnelEngineRakhsh, label: "Rakhsh", preinstalled: true,
		supervisor: func() *engine.Supervisor { return rakhshEngines },
		pinned:     rakhsh.PinnedVersion,
		arch:       func() (string, error) { return runtime.GOARCH, nil },
		latest: func(ctx context.Context, client *http.Client) (string, error) {
			return rakhsh.LatestVersion(ctx, client, rakhsh.APIBaseURL)
		},
		recent: func(ctx context.Context, client *http.Client, limit int) ([]string, error) {
			return rakhsh.RecentVersions(ctx, client, rakhsh.APIBaseURL, limit)
		},
		fetch: func(ctx context.Context, client *http.Client, version, arch string) (engine.Release, error) {
			return rakhsh.Fetch(ctx, client, rakhsh.ReleaseBaseURL, version, arch)
		},
		place:     rakhsh.Place,
		rollback:  rakhsh.Rollback,
		installed: rakhsh.InstalledVersion,
		connected: func(dir string, t model.ServerTunnel) bool {
			ctx, cancel := context.WithTimeout(context.Background(), rakhshStatusTimeout)
			defer cancel()
			st, err := rakhsh.ReadStatus(ctx, rakhshEngines.Binary, rakhsh.ControlSocket(dir, t.Name))
			return err == nil && st.Connected
		},
	}
}

const rakhshStatusTimeout = 5 * time.Second

// forwarderStatusTimeout bounds the greeting a forwarding engine's relay is asked to answer.
const forwarderStatusTimeout = 5 * time.Second

// forwarderEngines run the GOST, Rathole, Backhaul, Kariz, sing-box and Xray processes of this panel,
// keyed by engine id; tests replace entries with ones on a fake engine.
var forwarderEngines = newForwarderEngines()

func newForwarderEngines() map[string]*engine.Supervisor {
	out := map[string]*engine.Supervisor{}
	for _, e := range forwarder.Registry() {
		out[e.ID] = e.NewSupervisor(
			filepath.Join(config.GetBinFolderPath(), e.ID),
			filepath.Join(config.GetDBFolderPath(), "tunnels"),
		)
	}
	return out
}

func forwarderKind(e forwarder.Engine) externalEngine {
	return externalEngine{
		id: e.ID, label: e.Label,
		supervisor: func() *engine.Supervisor { return forwarderEngines[e.ID] },
		pinned:     e.Pinned,
		arch:       func() (string, error) { return forwarder.Arch(), nil },
		latest: func(ctx context.Context, client *http.Client) (string, error) {
			return e.LatestVersion(ctx, client, forwarder.APIBaseURL)
		},
		recent: func(ctx context.Context, client *http.Client, limit int) ([]string, error) {
			return e.RecentVersions(ctx, client, forwarder.APIBaseURL, limit)
		},
		fetch: func(ctx context.Context, client *http.Client, version, arch string) (engine.Release, error) {
			return e.Fetch(ctx, client, forwarder.ReleaseBaseURL, version, arch)
		},
		place:     e.Place,
		rollback:  e.Rollback,
		installed: e.InstalledVersion,
		connected: func(_ string, t model.ServerTunnel) bool {
			_, ok := forwarder.Check(t, forwarderEngines[e.ID].Running, forwarderStatusTimeout)
			return ok
		},
	}
}

var errEngineHasNoBinary = errors.New("this tunnel engine runs inside the panel and has no engine to manage")

// externalEngineFor finds an engine by its id; an empty id means BackPack, as before there was a second one.
func externalEngineFor(id string) (externalEngine, error) {
	switch id {
	case "", model.TunnelEngineBackpack:
		return backpackEngine(), nil
	case model.TunnelEngineRakhsh:
		return rakhshEngine(), nil
	}
	if e, ok := forwarder.ByID(id); ok {
		return forwarderKind(e), nil
	}
	return externalEngine{}, fmt.Errorf("tunnel engine %q: %w", id, errEngineHasNoBinary)
}

func allExternalEngines() []externalEngine {
	kinds := []externalEngine{backpackEngine(), rakhshEngine()}
	for _, e := range forwarder.Registry() {
		kinds = append(kinds, forwarderKind(e))
	}
	return kinds
}

// engineSource fetches a release of one engine; an empty version means the newest one.
type engineSource func(ctx context.Context, id, version, arch string) (engine.Release, error)

func directEngineSource(ctx context.Context, id, version, arch string) (engine.Release, error) {
	return engineSourceWith((&SettingService{}).NewProxiedHTTPClient(engineInstallTimeout))(ctx, id, version, arch)
}

// engineSourceWith downloads releases through client.
func engineSourceWith(client *http.Client) engineSource {
	return func(ctx context.Context, id, version, arch string) (engine.Release, error) {
		kind, err := externalEngineFor(id)
		if err != nil {
			return engine.Release{}, err
		}
		if version == "" {
			latest, err := kind.latest(ctx, client)
			if err != nil {
				return engine.Release{}, err
			}
			version = latest
		}
		return kind.fetch(ctx, client, version, arch)
	}
}

// nodeEngineSource is where this server gets a release it passes on to a main panel; tests replace it.
var nodeEngineSource engineSource = directEngineSource

const engineDownloadKeep = 10 * time.Minute

type engineDownload struct {
	done    chan struct{}
	release engine.Release
	err     error
	started time.Time
}

var (
	engineDownloadsMu sync.Mutex
	engineDownloads   = map[string]*engineDownload{}
)

// EngineReleaseChunk serves a release of one engine to a main panel in parts. The first call starts the
// download and answers Pending until it is done, so no single call outlasts the link between the panels.
func (s *TunnelService) EngineReleaseChunk(id, version string, offset int) (engine.Chunk, error) {
	kind, err := externalEngineFor(id)
	if err != nil {
		return engine.Chunk{}, err
	}
	if err := checkEngineVersion(kind, version); err != nil {
		return engine.Chunk{}, err
	}
	arch, err := kind.arch()
	if err != nil {
		return engine.Chunk{}, err
	}

	key := kind.id + "/" + version
	engineDownloadsMu.Lock()
	for k, d := range engineDownloads {
		if time.Since(d.started) > engineDownloadKeep {
			delete(engineDownloads, k)
		}
	}
	d, ok := engineDownloads[key]
	if !ok {
		d = &engineDownload{done: make(chan struct{}), started: time.Now()}
		engineDownloads[key] = d
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), engineInstallTimeout)
			defer cancel()
			d.release, d.err = nodeEngineSource(ctx, kind.id, version, arch)
			close(d.done)
		}()
	}
	engineDownloadsMu.Unlock()

	select {
	case <-d.done:
	default:
		return engine.Chunk{Pending: true}, nil
	}
	if d.err != nil {
		engineDownloadsMu.Lock()
		delete(engineDownloads, key)
		engineDownloadsMu.Unlock()
		return engine.Chunk{}, d.err
	}
	return d.release.ChunkAt(offset), nil
}

var engineVersionPattern = engine.VersionPattern

// checkEngineVersion lets an empty version (the newest) or a plain vX.Y.Z tag through; the tag becomes part of a URL.
func checkEngineVersion(kind externalEngine, version string) error {
	if version != "" && !engineVersionPattern.MatchString(version) {
		return fmt.Errorf("%q is not a %s version such as v1.8.5", version, kind.label)
	}
	return nil
}

// engineSourceFor opens the release source of a route; tests replace it.
var engineSourceFor = func(route EngineRoute) (engineSource, func(), error) {
	return (&EngineRouteService{}).Source(route)
}

// engineListClient opens the client a release list is read through; tests replace it.
var engineListClient = func(route EngineRoute) (*http.Client, func(), error) {
	return (&EngineRouteService{}).Client(route, engineListTimeout)
}

const (
	// engineReleaseChoices is how many recent releases the install dialog offers.
	engineReleaseChoices = 4
	// engineListTimeout bounds one release list; the list is small, so a slow route should fail rather than hang.
	engineListTimeout = 30 * time.Second
)

// EngineReleases lists the newest releases of one engine. With a node the node reads the list, since it may
// reach GitHub where this server cannot; otherwise this server reads it through route.
func (s *TunnelService) EngineReleases(ctx context.Context, nodeID int, id string, route EngineRoute) ([]string, error) {
	kind, err := externalEngineFor(id)
	if err != nil {
		return nil, err
	}
	if nodeID != 0 {
		remote, err := s.remoteFor(nodeID)
		if err != nil {
			return nil, err
		}
		return remote.ListEngineReleases(ctx, kind.id)
	}
	client, release, err := engineListClient(route)
	if err != nil {
		return nil, err
	}
	defer release()
	versions, err := kind.recent(ctx, client, engineReleaseChoices)
	if err != nil {
		return nil, fmt.Errorf("could not list the %s releases: %w", kind.label, err)
	}
	return versions, nil
}

// UpdateEngine installs a release of one engine (empty version means the newest), downloaded through route. With a
// node it is downloaded there and sent over the management channel, for a server that cannot reach GitHub.
func (s *TunnelService) UpdateEngine(ctx context.Context, nodeID int, id, version string, route EngineRoute) error {
	kind, err := externalEngineFor(id)
	if err != nil {
		return err
	}
	if err := checkEngineVersion(kind, version); err != nil {
		return err
	}
	if nodeID == 0 {
		src, release, err := engineSourceFor(route)
		if err != nil {
			return err
		}
		defer release()
		return s.updateEngineFrom(ctx, kind, src, version)
	}
	remote, err := s.remoteFor(nodeID)
	if err != nil {
		return err
	}
	return s.updateEngineFrom(ctx, kind, func(ctx context.Context, id, version, _ string) (engine.Release, error) {
		return remote.FetchEngineRelease(ctx, id, version)
	}, version)
}

func backpackArch() (string, error) {
	arch := runtime.GOARCH
	switch arch {
	case "amd64", "arm64", "386", "s390x":
		return arch, nil
	}
	return "", fmt.Errorf("BackPack publishes no build for %s", arch)
}

func (s *TunnelService) connectedTunnels(kind externalEngine) []model.ServerTunnel {
	tunnels, err := s.GetAll()
	if err != nil {
		return nil
	}
	dir := kind.supervisor().Dir
	var connected []model.ServerTunnel
	for _, t := range tunnels {
		if t.Engine == kind.id && t.Enable && kind.connected(dir, t) {
			connected = append(connected, t)
		}
	}
	return connected
}

func (s *TunnelService) waitConnected(kind externalEngine, tunnels []model.ServerTunnel, within time.Duration) bool {
	dir := kind.supervisor().Dir
	for deadline := time.Now().Add(within); ; time.Sleep(200 * time.Millisecond) {
		all := true
		for _, t := range tunnels {
			if !kind.connected(dir, t) {
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
func (s *TunnelService) updateEngineFrom(ctx context.Context, kind externalEngine, src engineSource, version string) error {
	arch, err := kind.arch()
	if err != nil {
		return err
	}
	rel, err := src(ctx, kind.id, version, arch)
	if err != nil {
		return err
	}
	if version == "" {
		version = rel.Version
	}
	if version == "" {
		return fmt.Errorf("the release source did not say which %s version it sent", kind.label)
	}
	supervisor := kind.supervisor()
	binary := supervisor.Binary
	if info, _ := s.EngineInfo(kind.id); info.Installed && kind.installed(binary) == version {
		return nil
	}

	connected := s.connectedTunnels(kind)
	if err := kind.place(rel, version, arch, binary); err != nil {
		return err
	}
	supervisor.StopAll()
	reconcileErr := s.ReconcileEngines()
	if len(connected) == 0 || s.waitConnected(kind, connected, engineSettleTimeout) {
		return reconcileErr
	}

	logger.Warningf("%s %s did not bring %d tunnel(s) back; rolling back", kind.label, version, len(connected))
	if err := kind.rollback(binary); err != nil {
		return fmt.Errorf("%s %s did not reconnect its tunnels and the earlier engine could not be restored: %w", kind.label, version, err)
	}
	supervisor.StopAll()
	if err := s.ReconcileEngines(); err != nil {
		logger.Warningf("%s tunnels after rollback: %v", kind.label, err)
	}
	return fmt.Errorf("%s %s did not reconnect %d tunnel(s) within %s; rolled back to the previous engine", kind.label, version, len(connected), engineSettleTimeout)
}
