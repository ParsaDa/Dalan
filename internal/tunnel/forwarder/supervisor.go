package forwarder

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/engine"
)

// NewSupervisor runs one process of the engine per enabled tunnel that uses it, with state kept in dir.
func (e Engine) NewSupervisor(binary, dir string) *engine.Supervisor {
	return &engine.Supervisor{
		Name: e.Label, Binary: binary, Dir: dir, Ext: e.Ext,
		Wants: func(t model.ServerTunnel) bool { return t.Engine == e.ID },
		Render: func(t model.ServerTunnel) (string, error) {
			p, err := e.PlanFor(t, dir)
			if err != nil {
				return "", err
			}
			return e.Render(p)
		},
		Args:         e.Args,
		NotInstalled: errors.New(e.Label + " is not installed on this server"),
		Leftovers: func(dir, name string) []string {
			return []string{filepath.Join(dir, name+".crt"), filepath.Join(dir, name+".key")}
		},
	}
}

// socksGreeting offers password authentication, which is the only method the exit's relay inbound accepts.
var socksGreeting = []byte{5, 1, 2}

// RelayRTT sends a SOCKS5 greeting to the entry's relay port and reports how long the answer took. The
// answer comes from the exit's Xray, so it proves the whole path through the engine, not a local listener.
func RelayRTT(port int, timeout time.Duration) (time.Duration, bool) {
	start := time.Now()
	conn, err := (&net.Dialer{Timeout: timeout}).DialContext(context.Background(), "tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return 0, false
	}
	defer conn.Close()
	_ = conn.SetDeadline(start.Add(timeout))
	if _, err := conn.Write(socksGreeting); err != nil {
		return 0, false
	}
	reply := make([]byte, 2)
	if _, err := conn.Read(reply); err != nil || reply[0] != 5 {
		return 0, false
	}
	return max(time.Since(start), time.Microsecond), true
}

// Check says whether a tunnel end is up. The entry end proves it through its relay and reports the answer's
// delay; the exit end has no listener of its own to ask, so its running process is all there is to show.
func Check(t model.ServerTunnel, running func(name string) bool, timeout time.Duration) (time.Duration, bool) {
	if !running(t.Name) {
		return 0, false
	}
	if t.Role != model.TunnelRolePortal || t.RelayPort == 0 {
		return 0, true
	}
	return RelayRTT(t.RelayPort, timeout)
}

var errUnknownEngine = errors.New("unknown tunnel engine")

// Validate renders both ends of a tunnel the way the panels will run them, so a spec that cannot work is
// refused when it is made and not when an engine fails to start. Certificates go to a folder that is removed.
func Validate(t model.ServerTunnel) error {
	e, ok := ByID(t.Engine)
	if !ok {
		return fmt.Errorf("%w %q", errUnknownEngine, t.Engine)
	}
	dir, err := os.MkdirTemp("", "dalan-validate-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	for _, role := range []string{model.TunnelRolePortal, model.TunnelRoleBridge} {
		end := t
		end.Role = role
		p, err := e.PlanFor(end, dir)
		if err != nil {
			return err
		}
		if _, err := e.Render(p); err != nil {
			return err
		}
	}
	return nil
}
