package carrier

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeRunner struct {
	mu    sync.Mutex
	calls [][]string
	fail  func(args []string) error
}

func (f *fakeRunner) Run(_ context.Context, args ...string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, args)
	if f.fail != nil {
		if err := f.fail(args); err != nil {
			return []byte("ip: " + err.Error()), err
		}
	}
	return nil, nil
}

func (f *fakeRunner) seen() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.calls))
	for i, c := range f.calls {
		out[i] = strings.Join(c, " ")
	}
	return out
}

func config(kind string) Config {
	return Config{
		Kind: kind, Interface: "dlc63421", Local: "203.0.113.10", Remote: "198.51.100.7",
		Address: "10.222.222.246", VNI: 63421, Port: 4789,
	}
}

func TestUpBuildsTheInterfaceTheKindNeeds(t *testing.T) {
	cases := []struct {
		kind string
		want []string
	}{
		{GRE, []string{
			"link del dlc63421",
			"tunnel add dlc63421 mode gre local 203.0.113.10 remote 198.51.100.7 ttl 64",
			"addr add 10.222.222.246/30 dev dlc63421",
			"link set dlc63421 mtu 1476 up",
		}},
		{IPIP, []string{
			"link del dlc63421",
			"tunnel add dlc63421 mode ipip local 203.0.113.10 remote 198.51.100.7 ttl 64",
			"addr add 10.222.222.246/30 dev dlc63421",
			"link set dlc63421 mtu 1480 up",
		}},
		{VXLAN, []string{
			"link del dlc63421",
			"link add dlc63421 type vxlan id 63421 local 203.0.113.10 remote 198.51.100.7 dstport 4789",
			"addr add 10.222.222.246/30 dev dlc63421",
			"link set dlc63421 mtu 1450 up",
		}},
	}
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			r := &fakeRunner{fail: func(args []string) error {
				if args[0] == "link" && args[1] == "del" {
					return errors.New("Cannot find device")
				}
				return nil
			}}
			if err := Up(context.Background(), r, config(tc.kind)); err != nil {
				t.Fatal(err)
			}
			if got := r.seen(); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("commands = %q\nwant      %q", got, tc.want)
			}
		})
	}
}

func TestUpRemovesTheInterfaceWhenALaterStepFails(t *testing.T) {
	r := &fakeRunner{fail: func(args []string) error {
		if args[0] == "addr" {
			return errors.New("RTNETLINK answers: File exists")
		}
		return nil
	}}
	err := Up(context.Background(), r, config(GRE))
	if err == nil || !strings.Contains(err.Error(), "File exists") {
		t.Fatalf("err = %v, want the ip error", err)
	}
	seen := r.seen()
	if last := seen[len(seen)-1]; last != "link del dlc63421" {
		t.Fatalf("last command = %q, want the half-made interface removed", last)
	}
}

func TestValidateRefusesValuesThatCouldReachTheIPCommandAsOptions(t *testing.T) {
	cases := map[string]func(*Config){
		"unknown kind":          func(c *Config) { c.Kind = "sit" },
		"option as interface":   func(c *Config) { c.Interface = "-j" },
		"interface too long":    func(c *Config) { c.Interface = "dlc1234567890123" },
		"hostname as local":     func(c *Config) { c.Local = "node.example.com" },
		"ipv6 remote":           func(c *Config) { c.Remote = "2001:db8::1" },
		"inner address garbage": func(c *Config) { c.Address = "10.222.0.86 dev lo" },
		"vxlan without a port":  func(c *Config) { c.Kind, c.Port = VXLAN, 0 },
		"vxlan id past 24 bits": func(c *Config) { c.Kind, c.VNI = VXLAN, 1<<24 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			c := config(GRE)
			mutate(&c)
			if err := c.Validate(); err == nil {
				t.Fatalf("Validate accepted %+v", c)
			}
		})
	}
	if err := config(VXLAN).Validate(); err != nil {
		t.Fatalf("a good vxlan config was refused: %v", err)
	}
}

func TestEachRelayPortGetsItsOwnInnerPair(t *testing.T) {
	entry, exit, err := Inner(63421)
	if err != nil {
		t.Fatal(err)
	}
	if entry != "10.222.222.245" || exit != "10.222.222.246" {
		t.Fatalf("Inner(63421) = %s, %s", entry, exit)
	}
	seen := map[string]int{}
	for port := 63400; port <= 64399; port++ {
		e, x, err := Inner(port)
		if err != nil {
			t.Fatal(err)
		}
		if other, dup := seen[e]; dup {
			t.Fatalf("relay ports %d and %d share the inner address %s", other, port, e)
		}
		seen[e], seen[x] = port, port
	}
	if _, _, err := Inner(0); err == nil {
		t.Fatal("a tunnel with no relay port has no inner pair")
	}
}

