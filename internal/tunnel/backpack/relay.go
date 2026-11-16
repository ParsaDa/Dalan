package backpack

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

// The relay port range sits clear of the management (61000-61999) and probe (62900-63399) ports.
const (
	RelayPortMin = 63400
	RelayPortMax = 64399
)

// NewRelayPort picks a port in the relay range.
func NewRelayPort() (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(RelayPortMax-RelayPortMin+1))
	if err != nil {
		return 0, err
	}
	return RelayPortMin + int(n.Int64()), nil
}

// RelayForward is the forwarded-port entry that exposes the relay on loopback at both ends.
func RelayForward(port int) string { return fmt.Sprintf("127.0.0.1:%d=127.0.0.1:%d", port, port) }

// RelayCredentials are the SOCKS account of the relay; both ends derive them from the tunnel token.
func RelayCredentials(t model.ServerTunnel) (user, pass string) {
	mac := hmac.New(sha256.New, []byte(t.UUID))
	mac.Write([]byte("dalan-relay"))
	return "dalan", hex.EncodeToString(mac.Sum(nil))[:32]
}

// ManagementForward is the loopback forward a node's management channel adds to its BackPack tunnel.
func ManagementForward(managePort, nodePanelPort int) string {
	return fmt.Sprintf("127.0.0.1:%d=127.0.0.1:%d", managePort, nodePanelPort)
}

// WithManagement gives each BackPack tunnel the forwards of the node channels riding on it and drops the channels.
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
		if extra := forwards[t.Id]; len(extra) > 0 && t.Engine == model.TunnelEngineBackpack {
			t.ForwardPorts = append(append([]string(nil), t.ForwardPorts...), extra...)
		}
		out = append(out, t)
	}
	return out
}
