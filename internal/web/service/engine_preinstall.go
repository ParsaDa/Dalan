package service

import (
	"context"
	"os"
	"runtime"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/tor"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/engine"
)

// preinstallTimeout bounds each engine's download, so one blocked release server cannot stall an install.
var preinstallTimeout = 2 * time.Minute

// EngineInstall says what happened to one engine when the install script asked for all of them.
type EngineInstall struct {
	Label   string
	Version string
	// Installed is true when the engine is on this server after the call, whether it was there before or not.
	Installed bool
	Err       error
}

// InstallMissingEngines puts every engine that is not on this server yet in place at its pinned version.
// It never starts an engine and never fails as a whole: each engine reports its own outcome. route is how the downloads travel.
func InstallMissingEngines(ctx context.Context, route EngineRoute) []EngineInstall {
	client, release, err := (&EngineRouteService{}).Client(route, engineInstallTimeout)
	if err != nil {
		unreachable := func(context.Context) (engine.Release, error) { return engine.Release{}, err }
		return installMissingEngines(ctx, func(ctx context.Context, _, _, _ string) (engine.Release, error) { return unreachable(ctx) }, unreachable)
	}
	defer release()
	return installMissingEngines(ctx, engineSourceWith(client), func(ctx context.Context) (engine.Release, error) {
		return tor.Fetch(ctx, client, torReleaseBase, tor.PinnedVersion, runtime.GOARCH)
	})
}

func installMissingEngines(ctx context.Context, src engineSource, fetchTor func(context.Context) (engine.Release, error)) []EngineInstall {
	var results []EngineInstall
	for _, kind := range allExternalEngines() {
		engineCtx, cancel := context.WithTimeout(ctx, preinstallTimeout)
		results = append(results, installExternalEngine(engineCtx, kind, src))
		cancel()
	}
	torCtx, cancel := context.WithTimeout(ctx, preinstallTimeout)
	defer cancel()
	return append(results, installTorBundle(torCtx, fetchTor))
}

func installExternalEngine(ctx context.Context, kind externalEngine, src engineSource) EngineInstall {
	binary := kind.supervisor().Binary
	result := EngineInstall{Label: kind.label, Version: kind.pinned}
	if _, err := os.Stat(binary); err == nil {
		result.Installed, result.Version = true, kind.installed(binary)
		return result
	}
	arch, err := kind.arch()
	if err != nil {
		result.Err = err
		return result
	}
	rel, err := src(ctx, kind.id, kind.pinned, arch)
	if err == nil {
		err = kind.place(rel, kind.pinned, arch, binary)
	}
	result.Installed, result.Err = err == nil, err
	return result
}

func installTorBundle(ctx context.Context, fetch func(context.Context) (engine.Release, error)) EngineInstall {
	result := EngineInstall{Label: "Tor", Version: tor.PinnedVersion}
	if _, err := os.Stat(torLayout.Binary()); err == nil {
		result.Installed, result.Version = true, tor.InstalledVersion(torLayout.BundleDir)
		return result
	}
	rel, err := fetch(ctx)
	if err == nil {
		err = tor.Place(rel, tor.PinnedVersion, runtime.GOARCH, torLayout.BundleDir)
	}
	result.Installed, result.Err = err == nil, err
	return result
}
