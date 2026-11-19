package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mhsanaei/3x-ui/v3/internal/config"
	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
)

// installEngines runs "x-ui install-engines": every tunnel engine goes in place, and a missing one is only reported.
// It always exits 0, since the panel works without them and the install script must not stop here.
func installEngines() {
	if err := database.InitDB(config.GetDBPath()); err != nil {
		fmt.Println("Tunnel engines were not installed. Install them later from the panel.")
		return
	}
	fmt.Print(engineReport(service.InstallMissingEngines(context.Background(), engineRouteFromEnv())))
	_ = os.Stdout.Sync()
}

// engineRouteFromEnv lets a server that cannot reach GitHub give a proxy address or share link as XUI_ENGINE_PROXY.
func engineRouteFromEnv() service.EngineRoute {
	if value := strings.TrimSpace(os.Getenv("XUI_ENGINE_PROXY")); value != "" {
		return service.EngineRoute{Via: service.RouteLink, Value: value}
	}
	return service.EngineRoute{Via: service.RoutePanel}
}

// engineReport words the outcome for the install script; an engine that could not be fetched is not shown as an error.
func engineReport(results []service.EngineInstall) string {
	var ready, missing []string
	for _, r := range results {
		if r.Installed {
			ready = append(ready, r.Label+" "+r.Version)
		} else {
			missing = append(missing, r.Label)
		}
	}
	var b strings.Builder
	if len(ready) > 0 {
		fmt.Fprintf(&b, "Tunnel engines ready: %s.\n", strings.Join(ready, ", "))
	}
	if len(missing) > 0 {
		fmt.Fprintf(&b, "Some engines were not installed (%s). Install them later from the panel.\n", strings.Join(missing, ", "))
	}
	return b.String()
}
