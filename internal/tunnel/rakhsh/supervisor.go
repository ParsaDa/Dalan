package rakhsh

import (
	"errors"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/engine"
)

// ErrNotInstalled means a Rakhsh tunnel exists but the engine binary is not on this server.
var ErrNotInstalled = errors.New("the Rakhsh engine is not installed on this server")

// NewSupervisor runs one Rakhsh process per enabled Rakhsh tunnel, with state kept in dir.
func NewSupervisor(binary, dir string) *engine.Supervisor {
	return &engine.Supervisor{
		Name: "Rakhsh", Binary: binary, Dir: dir,
		Wants: func(t model.ServerTunnel) bool { return t.Engine == model.TunnelEngineRakhsh },
		Render: func(t model.ServerTunnel) (string, error) {
			return RenderConfig(t, ControlSocket(dir, t.Name))
		},
		Args:         func(config string) []string { return []string{"run", "-c", config} },
		NotInstalled: ErrNotInstalled,
		Leftovers:    func(dir, name string) []string { return []string{ControlSocket(dir, name)} },
	}
}
