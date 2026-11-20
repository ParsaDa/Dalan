// Package tunnelroute wires the tunnels of the BackPack and Rakhsh engines into the generated Xray config.
package tunnelroute

import (
	"encoding/json"
	"fmt"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/backpack"
	"github.com/mhsanaei/3x-ui/v3/internal/util/json_util"
	"github.com/mhsanaei/3x-ui/v3/internal/xray"
)

const defaultExitOutbound = "direct"

// PortalOutboundTag is the routable outbound a portal tunnel appears as; balancers and rules may target it.
func PortalOutboundTag(name string) string { return "tunnel-" + name }

func relayInboundTag(name string) string { return "tunnel-" + name + "-relay" }

// Apply adds the relay of every enabled tunnel to a generated config; the stored template is never touched.
func Apply(cfg *xray.Config, tunnels []model.ServerTunnel) error {
	var outbounds []any
	if len(cfg.OutboundConfigs) > 0 {
		if err := json.Unmarshal(cfg.OutboundConfigs, &outbounds); err != nil {
			return fmt.Errorf("tunnels: outbounds section is unparsable: %w", err)
		}
	}
	routing := map[string]any{}
	if len(cfg.RouterConfig) > 0 {
		if err := json.Unmarshal(cfg.RouterConfig, &routing); err != nil {
			return fmt.Errorf("tunnels: routing section is unparsable: %w", err)
		}
	}
	existing, _ := routing["rules"].([]any)
	var rules []any
	changed := false

	for _, t := range tunnels {
		if !t.Enable || t.ParentId > 0 || !model.ExternalEngine(t.Engine) || t.RelayPort == 0 {
			continue
		}
		changed = true
		switch t.Role {
		case model.TunnelRolePortal:
			outbounds = append(outbounds, relayOutbound(t))
			if len(t.RouteInbounds) > 0 {
				rules = append(rules, map[string]any{
					"type": "field", "inboundTag": t.RouteInbounds, "outboundTag": PortalOutboundTag(t.Name),
				})
			}
		case model.TunnelRoleBridge:
			cfg.InboundConfigs = append(cfg.InboundConfigs, relayInbound(t))
			exit := t.ExitOutbound
			if exit == "" {
				exit = defaultExitOutbound
			}
			rules = append(rules, map[string]any{
				"type": "field", "inboundTag": []string{relayInboundTag(t.Name)}, "outboundTag": exit,
			})
		default:
			return fmt.Errorf("tunnel %q: unknown role %q", t.Name, t.Role)
		}
	}
	// Re-marshalling reorders keys, so an untouched config must stay byte-identical for hot-apply diffs.
	if !changed {
		return nil
	}
	if hasEntry(tunnels) {
		if err := EnableOutboundStats(cfg); err != nil {
			return err
		}
	}

	// Tunnel rules go first so an operator's catch-all rule cannot swallow tunnel traffic.
	routing["rules"] = append(rules, existing...)
	newRouting, err := json.Marshal(routing)
	if err != nil {
		return err
	}
	newOutbounds, err := json.Marshal(outbounds)
	if err != nil {
		return err
	}
	cfg.RouterConfig = json_util.RawMessage(newRouting)
	cfg.OutboundConfigs = json_util.RawMessage(newOutbounds)
	return nil
}

// The entry end reaches the exit end's relay through the engine's loopback forward, so the relay is a plain SOCKS hop.
func relayOutbound(t model.ServerTunnel) map[string]any {
	user, pass := backpack.RelayCredentials(t)
	return map[string]any{
		"tag": PortalOutboundTag(t.Name), "protocol": "socks",
		"settings": map[string]any{"servers": []any{map[string]any{
			"address": "127.0.0.1", "port": t.RelayPort,
			"users": []any{map[string]any{"user": user, "pass": pass}},
		}}},
	}
}

func relayInbound(t model.ServerTunnel) xray.InboundConfig {
	user, pass := backpack.RelayCredentials(t)
	settings, _ := json.Marshal(map[string]any{
		"auth": "password", "udp": false,
		"accounts": []any{map[string]any{"user": user, "pass": pass}},
	})
	return xray.InboundConfig{
		Listen: json_util.RawMessage(`"127.0.0.1"`), Port: t.RelayPort, Protocol: "socks",
		Settings: settings, Tag: relayInboundTag(t.Name),
	}
}

func hasEntry(tunnels []model.ServerTunnel) bool {
	for _, t := range tunnels {
		if Routable(t) {
			return true
		}
	}
	return false
}

// EnableOutboundStats turns on the counters behind a tunnel's traffic totals; the stored template is untouched.
func EnableOutboundStats(cfg *xray.Config) error {
	policy := map[string]any{}
	if len(cfg.Policy) > 0 {
		if err := json.Unmarshal(cfg.Policy, &policy); err != nil {
			return fmt.Errorf("tunnels: policy section is unparsable: %w", err)
		}
	}
	system, _ := policy["system"].(map[string]any)
	if system == nil {
		system = map[string]any{}
	}
	if system["statsOutboundUplink"] == true && system["statsOutboundDownlink"] == true {
		return nil
	}
	system["statsOutboundUplink"], system["statsOutboundDownlink"] = true, true
	policy["system"] = system
	raw, err := json.Marshal(policy)
	if err != nil {
		return err
	}
	cfg.Policy = json_util.RawMessage(raw)
	return nil
}
