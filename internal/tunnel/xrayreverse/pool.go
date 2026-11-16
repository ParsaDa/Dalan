package xrayreverse

import (
	"encoding/json"
	"fmt"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/util/json_util"
	"github.com/mhsanaei/3x-ui/v3/internal/xray"
)

// PoolBalancerTag is the balancer a pool appears as; routing rules may target it.
func PoolBalancerTag(name string) string { return "pool-" + name }

// Routable reports whether an entry tunnel exists as an outbound in the generated config.
func Routable(t model.ServerTunnel) bool {
	if !t.Enable || t.Role != model.TunnelRolePortal || t.ManagePort > 0 {
		return false
	}
	return t.Engine != model.TunnelEngineBackpack || t.RelayPort > 0
}

// ActiveMember is the pool's member that carries traffic: the stored active one while it still
// works, else the first member that does. It is "" when no member can carry anything.
func ActiveMember(pool model.TunnelPool, tunnels []model.ServerTunnel) string {
	usable := map[string]bool{}
	for _, t := range tunnels {
		if Routable(t) {
			usable[t.Name] = true
		}
	}
	if usable[pool.Active] {
		for _, m := range pool.Members {
			if m == pool.Active {
				return m
			}
		}
	}
	for _, m := range pool.Members {
		if usable[m] {
			return m
		}
	}
	return ""
}

// ApplyPools adds one balancer per pool, pointing at the pool's active tunnel, and routes the
// pool's inbounds to it. The panel moves the balancer between members while Xray runs.
func ApplyPools(cfg *xray.Config, tunnels []model.ServerTunnel, pools []model.TunnelPool) error {
	routing := map[string]any{}
	if len(cfg.RouterConfig) > 0 {
		if err := json.Unmarshal(cfg.RouterConfig, &routing); err != nil {
			return fmt.Errorf("tunnels: routing section is unparsable: %w", err)
		}
	}
	balancers, _ := routing["balancers"].([]any)
	existing, _ := routing["rules"].([]any)
	var rules []any
	for _, pool := range pools {
		active := ActiveMember(pool, tunnels)
		if active == "" {
			continue
		}
		tag := PoolBalancerTag(pool.Name)
		balancers = append(balancers, map[string]any{
			"tag": tag, "selector": []string{PortalOutboundTag(active)}, "strategy": map[string]any{"type": "random"},
		})
		if len(pool.RouteInbounds) > 0 {
			rules = append(rules, map[string]any{"type": "field", "inboundTag": pool.RouteInbounds, "balancerTag": tag})
		}
	}
	if len(rules) == 0 && len(balancers) == 0 {
		return nil
	}
	routing["balancers"] = balancers
	routing["rules"] = append(rules, existing...)
	raw, err := json.Marshal(routing)
	if err != nil {
		return err
	}
	cfg.RouterConfig = json_util.RawMessage(raw)
	return nil
}
