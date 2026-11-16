// Package xrayreverse turns server tunnels into Xray's VLESS reverse proxy configuration.
package xrayreverse

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

func portalInboundTag(name string) string  { return "tunnel-" + name + "-link" }
func bridgeOutboundTag(name string) string { return "tunnel-" + name + "-dial" }
func bridgeInboundTag(name string) string  { return "tunnel-" + name + "-exit" }
func probeInboundTag(name string) string   { return "tunnel-" + name + "-probe" }
func manageInboundTag(name string) string  { return "tunnel-" + name + "-manage" }
func panelOutboundTag(name string) string  { return "tunnel-" + name + "-panel" }
func relayInboundTag(name string) string   { return "tunnel-" + name + "-relay" }

const probeBasePort = 62900

// ProbePort is the loopback SOCKS port routed into a portal tunnel so the panel can test it end to end.
func ProbePort(t model.ServerTunnel) int { return probeBasePort + t.Id%500 }

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
	riders := ridersByParent(tunnels)

	for _, t := range tunnels {
		if !t.Enable {
			continue
		}
		if t.ParentId > 0 {
			if model.ExternalEngine(t.Engine) || !hasRider(riders, t) {
				continue
			}
			changed = true
			cfg.InboundConfigs = append(cfg.InboundConfigs, manageInbound(t))
			rules = append(rules, map[string]any{
				"type": "field", "inboundTag": []string{manageInboundTag(t.Name)}, "outboundTag": PortalOutboundTag(t.Name),
			})
			continue
		}
		if model.ExternalEngine(t.Engine) {
			if t.RelayPort == 0 {
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
			continue
		}
		if t.Engine != "" && t.Engine != model.TunnelEngineXray {
			continue
		}
		changed = true
		switch t.Role {
		case model.TunnelRolePortal:
			inbound, err := portalInbound(t, riders[t.Id])
			if err != nil {
				return err
			}
			cfg.InboundConfigs = append(cfg.InboundConfigs, inbound)
			if t.ManagePort > 0 {
				cfg.InboundConfigs = append(cfg.InboundConfigs, manageInbound(t))
				rules = append(rules, map[string]any{
					"type": "field", "inboundTag": []string{manageInboundTag(t.Name)}, "outboundTag": PortalOutboundTag(t.Name),
				})
				continue
			}
			if probe, ok := probeInbound(cfg, t); ok {
				cfg.InboundConfigs = append(cfg.InboundConfigs, probe)
				rules = append(rules, map[string]any{
					"type": "field", "inboundTag": []string{probeInboundTag(t.Name)}, "outboundTag": PortalOutboundTag(t.Name),
				})
			}
			if len(t.RouteInbounds) > 0 {
				rules = append(rules, map[string]any{
					"type": "field", "inboundTag": t.RouteInbounds, "outboundTag": PortalOutboundTag(t.Name),
				})
			}
		case model.TunnelRoleBridge:
			outbounds = append(outbounds, bridgeOutbound(t))
			exit := t.ExitOutbound
			if t.ManagePort > 0 {
				outbounds = append(outbounds, panelOutbound(t))
				exit = panelOutboundTag(t.Name)
			} else if exit == "" {
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
	if hasEntry(tunnels) {
		if err := enableOutboundStats(cfg); err != nil {
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

// ridersByParent groups the management channels that share an enabled Xray tunnel's port.
func ridersByParent(tunnels []model.ServerTunnel) map[int][]model.ServerTunnel {
	parents := map[int]bool{}
	for _, t := range tunnels {
		if t.Enable && t.Role == model.TunnelRolePortal && !model.ExternalEngine(t.Engine) &&
			t.ParentId == 0 && t.ManagePort == 0 {
			parents[t.Id] = true
		}
	}
	riders := map[int][]model.ServerTunnel{}
	for _, t := range tunnels {
		if t.Enable && t.Role == model.TunnelRolePortal && t.ManagePort > 0 && parents[t.ParentId] {
			riders[t.ParentId] = append(riders[t.ParentId], t)
		}
	}
	return riders
}

func hasRider(riders map[int][]model.ServerTunnel, t model.ServerTunnel) bool {
	for _, r := range riders[t.ParentId] {
		if r.Id == t.Id {
			return true
		}
	}
	return false
}

func portalInbound(t model.ServerTunnel, riders []model.ServerTunnel) (xray.InboundConfig, error) {
	clients := []any{reverseClient(t)}
	for _, r := range riders {
		clients = append(clients, reverseClient(r))
	}
	settings, err := json.Marshal(map[string]any{"decryption": "none", "clients": clients})
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

func reverseClient(t model.ServerTunnel) map[string]any {
	return map[string]any{
		"id": t.UUID, "email": PortalOutboundTag(t.Name),
		"reverse": map[string]any{"tag": PortalOutboundTag(t.Name)},
	}
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

// The address is a placeholder: the bridge decides where management traffic lands, never the portal.
func manageInbound(t model.ServerTunnel) xray.InboundConfig {
	return xray.InboundConfig{
		Listen: json_util.RawMessage(`"127.0.0.1"`), Port: t.ManagePort, Protocol: "dokodemo-door",
		Settings: json_util.RawMessage(fmt.Sprintf(`{"address":"127.0.0.1","port":%d,"network":"tcp"}`, t.ManagePort)),
		Tag:      manageInboundTag(t.Name),
	}
}

// Redirect pins the destination to this server's panel, whatever the portal asked for.
func panelOutbound(t model.ServerTunnel) map[string]any {
	panel := fmt.Sprintf("127.0.0.1:%d", t.ManagePort)
	return map[string]any{
		"tag": panelOutboundTag(t.Name), "protocol": "freedom",
		"settings": map[string]any{
			"redirect": panel,
			"finalRules": []any{map[string]any{
				"action": "allow", "network": "tcp", "ip": []string{"127.0.0.1/32"}, "port": fmt.Sprint(t.ManagePort),
			}},
		},
	}
}

// The probe is skipped, not moved, when its port is taken: the panel must know the port without asking Xray.
func probeInbound(cfg *xray.Config, t model.ServerTunnel) (xray.InboundConfig, bool) {
	port := ProbePort(t)
	for i := range cfg.InboundConfigs {
		if cfg.InboundConfigs[i].Port == port {
			return xray.InboundConfig{}, false
		}
	}
	return xray.InboundConfig{
		Listen: json_util.RawMessage(`"127.0.0.1"`), Port: port, Protocol: "mixed",
		Settings: json_util.RawMessage(`{"auth":"noauth","udp":false}`), Tag: probeInboundTag(t.Name),
	}, true
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

// enableOutboundStats turns on the counters behind a tunnel's traffic totals; the stored template is untouched.
func enableOutboundStats(cfg *xray.Config) error {
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
