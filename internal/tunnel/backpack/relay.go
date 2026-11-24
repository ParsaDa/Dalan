package backpack

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"net"
	"strconv"
	"time"

	"golang.org/x/net/proxy"

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

// ThroughputPort is the loopback port where the exit end of a tunnel answers its speed test; it sits just
// above the relay range.
func ThroughputPort(relayPort int) int { return relayPort + 1000 }

// RelayForward is the forwarded-port entry that exposes the relay on loopback at both ends.
func RelayForward(port int) string { return fmt.Sprintf("127.0.0.1:%d=127.0.0.1:%d", port, port) }

// RelayCredentials are the SOCKS account of the relay; both ends derive them from the tunnel token.
func RelayCredentials(t model.ServerTunnel) (user, pass string) {
	mac := hmac.New(sha256.New, []byte(t.UUID))
	mac.Write([]byte("dalan-relay"))
	return "dalan", hex.EncodeToString(mac.Sum(nil))[:32]
}

// RelayDial opens connections from this end of an entry tunnel, so they leave at the exit server.
func RelayDial(t model.ServerTunnel) func(ctx context.Context, network, addr string) (net.Conn, error) {
	user, pass := RelayCredentials(t)
	dialer, err := proxy.SOCKS5("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(t.RelayPort)),
		&proxy.Auth{User: user, Password: pass}, &net.Dialer{Timeout: 5 * time.Second})
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		if err != nil {
			return nil, err
		}
		if cd, ok := dialer.(proxy.ContextDialer); ok {
			return cd.DialContext(ctx, network, addr)
		}
		return dialer.Dial(network, addr)
	}
}
