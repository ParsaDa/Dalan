package tor

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/logger"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/engine"
)

// ErrNotInstalled means an exit exists but the Tor bundle is not on this server.
var ErrNotInstalled = errors.New("Tor is not installed on this server")

// engineName marks the rows ExitRows builds, so the shared supervisor runs only them.
const engineName = "tor"

// ExitRows presents exits as the rows the shared engine supervisor works on; the name is the exit's outbound tag.
func ExitRows(exits []model.TorExit) []model.ServerTunnel {
	rows := make([]model.ServerTunnel, 0, len(exits))
	for _, exit := range exits {
		rows = append(rows, model.ServerTunnel{Name: OutboundTag(exit.Country), Engine: engineName, Port: exit.Port, Enable: exit.Enable})
	}
	return rows
}

// NewSupervisor runs one Tor per enabled exit; bridges is read each time a Tor is (re)started.
func NewSupervisor(layout Layout, bridges func() []string, progress *Progress) *engine.Supervisor {
	return &engine.Supervisor{
		Name: "Tor", Binary: layout.Binary(), Dir: layout.Root,
		Ext:   ".torrc",
		Env:   []string{"LD_LIBRARY_PATH=" + layout.LibraryDir()},
		Wants: func(t model.ServerTunnel) bool { return t.Engine == engineName },
		Render: func(t model.ServerTunnel) (string, error) {
			if err := os.MkdirAll(filepath.Join(layout.Root, "data"), 0o700); err != nil {
				return "", err
			}
			exit := model.TorExit{Country: countryOf(t.Name), Port: t.Port, Enable: t.Enable}
			return RenderConfig(exit, bridges(), layout)
		},
		Args:         func(config string) []string { return []string{"-f", config} },
		NotInstalled: ErrNotInstalled,
		LogLine: func(name, line string) {
			country := countryOf(name)
			if percent, ok := ParseBootstrap(line); ok {
				progress.Set(country, percent)
			}
			logger.Debugf("Tor %s: %s", country, line)
		},
	}
}

func countryOf(name string) string { return strings.TrimPrefix(name, OutboundTag("")) }
