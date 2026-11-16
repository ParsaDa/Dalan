// Package backpack runs server tunnels on the BackPack engine: it writes the engine's
// configuration, supervises one process per tunnel and reads the status it reports.
package backpack

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

// Transports are the carriers of BackPack's reverse engine, as its configuration spells them.
var Transports = []string{
	"tcp", "tcpmux", "stealth", "ws", "wss", "wsmux", "wssmux", "kcp", "quic", "udp", "xdi", "pck",
}

// A forwarded port is written into a quoted TOML string, so it may hold nothing that could end one.
var forwardPortPattern = regexp.MustCompile(`^[0-9A-Za-z.:\[\]=|-]+$`)

func knownTransport(name string) bool {
	for _, t := range Transports {
		if t == name {
			return true
		}
	}
	return false
}

func quoteList(items []string) string {
	quoted := make([]string, len(items))
	for i, item := range items {
		quoted[i] = strconv.Quote(item)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

// RenderConfig writes the BackPack configuration file for one end of a tunnel.
func RenderConfig(t model.ServerTunnel) (string, error) {
	if !knownTransport(t.Transport) {
		return "", fmt.Errorf("tunnel %q: unknown BackPack transport %q", t.Name, t.Transport)
	}
	for _, fallback := range t.FallbackTransports {
		if !knownTransport(fallback) {
			return "", fmt.Errorf("tunnel %q: unknown BackPack transport %q in the fallback list", t.Name, fallback)
		}
	}

	var b strings.Builder
	var ports []string
	switch t.Role {
	case model.TunnelRolePortal:
		ports = append(ports, t.ForwardPorts...)
		for _, port := range ports {
			if !forwardPortPattern.MatchString(port) {
				return "", fmt.Errorf("tunnel %q: forwarded port %q is not valid", t.Name, port)
			}
		}
		if t.RelayPort > 0 {
			ports = append(ports, RelayForward(t.RelayPort))
		}
		if len(ports) == 0 {
			return "", fmt.Errorf("tunnel %q: an entry tunnel needs at least one forwarded port", t.Name)
		}
		fmt.Fprintf(&b, "[server]\nbind_addr = %s\n", strconv.Quote(net.JoinHostPort("0.0.0.0", strconv.Itoa(t.Port))))
	case model.TunnelRoleBridge:
		fmt.Fprintf(&b, "[client]\nremote_addr = %s\n", strconv.Quote(net.JoinHostPort(t.Address, strconv.Itoa(t.Port))))
	default:
		return "", fmt.Errorf("tunnel %q: unknown role %q", t.Name, t.Role)
	}
	fmt.Fprintf(&b, "transport = %s\ntoken = %s\n", strconv.Quote(t.Transport), strconv.Quote(t.UUID))
	if t.Role == model.TunnelRolePortal {
		fmt.Fprintf(&b, "ports = %s\n", quoteList(ports))
		if t.AcceptUDP {
			b.WriteString("accept_udp = true\n")
		}
	}
	if len(t.FallbackTransports) > 0 {
		fmt.Fprintf(&b, "fallback_transports = %s\n", quoteList(t.FallbackTransports))
	}
	return b.String(), nil
}

// Snapshot is the part of the engine's periodic status file the panel shows.
type Snapshot struct {
	Connected bool   `json:"connected"`
	Peer      string `json:"peer"`
	BytesIn   uint64 `json:"bytesIn"`
	BytesOut  uint64 `json:"bytesOut"`
	Uptime    string `json:"uptime"`
}

func snapshotPath(dir, name string) string { return filepath.Join(dir, name+".metrics.json") }

// ReadSnapshot reads the status file the engine writes next to a tunnel's configuration every 30 seconds.
func ReadSnapshot(dir, name string) (Snapshot, error) {
	raw, err := os.ReadFile(snapshotPath(dir, name))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Snapshot{}, fmt.Errorf("tunnel %q has not reported its status yet", name)
		}
		return Snapshot{}, err
	}
	var file struct {
		Connected *bool  `json:"connected"`
		Peer      string `json:"peer"`
		BytesIn   uint64 `json:"bytes_in"`
		BytesOut  uint64 `json:"bytes_out"`
		Uptime    string `json:"uptime"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		return Snapshot{}, fmt.Errorf("tunnel %q: unreadable status file: %w", name, err)
	}
	snap := Snapshot{Peer: file.Peer, BytesIn: file.BytesIn, BytesOut: file.BytesOut, Uptime: file.Uptime}
	if file.Connected != nil {
		snap.Connected = *file.Connected
	} else {
		snap.Connected = file.Peer != ""
	}
	return snap, nil
}
