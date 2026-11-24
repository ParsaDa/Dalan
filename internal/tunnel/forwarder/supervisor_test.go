package forwarder

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

func gostPortal() model.ServerTunnel {
	return model.ServerTunnel{
		Name: "de1", Role: model.TunnelRolePortal, Engine: model.TunnelEngineGost, Mode: model.TunnelModeReverse,
		Address: "203.0.113.10", Port: 8443, Transport: "wss", EngineSecret: "s3cret", RelayPort: 63500, Enable: true,
	}
}

func TestSupervisorRendersWhatThePlanRendersAndOnlyForItsOwnEngine(t *testing.T) {
	dir := t.TempDir()
	sup := gostEngine.NewSupervisor("/bin/gost", dir)
	tun := gostPortal()

	got, err := sup.Render(tun)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := gostEngine.PlanFor(tun, dir)
	if err != nil {
		t.Fatal(err)
	}
	want, err := gostEngine.Render(plan)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("supervisor rendered\n%s\nwant\n%s", got, want)
	}
	if !sup.Wants(tun) {
		t.Fatal("a GOST tunnel must be wanted by the GOST supervisor")
	}
	tun.Engine = model.TunnelEngineRathole
	if sup.Wants(tun) {
		t.Fatal("a Rathole tunnel must not be wanted by the GOST supervisor")
	}
}

func TestSupervisorCommandLineAndFileNameFollowTheEngine(t *testing.T) {
	sup := gostEngine.NewSupervisor("/bin/gost", t.TempDir())
	if sup.Ext != ".yml" || !reflect.DeepEqual(sup.Args("/x/de1.yml"), []string{"-C", "/x/de1.yml"}) {
		t.Fatalf("ext %q args %v", sup.Ext, sup.Args("/x/de1.yml"))
	}
	if sup.NotInstalled == nil {
		t.Fatal("a missing binary must be reported")
	}
}

func TestSupervisorCleansTheCertificateFilesAnEndWrote(t *testing.T) {
	dir := t.TempDir()
	sup := ratholeEngine.NewSupervisor("/bin/rathole", dir)
	got := sup.Leftovers(dir, "de1")
	want := []string{filepath.Join(dir, "de1.crt"), filepath.Join(dir, "de1.key")}
	if !slices.Equal(got, want) {
		t.Fatalf("leftovers %v, want %v", got, want)
	}
}

func listenSocks(t *testing.T, reply []byte) int {
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
			go func() {
				defer c.Close()
				buf := make([]byte, 3)
				if _, err := c.Read(buf); err == nil && reply != nil {
					_, _ = c.Write(reply)
				}
			}()
		}
	}()
	return l.Addr().(*net.TCPAddr).Port
}

func TestRelayRTTNeedsASocksServerBehindThePort(t *testing.T) {
	cases := []struct {
		name  string
		reply []byte
		want  bool
	}{
		{"socks5 answers", []byte{5, 2}, true},
		{"something else answers", []byte("HTTP/1.1 400"), false},
		{"connection closes without an answer", nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rtt, got := RelayRTT(listenSocks(t, c.reply), 500*time.Millisecond)
			if got != c.want || (got && rtt <= 0) || (!got && rtt != 0) {
				t.Fatalf("RelayRTT = %v, %v; want ok=%v and a delay only when ok", rtt, got, c.want)
			}
		})
	}
	if _, ok := RelayRTT(1, 200*time.Millisecond); ok {
		t.Fatal("nothing listens on port 1")
	}
}

func TestCheckOnTheEntryEndFollowsTheRelayAndOnTheExitEndTheProcess(t *testing.T) {
	port := listenSocks(t, []byte{5, 2})
	entry := gostPortal()
	entry.RelayPort = port
	exit := entry
	exit.Role = model.TunnelRoleBridge
	running := func(string) bool { return true }
	stopped := func(string) bool { return false }

	if rtt, ok := Check(entry, running, time.Second); !ok || rtt <= 0 {
		t.Fatalf("a running entry with a live relay: rtt %v ok %v", rtt, ok)
	}
	if _, ok := Check(exit, running, time.Second); !ok {
		t.Fatal("a running exit end is connected")
	}
	if _, ok := Check(entry, stopped, time.Second); ok {
		t.Fatal("a stopped entry is never connected")
	}
	if _, ok := Check(exit, stopped, time.Second); ok {
		t.Fatal("a stopped exit is never connected")
	}
	entry.RelayPort = 1
	if _, ok := Check(entry, running, 200*time.Millisecond); ok {
		t.Fatal("an entry end whose relay does not answer is not connected")
	}
}

