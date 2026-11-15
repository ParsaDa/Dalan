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
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

const (
	setupLinkPrefix  = "dalan://tunnel/"
	setupLinkVersion = 1
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
}

func validate(name, address string, port int, security, serverName string) error {
	switch {
	case !namePattern.MatchString(name):
		return fmt.Errorf("tunnel name %q must be lower-case letters, digits and dashes, at most 31 characters", name)
	case strings.TrimSpace(address) == "":
		return errors.New("tunnel address is required")
	case port < 1 || port > 65535:
		return fmt.Errorf("tunnel port %d is out of range", port)
	case security != model.TunnelSecurityNone && security != model.TunnelSecurityReality:
		return fmt.Errorf("tunnel security %q is not supported", security)
	case security == model.TunnelSecurityReality && strings.TrimSpace(serverName) == "":
		return errors.New("a REALITY tunnel needs a server name")
	}
	return nil
}

// NewPortal validates a spec and generates the secrets for the listening end of a tunnel.
func NewPortal(spec PortalSpec) (model.ServerTunnel, error) {
	if err := validate(spec.Name, spec.Address, spec.Port, spec.Security, spec.ServerName); err != nil {
		return model.ServerTunnel{}, err
	}
	portal := model.ServerTunnel{
		Name: spec.Name, Role: model.TunnelRolePortal, Engine: model.TunnelEngineXray, Enable: true,
		Address: strings.TrimSpace(spec.Address), Port: spec.Port, UUID: uuid.NewString(), Security: spec.Security,
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

// SetupLink encodes what the other server needs to dial a portal; the private key never leaves the portal.
func SetupLink(portal model.ServerTunnel) (string, error) {
	if portal.Role != model.TunnelRolePortal {
		return "", errors.New("a setup link can only be made from the portal end of a tunnel")
	}
	raw, err := json.Marshal(setupPayload{
		Version: setupLinkVersion, Name: portal.Name, Address: portal.Address, Port: portal.Port,
		UUID: portal.UUID, Security: portal.Security, ServerName: portal.ServerName,
		PublicKey: portal.RealityPublicKey, ShortId: portal.ShortId,
	})
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
	if p.Version != setupLinkVersion {
		return model.ServerTunnel{}, fmt.Errorf("tunnel link is version %d; this panel understands version %d", p.Version, setupLinkVersion)
	}
	if err := validate(p.Name, p.Address, p.Port, p.Security, p.ServerName); err != nil {
		return model.ServerTunnel{}, err
	}
	if _, err := uuid.Parse(p.UUID); err != nil {
		return model.ServerTunnel{}, errors.New("tunnel link has no valid uuid")
	}
	if p.Security == model.TunnelSecurityReality && (p.PublicKey == "" || p.ShortId == "") {
		return model.ServerTunnel{}, errors.New("tunnel link is missing its REALITY public key or short id")
	}
	return model.ServerTunnel{
		Name: p.Name, Role: model.TunnelRoleBridge, Engine: model.TunnelEngineXray, Enable: true,
		Address: p.Address, Port: p.Port, UUID: p.UUID, Security: p.Security,
		ServerName: p.ServerName, RealityPublicKey: p.PublicKey, ShortId: p.ShortId,
	}, nil
}
