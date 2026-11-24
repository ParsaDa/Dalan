// Package forwarder runs server tunnels on engines that carry a loopback port from the entry end to the
// exit end: GOST, Rathole, Backhaul, sing-box, Xray, Kariz and Dagger. Each is one Engine in the registry.
package forwarder

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

// Transport is one way an engine can carry a tunnel; the tunnel's Transport field holds its id.
type Transport struct {
	ID    string
	Label string
	// Reverse and Direct say which directions the engine can run this transport in.
	Reverse, Direct bool
}

// Forward is one port the entry end opens and the exit end delivers to a target of its own.
type Forward struct {
	Listen, Target string
	UDP            bool
}

// Plan is one end of a tunnel as an engine sees it, before the engine writes its own file.
type Plan struct {
	Name string
	// Entry is the portal end, where users connect. Listens is the end that opens the control port; the
	// other end dials Host:Port.
	Entry, Listens bool
	Host           string
	Port           int
	Secret         string
	Transport      string
	// Forwards lists the ports the entry end opens and the exit end delivers, the Xray relay first.
	Forwards []Forward
	// Dir is where an engine may keep files of its own for this tunnel, such as a certificate.
	Dir     string
	Options map[string]any
}

// Engine describes one forwarding engine: how to install it, which transports it has and how to write
// the file for each end.
type Engine struct {
	ID, Label  string
	Repo       string
	Pinned     string
	Transports []Transport
	// Render writes the engine's file for one end; Ext is that file's suffix and Args its command line.
	Render func(p Plan) (string, error)
	Ext    string
	Args   func(config string) []string
	// Check is the command that validates a file without running it; empty when the engine has none.
	Check func(config string) []string
	asset func(version, arch string) (Asset, error)
	// Pins are the SHA-256 sums of the pinned release's files, which Dalan verified when it chose them.
	Pins map[string]string
}

var errNoTransport = errors.New("pick a transport for this engine")

// TransportOf finds a transport of the engine by id.
func (e Engine) TransportOf(id string) (Transport, error) {
	if id == "" {
		return Transport{}, errNoTransport
	}
	for _, t := range e.Transports {
		if t.ID == id {
			return t, nil
		}
	}
	return Transport{}, fmt.Errorf("%s has no transport %q", e.Label, id)
}

// Dials reports whether this end opens the connection to its peer: the exit in a reverse tunnel, the entry
// in a direct one.
func Dials(t model.ServerTunnel) bool {
	return (t.Mode == model.TunnelModeDirect) == (t.Role == model.TunnelRolePortal)
}

// RelayForward is the forward that exposes the Xray relay on loopback at both ends.
func RelayForward(port int) Forward {
	endpoint := "127.0.0.1:" + strconv.Itoa(port)
	return Forward{Listen: endpoint, Target: endpoint}
}

// PlanFor reads one end of a tunnel into the plan an engine renders; dir is the engine's state directory.
func (e Engine) PlanFor(t model.ServerTunnel, dir string) (Plan, error) {
	mode := t.Mode
	if mode == "" {
		mode = model.TunnelModeReverse
	}
	t.Mode = mode
	tr, err := e.TransportOf(t.Transport)
	if err != nil {
		return Plan{}, err
	}
	if mode == model.TunnelModeReverse && !tr.Reverse {
		return Plan{}, fmt.Errorf("%s cannot run %s in a reverse tunnel", e.Label, tr.Label)
	}
	if mode == model.TunnelModeDirect && !tr.Direct {
		return Plan{}, fmt.Errorf("%s cannot run %s in a direct tunnel", e.Label, tr.Label)
	}
	if t.EngineSecret == "" {
		return Plan{}, errors.New("the tunnel has no shared secret")
	}
	if t.Port < 1 || t.Port > 65535 {
		return Plan{}, fmt.Errorf("tunnel port %d is out of range", t.Port)
	}
	p := Plan{
		Name: t.Name, Entry: t.Role == model.TunnelRolePortal, Listens: !Dials(t),
		Host: t.Address, Port: t.Port, Secret: t.EngineSecret, Transport: t.Transport,
		Dir: dir, Options: t.Options,
	}
	if p.Forwards, err = forwardsOf(t); err != nil {
		return Plan{}, err
	}
	return p, nil
}

func forwardsOf(t model.ServerTunnel) ([]Forward, error) {
	var out []Forward
	if t.RelayPort > 0 {
		out = append(out, RelayForward(t.RelayPort))
	}
	for _, entry := range t.ForwardPorts {
		f, ok := ParseForward(entry)
		if !ok {
			return nil, fmt.Errorf("forwarded port %q is not valid", entry)
		}
		out = append(out, f)
	}
	return out, nil
}

// ParseForward reads "listen=target[/udp]" where each side is a port or host:port.
func ParseForward(entry string) (Forward, bool) {
	listen, target, ok := strings.Cut(entry, "=")
	if !ok {
		return Forward{}, false
	}
	target, udp := strings.CutSuffix(target, "/udp")
	listen, lok := endpoint(listen, "0.0.0.0")
	target, tok := endpoint(target, "127.0.0.1")
	if !lok || !tok {
		return Forward{}, false
	}
	return Forward{Listen: listen, Target: target, UDP: udp}, true
}

func endpoint(s, defaultHost string) (string, bool) {
	host, port := defaultHost, s
	if i := strings.LastIndex(s, ":"); i >= 0 {
		host, port = s[:i], s[i+1:]
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 || host == "" {
		return "", false
	}
	return host + ":" + strconv.Itoa(n), true
}

// Registry lists the engines in the order the panel offers them.
func Registry() []Engine {
	return slices.Clone(registry)
}

// ByID finds an engine of the registry.
func ByID(id string) (Engine, bool) {
	for _, e := range registry {
		if e.ID == id {
			return e, true
		}
	}
	return Engine{}, false
}

var registry = []Engine{ratholeEngine}