func TestSupervisorReportsATunnelWithoutASecretInsteadOfRenderingOne(t *testing.T) {
	sup := gostEngine.NewSupervisor("/bin/gost", t.TempDir())
	tun := gostPortal()
	tun.EngineSecret = ""
	if _, err := sup.Render(tun); err == nil {
		t.Fatal("a tunnel with no shared secret must not render")
	}
}

func TestValidateRendersBothEndsAndNamesWhatCannotRun(t *testing.T) {
	good := gostPortal()
	good.ForwardPorts = []string{"2222=127.0.0.1:22"}
	if err := Validate(good); err != nil {
		t.Fatalf("a good tunnel failed: %v", err)
	}
	cases := []struct {
		name   string
		mutate func(*model.ServerTunnel)
		want   string
	}{
		{"unknown engine", func(t *model.ServerTunnel) { t.Engine = "nope" }, "engine"},
		{"unknown transport", func(t *model.ServerTunnel) { t.Transport = "carrier-pigeon" }, "transport"},
		{"direct on a reverse-only engine", func(t *model.ServerTunnel) {
			t.Engine, t.Transport, t.Mode = model.TunnelEngineRathole, "tcp", model.TunnelModeDirect
		}, "direct"},
		{"broken forward", func(t *model.ServerTunnel) { t.ForwardPorts = []string{"not a port"} }, "forwarded port"},
		{"no secret", func(t *model.ServerTunnel) { t.EngineSecret = "" }, "secret"},
		{"UDP forward on a TCP-only engine", func(t *model.ServerTunnel) { t.ForwardPorts = []string{"5353=127.0.0.1:53/udp"} }, "TCP"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tun := good
			c.mutate(&tun)
			if err := Validate(tun); err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("Validate error = %v, want one mentioning %q", err, c.want)
			}
		})
	}
}

func TestValidateLeavesNoCertificateBehind(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	if err := Validate(gostPortal()); err != nil {
		t.Fatal(err)
	}
	left, _ := os.ReadDir(tmp)
	if len(left) != 0 {
		t.Fatalf("validation left %d entries in the temp folder", len(left))
	}
}

func TestValidateRefusesATunnelWhoseExitEndCannotBeRendered(t *testing.T) {
	exitOnlyBroken := gostEngine
	exitOnlyBroken.ID = "exit-broken"
	exitOnlyBroken.Render = func(p Plan) (string, error) {
		if !p.Entry {
			return "", errors.New("the exit end cannot be written")
		}
		return "ok", nil
	}
	saved := registry
	registry = append(slices.Clone(registry), exitOnlyBroken)
	t.Cleanup(func() { registry = saved })

	tun := gostPortal()
	tun.Engine = exitOnlyBroken.ID
	if err := Validate(tun); err == nil || !strings.Contains(err.Error(), "exit end") {
		t.Fatalf("Validate error = %v, want the exit end's refusal", err)
	}
}

func TestOptionsAreCheckedAgainstTheEnginesCatalog(t *testing.T) {
	tun := gostPortal()
	tun.Engine, tun.Transport, tun.Mode = model.TunnelEngineKariz, "tcpmux", model.TunnelModeReverse
	for _, c := range []struct {
		name    string
		options map[string]any
		want    string
	}{
		{"unknown key", map[string]any{"turbo": true}, "turbo"},
		{"value outside the choices", map[string]any{"profile": "warp"}, "profile"},
		{"value of the wrong kind", map[string]any{"profile": 3}, "profile"},
	} {
		t.Run(c.name, func(t *testing.T) {
			bad := tun
			bad.Options = c.options
			if err := Validate(bad); err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("Validate error = %v, want one mentioning %q", err, c.want)
			}
		})
	}
	good := tun
	good.Options = map[string]any{"profile": "gaming"}
	if err := Validate(good); err != nil {
		t.Fatalf("a catalogued option failed: %v", err)
	}
	other := gostPortal()
	other.Options = map[string]any{"profile": "gaming"}
	if err := Validate(other); err == nil {
		t.Fatal("GOST has no options, so a profile must be refused")
	}
}
