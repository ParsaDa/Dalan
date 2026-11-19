package outbound

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/xray"
)

// listeningProcess binds the first inbound's port on Start, the way a real core would.
type listeningProcess struct {
	cfg      *xray.Config
	path     string
	listener net.Listener
	exitNow  bool
	stopped  bool
}

func (p *listeningProcess) Start() error {
	if p.exitNow {
		return nil
	}
	l, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", fmt.Sprintf("127.0.0.1:%d", p.cfg.InboundConfigs[0].Port))
	p.listener = l
	return err
}

func (p *listeningProcess) Stop() error {
	p.stopped = true
	if p.listener != nil {
		return p.listener.Close()
	}
	return nil
}
func (p *listeningProcess) IsRunning() bool   { return !p.exitNow && !p.stopped }
func (p *listeningProcess) GetResult() string { return "config rejected" }

func TestOpenProxyRunsOneOutboundBehindALoopbackSocksPortAndCloseStopsIt(t *testing.T) {
	var started *listeningProcess
	withStubProcess(t, func(cfg *xray.Config, path string) batchProcess {
		started = &listeningProcess{cfg: cfg, path: path}
		return started
	})
	outbound := map[string]any{"tag": "engine-route", "protocol": "vless", "settings": map[string]any{}}
	other := map[string]any{"tag": "chain-hop", "protocol": "freedom"}

	addr, closeProxy, err := (&OutboundService{}).OpenProxy(outbound, []any{other})
	if err != nil {
		t.Fatalf("OpenProxy: %v", err)
	}
	host, _, _ := net.SplitHostPort(addr)
	if host != "127.0.0.1" {
		t.Fatalf("address %q is not on loopback", addr)
	}
	if want := fmt.Sprintf("127.0.0.1:%d", started.cfg.InboundConfigs[0].Port); addr != want {
		t.Fatalf("address = %s, want the config's inbound %s", addr, want)
	}
	var tags []string
	var outs []map[string]any
	_ = json.Unmarshal(started.cfg.OutboundConfigs, &outs)
	for _, o := range outs {
		tags = append(tags, fmt.Sprint(o["tag"]))
	}
	if strings.Join(tags, ",") != "chain-hop,engine-route" {
		t.Fatalf("outbounds = %v, want the chain context and then the chosen one", tags)
	}
	if !strings.Contains(string(started.cfg.RouterConfig), `"outboundTag":"engine-route"`) {
		t.Fatalf("routing %s does not send the inbound to the chosen outbound", started.cfg.RouterConfig)
	}

	closeProxy()
	if !started.stopped {
		t.Fatal("close left the temporary core running")
	}
	if _, err := os.Stat(started.path); err == nil {
		t.Fatal("close left the temporary config file behind")
	}
}

func TestOpenProxyReportsACoreThatExitsAtOnce(t *testing.T) {
	withStubProcess(t, func(cfg *xray.Config, path string) batchProcess {
		return &listeningProcess{cfg: cfg, path: path, exitNow: true}
	})
	_, _, err := (&OutboundService{}).OpenProxy(map[string]any{"tag": "x", "protocol": "freedom"}, nil)
	if err == nil || !strings.Contains(err.Error(), "config rejected") {
		t.Fatalf("err = %v, want the core's own reason", err)
	}
}
