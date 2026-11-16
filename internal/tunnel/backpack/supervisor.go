package backpack

import (
	"errors"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/engine"
)

// ErrNotInstalled means a BackPack tunnel exists but the engine binary is not on this server.
var ErrNotInstalled = errors.New("the BackPack engine is not installed on this server")

// NewSupervisor runs one BackPack process per enabled BackPack tunnel, with state kept in dir.
func NewSupervisor(binary, dir string) *engine.Supervisor {
	return &engine.Supervisor{
		Name: "BackPack", Binary: binary, Dir: dir,
		Wants:        func(t model.ServerTunnel) bool { return t.Engine == model.TunnelEngineBackpack },
		Render:       RenderConfig,
		Args:         func(config string) []string { return []string{"-c", config} },
		NotInstalled: ErrNotInstalled,
		Leftovers:    func(dir, name string) []string { return []string{snapshotPath(dir, name)} },
	}
}
