package panel

import "testing"

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
