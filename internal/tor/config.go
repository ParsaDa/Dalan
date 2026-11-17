package tor

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

// Layout says where Tor keeps its state and where the unpacked Tor Expert Bundle lives.
type Layout struct {
	// Root holds each exit's torrc and data directory.
	Root string
	// BundleDir is the unpacked bundle: tor/, data/ and tor/pluggable_transports/.
	BundleDir string
}

// Binary is the tor executable inside the bundle.
func (l Layout) Binary() string { return filepath.Join(l.BundleDir, "tor", "tor") }

// LibraryDir holds the shared libraries the bundled tor loads.
func (l Layout) LibraryDir() string { return filepath.Join(l.BundleDir, "tor") }

// DataDir is one exit's Tor state; it survives restarts so Tor does not fetch the network list again.
func (l Layout) DataDir(country string) string { return filepath.Join(l.Root, "data", country) }

func (l Layout) transport(name string) string {
	return filepath.Join(l.BundleDir, "tor", "pluggable_transports", name)
}

// transportPlugins maps the transports a bridge line may start with to the bundled client that speaks them.
var transportPlugins = []struct {
	names  []string
	binary string
}{
	{[]string{"obfs4", "meek_lite", "webtunnel"}, "lyrebird"},
	{[]string{"snowflake"}, "snowflake-client"},
}

// RenderConfig writes the torrc of one exit: loopback SOCKS only, exits restricted to its country.
func RenderConfig(exit model.TorExit, bridges []string, layout Layout) (string, error) {
	country, ok := NormalizeCountry(exit.Country)
	if !ok || country != exit.Country {
		return "", fmt.Errorf("tor exit country %q is not a lower-case two-letter code", exit.Country)
	}
	if exit.Port < 1 || exit.Port > 65535 {
		return "", fmt.Errorf("tor exit %s: port %d is not a port", country, exit.Port)
	}
	for _, path := range []string{layout.Root, layout.BundleDir} {
		if strings.ContainsAny(path, "\r\n\x00") {
			return "", fmt.Errorf("tor path %q holds a control character", path)
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "DataDirectory %q\n", layout.DataDir(country))
	fmt.Fprintf(&b, "GeoIPFile %q\n", filepath.Join(layout.BundleDir, "data", "geoip"))
	fmt.Fprintf(&b, "GeoIPv6File %q\n", filepath.Join(layout.BundleDir, "data", "geoip6"))
	fmt.Fprintf(&b, "SocksPort 127.0.0.1:%d\n", exit.Port)
	b.WriteString("ClientOnly 1\n")
	fmt.Fprintf(&b, "ExitNodes {%s}\n", country)
	b.WriteString("StrictNodes 1\n")
	b.WriteString("Log notice stdout\n")
	if len(bridges) == 0 {
		return b.String(), nil
	}
	if strings.ContainsAny(layout.BundleDir, " \t") {
		return "", fmt.Errorf("tor bundle path %q holds a space, which a bridge transport command cannot carry", layout.BundleDir)
	}
	b.WriteString("UseBridges 1\n")
	for _, line := range bridges {
		fmt.Fprintf(&b, "Bridge %s\n", line)
	}
	for _, plugin := range transportPlugins {
		if usesTransport(bridges, plugin.names) {
			fmt.Fprintf(&b, "ClientTransportPlugin %s exec %s\n", strings.Join(plugin.names, ","), layout.transport(plugin.binary))
		}
	}
	return b.String(), nil
}

func usesTransport(bridges, names []string) bool {
	for _, line := range bridges {
		first, _, _ := strings.Cut(line, " ")
		for _, name := range names {
			if first == name {
				return true
			}
		}
	}
	return false
}
