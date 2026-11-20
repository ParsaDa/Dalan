package engine_test

import (
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/backpack"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/engine"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/rakhsh"
)

const (
	portalKey = "IiIiIiIiIiIiIiIiIiIiIiIiIiIiIiIiIiIiIiIiIiI="
	bridgeKey = "REREREREREREREREREREREREREREREREREREREREREQ="
)

func parentOf(engineID string) model.ServerTunnel {
	return model.ServerTunnel{
		Id: 4, Name: "p1", Role: model.TunnelRolePortal, Engine: engineID, Enable: true,
		Address: "203.0.113.10", Port: 8443, UUID: "8d6f3a0e-5c1b-4f0e-9a51-2f6d1c7b9e10", Transport: "tcpmux",
		ForwardPorts: []string{"443=127.0.0.1:2096"}, RelayPort: 63500,
		RakhshSecret: portalKey, RakhshPeerPublic: bridgeKey,
	}
}

func TestNodeManagementRidesAsAnExtraLoopbackForwardOfItsTunnelOnEitherEngine(t *testing.T) {
	for _, engineID := range []string{model.TunnelEngineBackpack, model.TunnelEngineRakhsh} {
		t.Run(engineID, func(t *testing.T) {
			parent := parentOf(engineID)
			if engineID == model.TunnelEngineRakhsh {
				parent.Transport = "wss"
			}
			rider := model.ServerTunnel{
				Id: 5, Name: "node-de1", Role: model.TunnelRolePortal, Engine: engineID, Enable: true,
				ManagePort: 61005, ParentId: 4, ManageTarget: 2053,
			}
			waiting := rider
			waiting.Id, waiting.Name, waiting.ManagePort, waiting.ManageTarget = 6, "node-wait", 61006, 0

			got := engine.WithManagement([]model.ServerTunnel{parent, rider, waiting})
			if len(got) != 1 {
				t.Fatalf("tunnels left for the engine = %d, want only the parent: %+v", len(got), got)
			}
			want := []string{"443=127.0.0.1:2096", "127.0.0.1:61005=127.0.0.1:2053"}
			if strings.Join(got[0].ForwardPorts, "|") != strings.Join(want, "|") {
				t.Fatalf("forwards = %v, want %v (a node that has not joined yet has no target and no forward)", got[0].ForwardPorts, want)
			}
			if len(parent.ForwardPorts) != 1 {
				t.Fatalf("the stored tunnel's forwards were changed: %v", parent.ForwardPorts)
			}
			var rendered string
			var err error
			if engineID == model.TunnelEngineRakhsh {
				rendered, err = rakhsh.RenderConfig(got[0], "")
			} else {
				rendered, err = backpack.RenderConfig(got[0])
			}
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			if !strings.Contains(rendered, "61005") || !strings.Contains(rendered, "2053") {
				t.Fatalf("rendered config lacks the management forward:\n%s", rendered)
			}
		})
	}
}
