package panel

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database/dbtest"
)

func TestPanelReleaseAPIURL(t *testing.T) {
	cases := []struct{ tag, want string }{
		{"", "https://api.github.com/repos/ParsaDa/dalan/releases/latest"},
		{"dev-latest", "https://api.github.com/repos/ParsaDa/dalan/releases/tags/dev-latest"},
	}
	for _, c := range cases {
		if got := panelReleaseAPIURL(c.tag); got != c.want {
			t.Errorf("panelReleaseAPIURL(%q) = %q, want %q", c.tag, got, c.want)
		}
	}
}

func TestPanelUpdaterScriptURL(t *testing.T) {
	want := "https://raw.githubusercontent.com/ParsaDa/dalan/main/update.sh"
	if got := panelUpdaterScriptURL(); got != want {
		t.Errorf("panelUpdaterScriptURL() = %q, want %q", got, want)
	}
}

func serveLatestRelease(t *testing.T, status int, body string) {
	t.Helper()
	dbtest.InitDB(t, filepath.Join(t.TempDir(), "x-ui.db"))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/ParsaDa/dalan/releases/latest" {
			t.Errorf("release lookup hit %q", r.URL.Path)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	previous := githubAPIBase
	githubAPIBase = srv.URL
	t.Cleanup(func() { githubAPIBase = previous })
}

func TestUpdateInfoComparesDalanReleases(t *testing.T) {
	cases := []struct {
		name, latestTag string
		wantAvailable   bool
	}{
		{"newer release", "v0.1.1", true},
		{"same release", "v0.1.0", false},
		{"older release", "v0.0.9", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			serveLatestRelease(t, http.StatusOK, `{"tag_name":"`+c.latestTag+`"}`)
			info, err := (&PanelService{}).GetUpdateInfo()
			if err != nil {
				t.Fatalf("GetUpdateInfo: %v", err)
			}
			if info.CurrentVersion != "0.1.0" {
				t.Errorf("CurrentVersion = %q, want %q", info.CurrentVersion, "0.1.0")
			}
			if info.UpdateAvailable != c.wantAvailable {
				t.Errorf("UpdateAvailable = %v, want %v", info.UpdateAvailable, c.wantAvailable)
			}
		})
	}
}

func TestUpdateInfoReportsMissingRelease(t *testing.T) {
	serveLatestRelease(t, http.StatusNotFound, `{"message":"Not Found"}`)
	_, err := (&PanelService{}).GetUpdateInfo()
	if err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("GetUpdateInfo error = %v, want one naming status 404", err)
	}
}
