package config

import (
	_ "embed"
	"strings"
)

const (
	// RepoSlug is the GitHub repository releases and update scripts are fetched from.
	RepoSlug = "ParsaDa/dalan"
	// DisplayName is the product name shown to operators.
	DisplayName = "Dalan"
)

//go:embed dalan_version
var dalanVersion string

// GetDalanVersion is Dalan's own release version; GetBaseVersion stays the 3x-ui
// version it is built on, because node protocol gates compare against that.
func GetDalanVersion() string {
	return strings.TrimSpace(dalanVersion)
}