func echoServer(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			go func() { defer c.Close(); _, _ = io.Copy(c, c) }()
		}
	}()
	return l.Addr().String()
}

func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().String()
}

func TestServeCarriesBytesBothWaysAndStopsWithItsContext(t *testing.T) {
	target, listen := echoServer(t), freeAddr(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Serve(ctx, []Relay{{Listen: listen, Dial: target}}) }()

	var conn net.Conn
	var err error
	for range 50 {
		if conn, err = net.Dial("tcp", listen); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("the relay never started listening: %v", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := conn.Write([]byte("hello through the carrier")); err != nil {
		t.Fatal(err)
	}
	reply := make([]byte, len("hello through the carrier"))
	if _, err := io.ReadFull(conn, reply); err != nil || string(reply) != "hello through the carrier" {
		t.Fatalf("reply = %q, %v", reply, err)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve = %v after its context ended", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not stop")
	}
	if c, err := net.Dial("tcp", listen); err == nil {
		c.Close()
		t.Fatal("the relay still listens after it stopped")
	}
}

func TestServeFailsWhenAnAddressIsTaken(t *testing.T) {
	held, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()
	err = Serve(context.Background(), []Relay{{Listen: held.Addr().String(), Dial: "127.0.0.1:1"}})
	if err == nil {
		t.Fatal("Serve started on an address that is in use")
	}
}

func TestRunMakesTheInterfaceServesAndRemovesItOnCancel(t *testing.T) {
	c := config(GRE)
	c.Relays = []Relay{{Listen: freeAddr(t), Dial: echoServer(t)}}
	r := &fakeRunner{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Run(ctx, r, c) }()

	for range 100 {
		if len(r.seen()) >= 4 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Run = %v", err)
	}
	seen := r.seen()
	if len(seen) != 5 || seen[3] != "link set dlc63421 mtu 1476 up" || seen[4] != "link del dlc63421" {
		t.Fatalf("commands = %q, want the interface made first and removed last", seen)
	}
}

func TestProbeNamesWhatThisServerLacks(t *testing.T) {
	has := func(string) (string, error) { return "/usr/sbin/ip", nil }
	hasNot := func(string) (string, error) { return "", errors.New("not found") }
	cases := []struct {
		name string
		goos string
		euid int
		look func(string) (string, error)
		want string
	}{
		{"windows", "windows", -1, has, "Linux"},
		{"not root", "linux", 1000, has, "root"},
		{"no ip command", "linux", 0, hasNot, "ip"},
		{"fine", "linux", 0, has, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := probe(tc.goos, tc.euid, tc.look)
			switch {
			case tc.want == "" && err != nil:
				t.Fatalf("err = %v, want none", err)
			case tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)):
				t.Fatalf("err = %v, want it to mention %q", err, tc.want)
			}
		})
	}
}

func TestServePassesAClientsHalfCloseOnSoTheReplyEnds(t *testing.T) {
	target, listen := echoServer(t), freeAddr(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = Serve(ctx, []Relay{{Listen: listen, Dial: target}}) }()

	var conn net.Conn
	var err error
	for range 50 {
		if conn, err = net.Dial("tcp", listen); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := conn.Write([]byte("last words")); err != nil {
		t.Fatal(err)
	}
	if err := conn.(*net.TCPConn).CloseWrite(); err != nil {
		t.Fatal(err)
	}
	reply, err := io.ReadAll(conn)
	if err != nil || string(reply) != "last words" {
		t.Fatalf("reply = %q, %v, want the echo and then the end of the stream", reply, err)
	}
}

func TestLoadReadsAConfigFileAndRefusesAnInvalidOne(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.json")
	if err := os.WriteFile(good, []byte(`{"kind":"gre","interface":"dlc63500","local":"203.0.113.10","remote":"198.51.100.7","address":"10.222.224.49","relays":[{"listen":"127.0.0.1:63500","dial":"10.222.224.50:8443"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(good)
	if err != nil || c.Interface != "dlc63500" || len(c.Relays) != 1 || c.Relays[0].Dial != "10.222.224.50:8443" {
		t.Fatalf("Load = %+v, %v", c, err)
	}

	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte(`{"kind":"gre","interface":"-j","local":"203.0.113.10","remote":"198.51.100.7","address":"10.222.224.49"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(bad); err == nil || !strings.Contains(err.Error(), "interface") {
		t.Fatalf("Load(bad) = %v, want the interface refused before anything runs", err)
	}
	if _, err := Load(filepath.Join(dir, "missing.json")); err == nil {
		t.Fatal("a missing file was loaded")
	}
}
