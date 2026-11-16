// Package tunnel creates the two ends of a server tunnel and the link that pairs them.
package tunnel

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math/big"
	"net"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/backpack"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/rakhsh"
)

const (
	setupLinkPrefix = "dalan://tunnel/"
	// Version 2 links describe BackPack tunnels; a panel that only knows version 1 refuses them by number.
	setupLinkXray     = 1
	setupLinkBackpack = 2
	// Version 3 carries BackPack options and the direct mode; a panel that stops at version 2 refuses it by number.
	setupLinkBackpackOptions = 3
	// Version 4 describes Rakhsh tunnels and carries the exit end's secret key.
	setupLinkRakhsh = 4
)

var errNotASetupLink = errors.New("not a Dalan tunnel link")

// Names become Xray tags, so they are kept to what a tag and a file name both accept.
var namePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,30}$`)

// PortalSpec is what an operator chooses when creating the listening end of a tunnel.
type PortalSpec struct {
	Name       string
	Address    string
	Port       int
	Security   string
	ServerName string

	// Engine is "xray" when empty. The fields below belong to the BackPack and Rakhsh engines;
	// for Rakhsh, Transport is the carrier.
	Engine             string
	Transport          string
	FallbackTransports []string
	ForwardPorts       []string
	AcceptUDP          bool
	// Mode is "reverse" when empty. A direct tunnel needs the BackPack engine and a carrier in Options.
	Mode    string
	Options map[string]any
}

type setupPayload struct {
	Version    int    `json:"v"`
	Name       string `json:"name"`
	Address    string `json:"address"`
	Port       int    `json:"port"`
	UUID       string `json:"uuid"`
	Security   string `json:"security"`
	ServerName string `json:"serverName,omitempty"`
	PublicKey  string `json:"publicKey,omitempty"`
	ShortId    string `json:"shortId,omitempty"`

	Transport          string         `json:"transport,omitempty"`
	FallbackTransports []string       `json:"fallbackTransports,omitempty"`
	RelayPort          int            `json:"relayPort,omitempty"`
	Mode               string         `json:"mode,omitempty"`
	Options            map[string]any `json:"options,omitempty"`

	// RakhshSecret is the exit end's secret key and RakhshPeer the entry's public key; the link is a secret.
	RakhshSecret string `json:"rakhshSecret,omitempty"`
	RakhshPeer   string `json:"rakhshPeer,omitempty"`
}

func validateEndpoint(name, address string, port int) error {
	switch {
	case !namePattern.MatchString(name):
		return fmt.Errorf("tunnel name %q must be lower-case letters, digits and dashes, at most 31 characters", name)
	case strings.TrimSpace(address) == "":
		return errors.New("tunnel address is required")
	case port < 1 || port > 65535:
		return fmt.Errorf("tunnel port %d is out of range", port)
	}
	return nil
}

func validateXray(security, serverName string) error {
	switch {
	case security != model.TunnelSecurityNone && security != model.TunnelSecurityReality:
		return fmt.Errorf("tunnel security %q is not supported", security)
	case security == model.TunnelSecurityReality && strings.TrimSpace(serverName) == "":
		return errors.New("a REALITY tunnel needs a server name")
	}
	return nil
}

func validateBackpackTransports(transport string, fallbacks []string) error {
	for _, name := range append([]string{transport}, fallbacks...) {
		known := false
		for _, t := range backpack.Transports {
			known = known || t == name
		}
		if !known {
			return fmt.Errorf("unknown BackPack transport %q", name)
		}
	}
	return nil
}

// addDirectAddresses gives a direct tunnel a /30 of its own in 10.10.0.0/16 unless the options already name one.
func addDirectAddresses(options map[string]any) error {
	if _, named := options["local_ip"]; named {
		return nil
	}
	n, err := rand.Int(rand.Reader, big.NewInt(250))
	if err != nil {
		return err
	}
	third := 1 + n.Int64()
	options["local_ip"] = fmt.Sprintf("10.10.%d.1/30", third)
	options["peer_ip"] = fmt.Sprintf("10.10.%d.2", third)
	return nil
}

// mirrorDirectAddresses turns the entry end's address pair into the exit end's: each takes the other's address.
func mirrorDirectAddresses(options map[string]any) (map[string]any, error) {
	local, _ := options["local_ip"].(string)
	peer, _ := options["peer_ip"].(string)
	host, prefix, hasPrefix := strings.Cut(local, "/")
	if net.ParseIP(host) == nil || net.ParseIP(peer) == nil || !hasPrefix {
		return nil, errors.New("tunnel link has no valid pair of tunnel addresses")
	}
	mirrored := maps.Clone(options)
	mirrored["local_ip"] = peer + "/" + prefix
	mirrored["peer_ip"] = host
	return mirrored, nil
}

// NewPortal validates a spec and generates the secrets for the listening end of a tunnel.
func NewPortal(spec PortalSpec) (model.ServerTunnel, error) {
	if err := validateEndpoint(spec.Name, spec.Address, spec.Port); err != nil {
		return model.ServerTunnel{}, err
	}
	portal := model.ServerTunnel{
		Name: spec.Name, Role: model.TunnelRolePortal, Engine: model.TunnelEngineXray, Enable: true,
		Address: strings.TrimSpace(spec.Address), Port: spec.Port, UUID: uuid.NewString(),
	}
	switch spec.Engine {
	case "", model.TunnelEngineXray:
		if err := validateXray(spec.Security, spec.ServerName); err != nil {
			return model.ServerTunnel{}, err
		}
		portal.Security = spec.Security
	case model.TunnelEngineBackpack:
		portal.Engine = model.TunnelEngineBackpack
		portal.Security = model.TunnelSecurityNone
		portal.ForwardPorts, portal.AcceptUDP = spec.ForwardPorts, spec.AcceptUDP
		portal.Mode = model.TunnelModeReverse
		portal.Options = maps.Clone(spec.Options)
		switch spec.Mode {
		case "", model.TunnelModeReverse:
			if err := validateBackpackTransports(spec.Transport, spec.FallbackTransports); err != nil {
				return model.ServerTunnel{}, err
			}
			portal.Transport, portal.FallbackTransports = spec.Transport, spec.FallbackTransports
			relay, err := backpack.NewRelayPort()
			if err != nil {
				return model.ServerTunnel{}, err
			}
			portal.RelayPort = relay
		case model.TunnelModeDirect:
			portal.Mode = model.TunnelModeDirect
			if portal.Options == nil {
				portal.Options = map[string]any{}
			}
			if err := addDirectAddresses(portal.Options); err != nil {
				return model.ServerTunnel{}, err
			}
		default:
			return model.ServerTunnel{}, fmt.Errorf("tunnel mode %q is not supported", spec.Mode)
		}
		// Rendering is the engine's own validation of the forwarded ports.
		if _, err := backpack.RenderConfig(portal); err != nil {
			return model.ServerTunnel{}, err
		}
		return portal, nil
	case model.TunnelEngineRakhsh:
		return newRakhshPortal(portal, spec)
	default:
		return model.ServerTunnel{}, fmt.Errorf("tunnel engine %q is not supported", spec.Engine)
	}
	if len(spec.Options) > 0 || spec.Mode == model.TunnelModeDirect {
		return model.ServerTunnel{}, errors.New("options and the direct mode belong to the BackPack engine")
	}
	if spec.Security == model.TunnelSecurityReality {
		key, err := ecdh.X25519().GenerateKey(rand.Reader)
		if err != nil {
			return model.ServerTunnel{}, err
		}
		shortID := make([]byte, 8)
		if _, err := rand.Read(shortID); err != nil {
			return model.ServerTunnel{}, err
		}
		portal.ServerName = strings.TrimSpace(spec.ServerName)
		portal.RealityPrivateKey = base64.RawURLEncoding.EncodeToString(key.Bytes())
		portal.RealityPublicKey = base64.RawURLEncoding.EncodeToString(key.PublicKey().Bytes())
		portal.ShortId = hex.EncodeToString(shortID)
	}
	return portal, nil
}

func newRakhshPortal(portal model.ServerTunnel, spec PortalSpec) (model.ServerTunnel, error) {
	if spec.Security == model.TunnelSecurityReality || spec.ServerName != "" {
		return model.ServerTunnel{}, errors.New("REALITY belongs to the Xray engine; Rakhsh sets its server name with the sni option")
	}
	portal.Engine, portal.Security = model.TunnelEngineRakhsh, model.TunnelSecurityNone
	portal.Transport = spec.Transport
	portal.ForwardPorts = spec.ForwardPorts
	portal.Options = maps.Clone(spec.Options)
	switch spec.Mode {
	case "", model.TunnelModeReverse:
		portal.Mode = model.TunnelModeReverse
	case model.TunnelModeDirect:
		portal.Mode = model.TunnelModeDirect
	default:
		return model.ServerTunnel{}, fmt.Errorf("tunnel mode %q is not supported", spec.Mode)
	}
	relay, err := backpack.NewRelayPort()
	if err != nil {
		return model.ServerTunnel{}, err
	}
	portal.RelayPort = relay
	secret, _, err := rakhsh.GenerateKeyPair()
	if err != nil {
		return model.ServerTunnel{}, err
	}
	bridgeSecret, err := rakhsh.BridgeSecret(secret, portal.UUID)
	if err != nil {
		return model.ServerTunnel{}, err
	}
	if portal.RakhshPeerPublic, err = rakhsh.PublicKey(bridgeSecret); err != nil {
		return model.ServerTunnel{}, err
	}
	portal.RakhshSecret = secret
	// Issuing and parsing the link renders both ends, which is the engine's own validation of the spec.
	link, err := SetupLink(portal)
	if err != nil {
		return model.ServerTunnel{}, err
	}
	if _, err := rakhsh.RenderConfig(portal, ""); err != nil {
		return model.ServerTunnel{}, err
	}
	if _, err := ParseSetupLink(link); err != nil {
		return model.ServerTunnel{}, err
	}
	return portal, nil
}

// SetupLink encodes what the other server needs to dial a portal; the private key never leaves the portal.
func SetupLink(portal model.ServerTunnel) (string, error) {
	if portal.Role != model.TunnelRolePortal {
		return "", errors.New("a setup link can only be made from the portal end of a tunnel")
	}
	payload := setupPayload{
		Version: setupLinkXray, Name: portal.Name, Address: portal.Address, Port: portal.Port,
		UUID: portal.UUID, Security: portal.Security, ServerName: portal.ServerName,
		PublicKey: portal.RealityPublicKey, ShortId: portal.ShortId,
	}
	if portal.Engine == model.TunnelEngineBackpack {
		payload.Version = setupLinkBackpack
		payload.Transport, payload.FallbackTransports = portal.Transport, portal.FallbackTransports
		payload.RelayPort = portal.RelayPort
		if portal.Mode == model.TunnelModeDirect || len(portal.Options) > 0 {
			payload.Version = setupLinkBackpackOptions
			payload.Mode, payload.Options = portal.Mode, portal.Options
		}
	}
	if portal.Engine == model.TunnelEngineRakhsh {
		bridgeSecret, err := rakhsh.BridgeSecret(portal.RakhshSecret, portal.UUID)
		if err != nil {
			return "", err
		}
		portalPublic, err := rakhsh.PublicKey(portal.RakhshSecret)
		if err != nil {
			return "", err
		}
		payload.Version = setupLinkRakhsh
		payload.Transport, payload.RelayPort = portal.Transport, portal.RelayPort
		payload.Mode, payload.Options = portal.Mode, rakhsh.SharedOptions(portal.Options)
		payload.RakhshSecret, payload.RakhshPeer = bridgeSecret, portalPublic
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return setupLinkPrefix + base64.RawURLEncoding.EncodeToString(raw), nil
}

// ParseSetupLink turns a portal's setup link into the dialing end of the same tunnel.
func ParseSetupLink(link string) (model.ServerTunnel, error) {
	encoded, ok := strings.CutPrefix(strings.TrimSpace(link), setupLinkPrefix)
	if !ok {
		return model.ServerTunnel{}, errNotASetupLink
	}
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return model.ServerTunnel{}, errNotASetupLink
	}
	var p setupPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return model.ServerTunnel{}, errNotASetupLink
	}
	if err := validateEndpoint(p.Name, p.Address, p.Port); err != nil {
		return model.ServerTunnel{}, err
	}
	if _, err := uuid.Parse(p.UUID); err != nil {
		return model.ServerTunnel{}, errors.New("tunnel link has no valid uuid")
	}
	bridge := model.ServerTunnel{
		Name: p.Name, Role: model.TunnelRoleBridge, Engine: model.TunnelEngineXray, Enable: true,
		Address: p.Address, Port: p.Port, UUID: p.UUID, Security: p.Security,
	}
	switch p.Version {
	case setupLinkXray:
		if err := validateXray(p.Security, p.ServerName); err != nil {
			return model.ServerTunnel{}, err
		}
		if p.Security == model.TunnelSecurityReality && (p.PublicKey == "" || p.ShortId == "") {
			return model.ServerTunnel{}, errors.New("tunnel link is missing its REALITY public key or short id")
		}
		bridge.ServerName, bridge.RealityPublicKey, bridge.ShortId = p.ServerName, p.PublicKey, p.ShortId
	case setupLinkBackpack:
		if err := validateBackpackTransports(p.Transport, p.FallbackTransports); err != nil {
			return model.ServerTunnel{}, err
		}
		bridge.Engine = model.TunnelEngineBackpack
		bridge.Security = model.TunnelSecurityNone
		bridge.Transport, bridge.FallbackTransports = p.Transport, p.FallbackTransports
		if p.RelayPort != 0 && (p.RelayPort < backpack.RelayPortMin || p.RelayPort > backpack.RelayPortMax) {
			return model.ServerTunnel{}, fmt.Errorf("tunnel link has a relay port outside %d-%d", backpack.RelayPortMin, backpack.RelayPortMax)
		}
		bridge.RelayPort = p.RelayPort
	case setupLinkBackpackOptions:
		bridge.Engine = model.TunnelEngineBackpack
		bridge.Security = model.TunnelSecurityNone
		bridge.Mode, bridge.Options = model.TunnelModeReverse, p.Options
		switch p.Mode {
		case "", model.TunnelModeReverse:
			if err := validateBackpackTransports(p.Transport, p.FallbackTransports); err != nil {
				return model.ServerTunnel{}, err
			}
			bridge.Transport, bridge.FallbackTransports = p.Transport, p.FallbackTransports
			if p.RelayPort != 0 && (p.RelayPort < backpack.RelayPortMin || p.RelayPort > backpack.RelayPortMax) {
				return model.ServerTunnel{}, fmt.Errorf("tunnel link has a relay port outside %d-%d", backpack.RelayPortMin, backpack.RelayPortMax)
			}
			bridge.RelayPort = p.RelayPort
		case model.TunnelModeDirect:
			mirrored, err := mirrorDirectAddresses(p.Options)
			if err != nil {
				return model.ServerTunnel{}, err
			}
			bridge.Mode, bridge.Options = model.TunnelModeDirect, mirrored
		default:
			return model.ServerTunnel{}, fmt.Errorf("tunnel link has the unknown mode %q", p.Mode)
		}
		// Rendering is the engine's own validation of the options.
		if _, err := backpack.RenderConfig(bridge); err != nil {
			return model.ServerTunnel{}, err
		}
	case setupLinkRakhsh:
		return parseRakhshLink(bridge, p)
	default:
		return model.ServerTunnel{}, fmt.Errorf("tunnel link is version %d; this panel understands versions %d to %d", p.Version, setupLinkXray, setupLinkRakhsh)
	}
	return bridge, nil
}

func parseRakhshLink(bridge model.ServerTunnel, p setupPayload) (model.ServerTunnel, error) {
	if p.PublicKey != "" || p.ShortId != "" || p.ServerName != "" {
		return model.ServerTunnel{}, errors.New("tunnel link mixes Xray fields into a Rakhsh tunnel")
	}
	if p.RelayPort != 0 && (p.RelayPort < backpack.RelayPortMin || p.RelayPort > backpack.RelayPortMax) {
		return model.ServerTunnel{}, fmt.Errorf("tunnel link has a relay port outside %d-%d", backpack.RelayPortMin, backpack.RelayPortMax)
	}
	if p.Mode != "" && p.Mode != model.TunnelModeReverse && p.Mode != model.TunnelModeDirect {
		return model.ServerTunnel{}, fmt.Errorf("tunnel link has the unknown mode %q", p.Mode)
	}
	bridge.Engine, bridge.Security = model.TunnelEngineRakhsh, model.TunnelSecurityNone
	bridge.Transport, bridge.RelayPort = p.Transport, p.RelayPort
	bridge.Mode, bridge.Options = model.TunnelModeReverse, p.Options
	if p.Mode == model.TunnelModeDirect {
		bridge.Mode = model.TunnelModeDirect
	}
	bridge.RakhshSecret, bridge.RakhshPeerPublic = p.RakhshSecret, p.RakhshPeer
	// Rendering is the engine's own validation of the keys, the carrier and the options.
	if _, err := rakhsh.RenderConfig(bridge, ""); err != nil {
		return model.ServerTunnel{}, err
	}
	return bridge, nil
}
