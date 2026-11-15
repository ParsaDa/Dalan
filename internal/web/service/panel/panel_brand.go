package panel

import "github.com/mhsanaei/3x-ui/v3/internal/config"

// githubAPIBase is a variable so tests can point release lookups at a local server.
var githubAPIBase = "https://api.github.com"

func panelUpdaterScriptURL() string {
	return "https://raw.githubusercontent.com/" + config.RepoSlug + "/main/update.sh"
}

func panelReleaseAPIURL(tag string) string {
	base := githubAPIBase + "/repos/" + config.RepoSlug + "/releases/"
	if tag == "" {
		return base + "latest"
	}
	return base + "tags/" + tag
}
