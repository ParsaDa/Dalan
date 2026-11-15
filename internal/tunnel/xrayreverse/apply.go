// Package xrayreverse turns server tunnels into Xray's VLESS reverse proxy configuration.
package xrayreverse

import (
	"encoding/json"
	"fmt"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/util/json_util"
	"github.com/mhsanaei/3x-ui/v3/internal/xray"
)

const defaultExitOutbound = "direct"

// PortalOutboundTag is the routable outbound a portal tunnel appears as; balancers and rules may target it.
func PortalOutboundTag(name string) string { return "tunnel-" + name }

func portalInboundTag(name string) string  { return "tunnel-" + name + "-link" }
func bridgeOutboundTag(name string) string { return "tunnel-" + name + "-dial" }
func bridgeInboundTag(name string) string  { return "tunnel-" + name + "-exit" }

// Apply adds every enabled Xray tunnel to a generated config; the stored template is never touched.
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
		if !t.Enable || (t.Engine != "" && t.Engine != model.TunnelEngineXray) {
			continue
		}
		changed = true
		switch t.Role {
		case model.TunnelRolePortal:
			inbound, err := portalInbound(t)
			if err != nil {
				return err
			}
			cfg.InboundConfigs = append(cfg.InboundConfigs, inbound)
			if len(t.RouteInbounds) > 0 {
				rules = append(rules, map[string]any{
					"type": "field", "inboundTag": t.RouteInbounds, "outboundTag": PortalOutboundTag(t.Name),
				})
			}
		case model.TunnelRoleBridge:
			outbounds = append(outbounds, bridgeOutbound(t))
			exit := t.ExitOutbound
			if exit == "" {
				exit = defaultExitOutbound
			}
			rules = append(rules, map[string]any{
				"type": "field", "inboundTag": []string{bridgeInboundTag(t.Name)}, "outboundTag": exit,
			})
		default:
			return fmt.Errorf("tunnel %q: unknown role %q", t.Name, t.Role)
		}
	}
	// Re-marshalling reorders keys, so an untouched config must stay byte-identical for hot-apply diffs.
	if !changed {
		return nil
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

func portalInbound(t model.ServerTunnel) (xray.InboundConfig, error) {
	settings, err := json.Marshal(map[string]any{
		"decryption": "none",
		"clients": []any{map[string]any{
			"id": t.UUID, "email": PortalOutboundTag(t.Name),
			"reverse": map[string]any{"tag": PortalOutboundTag(t.Name)},
		}},
	})
	if err != nil {
		return xray.InboundConfig{}, err
	}
	stream := map[string]any{"network": "raw", "security": "none"}
	if t.Security == model.TunnelSecurityReality {
		stream["security"] = "reality"
		stream["realitySettings"] = map[string]any{
			"show": false, "target": t.ServerName + ":443", "xver": 0,
			"serverNames": []string{t.ServerName}, "privateKey": t.RealityPrivateKey,
			"shortIds": []string{t.ShortId},
		}
	}
	streamJSON, err := json.Marshal(stream)
	if err != nil {
		return xray.InboundConfig{}, err
	}
	return xray.InboundConfig{
		Listen: json_util.RawMessage(`"0.0.0.0"`), Port: t.Port, Protocol: "vless",
		Settings: settings, StreamSettings: streamJSON, Tag: portalInboundTag(t.Name),
	}, nil
}

func bridgeOutbound(t model.ServerTunnel) map[string]any {
	stream := map[string]any{"network": "raw", "security": "none"}
	if t.Security == model.TunnelSecurityReality {
		stream["security"] = "reality"
		stream["realitySettings"] = map[string]any{
			"serverName": t.ServerName, "fingerprint": "chrome",
			"publicKey": t.RealityPublicKey, "shortId": t.ShortId,
		}
	}
	return map[string]any{
		"tag": bridgeOutboundTag(t.Name), "protocol": "vless",
		"settings": map[string]any{
			"address": t.Address, "port": t.Port, "id": t.UUID, "encryption": "none",
			"reverse": map[string]any{"tag": bridgeInboundTag(t.Name)},
		},
		"streamSettings": stream,
	}
}
