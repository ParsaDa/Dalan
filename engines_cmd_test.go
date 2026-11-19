package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
)

func TestEngineReportListsWhatIsReadyAndNamesWhatWasNotInstalledWithoutErrors(t *testing.T) {
	got := engineReport([]service.EngineInstall{
		{Label: "BackPack", Version: "v1.9.0", Installed: true},
		{Label: "Rakhsh", Version: "v0.1.0", Err: errors.New("dial tcp 140.82.112.3:443: i/o timeout")},
		{Label: "Tor", Version: "16.0.1", Err: errors.New("403 Forbidden")},
	})
	for _, want := range []string{"BackPack v1.9.0", "Rakhsh, Tor", "not installed", "from the panel"} {
		if !strings.Contains(got, want) {
			t.Errorf("report lacks %q:\n%s", want, got)
		}
	}
	for _, leak := range []string{"i/o timeout", "403", "rror", "ailed"} {
		if strings.Contains(got, leak) {
			t.Errorf("report shows %q, but an engine that could not be installed is not an error:\n%s", leak, got)
		}
	}
}

func TestEngineReportSaysNothingIsMissingWhenEverythingIsInstalled(t *testing.T) {
	got := engineReport([]service.EngineInstall{
		{Label: "BackPack", Version: "v1.9.0", Installed: true},
		{Label: "Tor", Version: "16.0.1", Installed: true},
	})
	if strings.Contains(got, "not installed") || !strings.Contains(got, "BackPack v1.9.0, Tor 16.0.1") {
		t.Fatalf("report = %q, want the ready engines and no missing notice", got)
	}
}

func TestTheInstallScriptCanSendEngineDownloadsThroughAProxyOrLink(t *testing.T) {
	t.Setenv("XUI_ENGINE_PROXY", "  socks5://127.0.0.1:1080 ")
	if got := engineRouteFromEnv(); got.Via != service.RouteLink || got.Value != "socks5://127.0.0.1:1080" {
		t.Fatalf("route = %+v, want the proxy as a link route", got)
	}
	t.Setenv("XUI_ENGINE_PROXY", "")
	if got := engineRouteFromEnv(); got.Via != service.RoutePanel {
		t.Fatalf("route = %+v, want the panel's own setting when the variable is empty", got)
	}
}
