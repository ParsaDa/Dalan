package engine

import (
	"fmt"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

// ManagementForward is the loopback forward a node's management channel adds to its tunnel.
func ManagementForward(managePort, nodePanelPort int) string {
	return fmt.Sprintf("127.0.0.1:%d=127.0.0.1:%d", managePort, nodePanelPort)
}

// WithManagement gives each external-engine tunnel the forwards of the node channels riding on it and drops the channels.
func WithManagement(tunnels []model.ServerTunnel) []model.ServerTunnel {
	forwards := map[int][]string{}
	for _, t := range tunnels {
		if t.ParentId > 0 && t.Enable && t.ManageTarget > 0 {
			forwards[t.ParentId] = append(forwards[t.ParentId], ManagementForward(t.ManagePort, t.ManageTarget))
		}
	}
	out := make([]model.ServerTunnel, 0, len(tunnels))
	for _, t := range tunnels {
		if t.ParentId > 0 {
			continue
		}
		if extra := forwards[t.Id]; len(extra) > 0 && model.ExternalEngine(t.Engine) {
			t.ForwardPorts = append(append([]string(nil), t.ForwardPorts...), extra...)
		}
		out = append(out, t)
	}
	return out
}
