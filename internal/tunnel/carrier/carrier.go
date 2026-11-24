// Package carrier runs a tunnel on a kernel interface: GRE, IPIP or VXLAN between the two servers, with
// a TCP relay on the link's private addresses that carries the ports a tunnel forwards. It needs root and
// the ip command, so a tunnel on it is refused where they are missing.
package carrier

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// The kinds of interface a carrier can make.
const (
	GRE   = "gre"
	IPIP  = "ipip"
	VXLAN = "vxlan"
)

// Relay accepts connections on Listen and carries each to Dial.
type Relay struct {
	Listen string `json:"listen"`
	Dial   string `json:"dial"`
}

// Config is what the carrier process reads: the interface to make and the relays to run on it.
type Config struct {
	Kind      string `json:"kind"`
	Interface string `json:"interface"`
	// Local and Remote are the public IPv4 addresses of this end and of the other end.
	Local  string `json:"local"`
	Remote string `json:"remote"`
	// Address is this end's address on the link; the pair comes from Inner.
	Address string `json:"address"`
	// VNI and Port are the VXLAN network id and the UDP port it uses; the other kinds ignore them.
	VNI    int     `json:"vni,omitempty"`
	Port   int     `json:"port,omitempty"`
	Relays []Relay `json:"relays"`
}

var interfaceName = regexp.MustCompile(`^[a-z][a-z0-9]{0,14}$`)

// Validate refuses what would reach the ip command as anything but a plain value.
func (c Config) Validate() error {
	switch c.Kind {
	case GRE, IPIP, VXLAN:
	default:
		return fmt.Errorf("carrier kind %q is not gre, ipip or vxlan", c.Kind)
	}
	if !interfaceName.MatchString(c.Interface) {
		return fmt.Errorf("interface name %q is not valid", c.Interface)
	}
	for label, value := range map[string]string{"local": c.Local, "remote": c.Remote, "link": c.Address} {
		if addr, err := netip.ParseAddr(value); err != nil || !addr.Is4() {
			return fmt.Errorf("the %s address %q is not an IPv4 address", label, value)
		}
	}
	if c.Kind == VXLAN {
		if c.Port < 1 || c.Port > 65535 {
			return fmt.Errorf("vxlan port %d is out of range", c.Port)
		}
		if c.VNI < 1 || c.VNI >= 1<<24 {
			return fmt.Errorf("vxlan id %d is out of range", c.VNI)
		}
	}
	return nil
}

const linkPrefix = 30

// Inner gives the two ends of the tunnel on the relay port its link addresses: a /30 of 10.222.0.0/16 that no
// other relay port of a server shares, while the ports sit within 16384 of each other.
func Inner(relayPort int) (entry, exit string, err error) {
	if relayPort < 1 || relayPort > 65535 {
		return "", "", errors.New("a carrier tunnel needs a relay port")
	}
	block := relayPort % (1 << 14)
	third, fourth := block/64, (block%64)*4
	return fmt.Sprintf("10.222.%d.%d", third, fourth+1), fmt.Sprintf("10.222.%d.%d", third, fourth+2), nil
}

// Runner runs the ip command; tests replace it.
type Runner interface {
	Run(ctx context.Context, args ...string) ([]byte, error)
}

// IP runs the ip command of this server.
type IP struct{}

// Run runs ip with args and returns what it printed.
func (IP) Run(ctx context.Context, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, "ip", args...).CombinedOutput()
}

func mtuOf(kind string) int {
	switch kind {
	case GRE:
		return 1476
	case IPIP:
		return 1480
	}
	return 1450
}

func create(c Config) []string {
	switch c.Kind {
	case VXLAN:
		return []string{"link", "add", c.Interface, "type", "vxlan", "id", strconv.Itoa(c.VNI), "local", c.Local, "remote", c.Remote, "dstport", strconv.Itoa(c.Port)}
	default:
		return []string{"tunnel", "add", c.Interface, "mode", c.Kind, "local", c.Local, "remote", c.Remote, "ttl", "64"}
	}
}

// Up makes the interface and brings it up, first removing one a crashed process left. A step that fails
// takes the half-made interface away again.
func Up(ctx context.Context, r Runner, c Config) error {
	if err := c.Validate(); err != nil {
		return err
	}
	_, _ = r.Run(ctx, "link", "del", c.Interface)
	steps := [][]string{
		create(c),
		{"addr", "add", fmt.Sprintf("%s/%d", c.Address, linkPrefix), "dev", c.Interface},
		{"link", "set", c.Interface, "mtu", strconv.Itoa(mtuOf(c.Kind)), "up"},
	}
	for _, step := range steps {
		if out, err := r.Run(ctx, step...); err != nil {
			_ = Down(context.WithoutCancel(ctx), r, c)
			return fmt.Errorf("ip %s: %w: %s", strings.Join(step, " "), err, strings.TrimSpace(string(out)))
		}
	}
	return nil
}

// Down removes the interface.
func Down(ctx context.Context, r Runner, c Config) error {
	_, err := r.Run(ctx, "link", "del", c.Interface)
	return err
}

var errNotLinux = errors.New("kernel carriers run on Linux only")

// Probe says whether this server can make a carrier interface.
func Probe() error { return probe(goos, geteuid(), exec.LookPath) }

func probe(os string, euid int, lookPath func(string) (string, error)) error {
	if os != "linux" {
		return errNotLinux
	}
	if euid != 0 {
		return errors.New("kernel carriers need the panel to run as root")
	}
	if _, err := lookPath("ip"); err != nil {
		return errors.New("kernel carriers need the ip command (iproute2)")
	}
	return nil
}

// Load reads and validates the file the panel wrote for a carrier process.
func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err := json.Unmarshal(raw, &c); err != nil {
		return Config{}, fmt.Errorf("carrier file %s: %w", path, err)
	}
	return c, c.Validate()
}
