package rakhsh

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"
	"net/netip"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

// A forward reads "<listen>=<target>" with an optional "/udp" suffix; a bare port means 0.0.0.0 or 127.0.0.1.
var forwardPattern = regexp.MustCompile(`^[0-9A-Za-z.:\[\]-]+=[0-9A-Za-z.:\[\]-]+(/udp)?$`)

// ControlSocket is where one tunnel's Rakhsh process serves `rakhsh status`.
func ControlSocket(dir, name string) string { return filepath.Join(dir, name+".rakhsh.sock") }

// RelayForward is the forward that exposes the Xray relay on loopback at both ends.
func RelayForward(port int) string { return fmt.Sprintf("127.0.0.1:%d=127.0.0.1:%d", port, port) }

// MirrorL3Addr gives the exit end its tunnel address: the entry's address plus one, inside the same network.
func MirrorL3Addr(addr string) (string, error) {
	host, prefix, hasPrefix := strings.Cut(addr, "/")
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return "", fmt.Errorf("layer-3 address %q is not an IP address", addr)
	}
	bits := 24 // the default prefix when none is given: /24 for IPv4, /64 for IPv6
	if ip.Is6() {
		bits = 64
	}
	if hasPrefix {
		if bits, err = strconv.Atoi(prefix); err != nil {
			return "", fmt.Errorf("layer-3 address %q has no valid prefix length", addr)
		}
	}
	network, err := ip.Prefix(bits)
	if err != nil {
		return "", fmt.Errorf("layer-3 address %q has no valid prefix length", addr)
	}
	next := ip.Next()
	if !next.IsValid() || !network.Contains(next) {
		return "", fmt.Errorf("no free address after %s in its network", host)
	}
	if hasPrefix {
		return next.String() + "/" + prefix, nil
	}
	return next.String(), nil
}

func tokenFor(uuid string) string {
	sum := sha256.Sum256([]byte("dalan-rakhsh:" + uuid))
	return base64.StdEncoding.EncodeToString(sum[:])
}

func checkKey(label, value string) error {
	if value == "" {
		return fmt.Errorf("the tunnel has no %s", label)
	}
	if raw, err := base64.StdEncoding.DecodeString(value); err != nil || len(raw) != 32 {
		return fmt.Errorf("the %s is not a 32-byte base64 value", label)
	}
	return nil
}

func splitForward(forward string) (listen, target string, udp, ok bool) {
	if !forwardPattern.MatchString(forward) {
		return "", "", false, false
	}
	listen, target, _ = strings.Cut(forward, "=")
	target, udp = strings.CutSuffix(target, "/udp")
	listen = withHost(listen, "0.0.0.0")
	target = withHost(target, "127.0.0.1")
	for _, endpoint := range []string{listen, target} {
		_, port, err := net.SplitHostPort(endpoint)
		if n, convErr := strconv.Atoi(port); err != nil || convErr != nil || n < 1 || n > 65535 {
			return "", "", false, false
		}
	}
	return listen, target, udp, true
}

func withHost(endpoint, host string) string {
	if _, err := strconv.Atoi(endpoint); err == nil {
		return net.JoinHostPort(host, endpoint)
	}
	return endpoint
}

func renderValue(o Option, value any) (string, error) {
	switch o.Kind {
	case KindBool:
		return strconv.FormatBool(value.(bool)), nil
	case KindInt:
		n, _ := wholeNumber(value)
		return strconv.FormatInt(n, 10), nil
	case KindChoice, KindString:
		return strconv.Quote(value.(string)), nil
	case KindList:
		items, _ := textItems(value)
		quoted := make([]string, len(items))
		for i, item := range items {
			quoted[i] = strconv.Quote(item)
		}
		return "[" + strings.Join(quoted, ", ") + "]", nil
	}
	return "", fmt.Errorf("option %q cannot be written as a single value", o.Key)
}

// dials reports whether this end opens the lanes: the exit in a reverse tunnel, the entry in a direct one.
func dials(t model.ServerTunnel) bool {
	return (t.Mode == model.TunnelModeDirect) == (t.Role == model.TunnelRolePortal)
}

// RenderConfig writes the Rakhsh configuration file for one end of a tunnel.
func RenderConfig(t model.ServerTunnel, controlSocket string) (string, error) {
	text, err := renderConfig(t, controlSocket)
	if err != nil {
		return "", fmt.Errorf("tunnel %q: %w", t.Name, err)
	}
	return text, nil
}

