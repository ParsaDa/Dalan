package forwarder

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"strconv"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/carrier"
)

const settingKindIP = "ip"

// The carrier makes a GRE, IPIP or VXLAN interface between the two servers and relays the tunnel's ports
// over its private addresses. It is the panel's own binary run as `x-ui carrier <file>`, so there is
// nothing to install; the end that dials is told the other end's public address by the dialerIp setting.
var carrierEngine = Engine{
	ID: model.TunnelEngineCarrier, Label: "Kernel carriers", Pinned: "built in", Self: true,
	Transports: []ForwardingTransport{
		{ID: carrier.GRE, Label: "GRE", Reverse: true, Direct: true},
		{ID: carrier.IPIP, Label: "IPIP", Reverse: true, Direct: true},
		{ID: carrier.VXLAN, Label: "VXLAN", Reverse: true, Direct: true},
	},
	Options: []ForwardingSetting{{
		Key: "dialerIp", Label: "Public IPv4 address of the end that dials (the exit in a reverse tunnel, the entry in a direct one)",
		Choices: []string{}, Kind: settingKindIP,
	}},
	Ext:       ".json",
	Args:      func(config string) []string { return []string{"carrier", config} },
	Render:    renderCarrier,
	Preflight: func() error { return CarrierProbe() },
}

// CarrierProbe is replaced by tests; it asks this server whether it can make a carrier interface.
var CarrierProbe = carrier.Probe

var errCarrierUDP = errors.New("kernel carriers carry TCP forwards only")

func parseIPv4(s string) (netip.Addr, error) {
	addr, err := netip.ParseAddr(s)
	if err != nil || !addr.Is4() {
		return netip.Addr{}, fmt.Errorf("%q is not an IPv4 address", s)
	}
	return addr, nil
}

func renderCarrier(p Plan) (string, error) {
	dialer, _ := p.Options["dialerIp"].(string)
	if dialer == "" {
		return "", errors.New("set the public IP address of the end that dials the carrier")
	}
	if _, err := parseIPv4(dialer); err != nil {
		return "", fmt.Errorf("the dialerIp setting: %w", err)
	}
	entry, exit, err := carrier.Inner(p.RelayPort)
	if err != nil {
		return "", err
	}
	cfg := carrier.Config{
		Kind: p.Transport, Interface: "dlc" + strconv.Itoa(p.RelayPort),
		VNI: p.RelayPort, Port: p.Port, Address: exit,
	}
	if p.Listens {
		cfg.Local, cfg.Remote = p.Host, dialer
	} else {
		cfg.Local, cfg.Remote = dialer, p.Host
	}
	peer := entry
	if p.Entry {
		cfg.Address, peer = entry, exit
	}
	for i, f := range p.Forwards {
		if f.UDP {
			return "", errCarrierUDP
		}
		linkPort := p.Port + i
		if linkPort > 65535 {
			return "", fmt.Errorf("port %d leaves no link port for forward %d", p.Port, i+1)
		}
		if p.Entry {
			cfg.Relays = append(cfg.Relays, carrier.Relay{Listen: f.Listen, Dial: hostPort(peer, linkPort)})
		} else {
			cfg.Relays = append(cfg.Relays, carrier.Relay{Listen: hostPort(cfg.Address, linkPort), Dial: f.Target})
		}
	}
	if err := cfg.Validate(); err != nil {
		return "", err
	}
	out, err := json.MarshalIndent(cfg, "", "  ")
	return string(out), err
}

func hostPort(host string, port int) string { return host + ":" + strconv.Itoa(port) }

// Preflight asks the engine of a tunnel whether this server can run it; an engine with nothing to check passes.
func Preflight(t model.ServerTunnel) error {
	if e, ok := ByID(t.Engine); ok && e.Preflight != nil {
		return e.Preflight()
	}
	return nil
}