func renderConfig(t model.ServerTunnel, controlSocket string) (string, error) {
	mode := t.Mode
	if mode == "" {
		mode = model.TunnelModeReverse
	}
	t.Mode = mode
	if mode != model.TunnelModeReverse && mode != model.TunnelModeDirect {
		return "", fmt.Errorf("unknown mode %q", mode)
	}
	role := "entry"
	switch t.Role {
	case model.TunnelRolePortal:
	case model.TunnelRoleBridge:
		role = "exit"
	default:
		return "", fmt.Errorf("unknown role %q", t.Role)
	}
	if t.UUID == "" {
		return "", fmt.Errorf("the tunnel has no uuid to derive its token from")
	}
	if !slices.Contains(Carriers, t.Transport) {
		return "", fmt.Errorf("unknown carrier %q", t.Transport)
	}
	if err := checkKey("Rakhsh secret key", t.RakhshSecret); err != nil {
		return "", err
	}
	if err := checkKey("peer public key", t.RakhshPeerPublic); err != nil {
		return "", err
	}
	if err := ValidateOptions(mode, t.Options); err != nil {
		return "", err
	}
	if slices.Contains(RawCarriers, t.Transport) && !hasKey(t.Options, "peer_ip") {
		return "", fmt.Errorf("carrier %q needs the option peer_ip", t.Transport)
	}
	forwards, err := forwardsOf(t)
	if err != nil {
		return "", err
	}
	_, hasL3 := t.Options["l3_addr"]
	if t.Role == model.TunnelRolePortal && len(forwards) == 0 && !hasL3 {
		return "", fmt.Errorf("an entry tunnel needs at least one forwarded port")
	}
	dialling := dials(t)
	if dialling && t.Address == "" {
		return "", fmt.Errorf("the tunnel has no address to dial")
	}

	dial := "exit"
	if mode == model.TunnelModeDirect {
		dial = "entry"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "role = %s\ndial = %s\n", strconv.Quote(role), strconv.Quote(dial))
	if dialling {
		fmt.Fprintf(&b, "tunnel_connect = %s\n", strconv.Quote(net.JoinHostPort(t.Address, strconv.Itoa(t.Port))))
	} else {
		fmt.Fprintf(&b, "tunnel_listen = %s\n", strconv.Quote(net.JoinHostPort("0.0.0.0", strconv.Itoa(t.Port))))
	}
	fmt.Fprintf(&b, "carrier = %s\ntoken = %s\nstatic_secret = %s\npeer_public = %s\n",
		strconv.Quote(t.Transport), strconv.Quote(tokenFor(t.UUID)), strconv.Quote(t.RakhshSecret), strconv.Quote(t.RakhshPeerPublic))
	if controlSocket != "" {
		fmt.Fprintf(&b, "control_socket = %s\n", strconv.Quote(controlSocket))
	}
	if err := writeOptions(&b, t); err != nil {
		return "", err
	}
	if err := writeLinks(&b, t, dialling); err != nil {
		return "", err
	}
	for _, f := range forwards {
		fmt.Fprintf(&b, "\n[[forward]]\nlisten = %s\ntarget = %s\n", strconv.Quote(f.listen), strconv.Quote(f.target))
		if f.udp {
			b.WriteString("udp = true\n")
		}
	}
	return b.String(), nil
}

type forward struct {
	listen, target string
	udp            bool
}

// forwardsOf lists what the entry end accepts: the user's forwards, then the Xray relay.
func forwardsOf(t model.ServerTunnel) ([]forward, error) {
	if t.Role != model.TunnelRolePortal {
		return nil, nil
	}
	entries := slices.Clone(t.ForwardPorts)
	if t.RelayPort > 0 {
		entries = append(entries, RelayForward(t.RelayPort))
	}
	out := make([]forward, 0, len(entries))
	for _, entry := range entries {
		listen, target, udp, ok := splitForward(entry)
		if !ok {
			return nil, fmt.Errorf("forwarded port %q is not valid", entry)
		}
		out = append(out, forward{listen, target, udp})
	}
	return out, nil
}

// writeOptions writes the catalog's single-value options this end applies, in catalog order.
// mirroredPair is stored as the entry end sees it; the exit end reads each of the two as the other.
var mirroredPair = map[string]string{"spoof_src_ip": "spoof_peer_src", "spoof_peer_src": "spoof_src_ip"}

func writeOptions(b *strings.Builder, t model.ServerTunnel) error {
	for _, o := range Catalog {
		stored := o.Key
		if other, swapped := mirroredPair[o.Key]; swapped && t.Role == model.TunnelRoleBridge {
			stored = other
		}
		value, ok := t.Options[stored]
		if !ok || o.Kind == KindLinks || (o.Role == rolePortal && t.Role != model.TunnelRolePortal) {
			continue
		}
		// The stored peer_ip is the dialling end's address, which only the listening end needs.
		if dials(t) && o.Key == "spoof_peer_ip" {
			continue
		}
		if dials(t) && o.Key == "peer_ip" {
			value = t.Address
		}
		if o.Key == "l3_addr" && t.Role == model.TunnelRoleBridge {
			mirrored, err := MirrorL3Addr(value.(string))
			if err != nil {
				return fmt.Errorf("option %q: %w", o.Key, err)
			}
			value = mirrored
		}
		text, err := renderValue(o, value)
		if err != nil {
			return err
		}
		fmt.Fprintf(b, "%s = %s\n", o.Key, text)
	}
	return nil
}

// writeLinks writes one [[link]] per entry; a wildcard address becomes the peer's on the end that dials.
func writeLinks(b *strings.Builder, t model.ServerTunnel, dialling bool) error {
	entries, _ := t.Options["links"].([]any)
	for _, raw := range entries {
		entry := raw.(map[string]any)
		addr := entry["addr"].(string)
		if dialling {
			host, port, _ := net.SplitHostPort(addr)
			if ip := net.ParseIP(host); ip != nil && ip.IsUnspecified() {
				addr = net.JoinHostPort(t.Address, port)
			}
		}
		fmt.Fprintf(b, "\n[[link]]\ncarrier = %s\naddr = %s\n", strconv.Quote(entry["carrier"].(string)), strconv.Quote(addr))
		for _, o := range Catalog {
			value, ok := entry[o.Key]
			if !ok || !slices.Contains(linkOverrides, o.Key) {
				continue
			}
			text, err := renderValue(o, value)
			if err != nil {
				return err
			}
			fmt.Fprintf(b, "%s = %s\n", o.Key, text)
		}
	}
	return nil
}
