package service

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/xrayreverse"
)

type openedProxy struct {
	chosen map[string]any
	all    []any
	closed int
}

// stubEngineRouting replaces the temporary core and the running config's outbounds.
func stubEngineRouting(t *testing.T, outbounds []any) *openedProxy {
	t.Helper()
	opened := &openedProxy{}
	oldOpen, oldOutbounds := openEngineProxy, engineRouteOutbounds
	openEngineProxy = func(chosen map[string]any, all []any) (string, func(), error) {
		opened.chosen, opened.all = chosen, all
		return "127.0.0.1:5555", func() { opened.closed++ }, nil
	}
	engineRouteOutbounds = func() []any { return outbounds }
	t.Cleanup(func() { openEngineProxy, engineRouteOutbounds = oldOpen, oldOutbounds })
	return opened
}

func TestAProxyUrlIsUsedAsItIsWithoutStartingACore(t *testing.T) {
	opened := stubEngineRouting(t, nil)
	for _, link := range []string{"socks5://127.0.0.1:1080", "http://user:pw@198.51.100.4:3128"} {
		got, closeRoute, err := (&EngineRouteService{}).resolve(EngineRoute{Via: RouteLink, Value: link})
		if err != nil || got != link {
			t.Fatalf("resolve(%s) = %q, %v; want it unchanged", link, got, err)
		}
		closeRoute()
	}
	if opened.chosen != nil {
		t.Fatalf("a core was started for a plain proxy: %v", opened.chosen)
	}
}

func TestAShareLinkRunsATemporaryCoreAndCloseStopsIt(t *testing.T) {
	opened := stubEngineRouting(t, nil)
	link := "vless://11111111-1111-1111-1111-111111111111@203.0.113.7:443?security=none&type=tcp#x"
	got, closeRoute, err := (&EngineRouteService{}).resolve(EngineRoute{Via: RouteLink, Value: " " + link + "\n"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "socks5://127.0.0.1:5555" {
		t.Fatalf("proxy = %q, want the temporary core's SOCKS port", got)
	}
	if opened.chosen["protocol"] != "vless" || opened.chosen["tag"] != "engine-route" {
		t.Fatalf("the core was given %v, want the link's vless outbound tagged engine-route", opened.chosen)
	}
	closeRoute()
	if opened.closed != 1 {
		t.Fatalf("close ran %d times, want once", opened.closed)
	}
}

func TestALinkThatIsNoProxyAndNoShareLinkIsRefusedWithAReason(t *testing.T) {
	stubEngineRouting(t, nil)
	for _, bad := range []string{"", "ftp://example.com", "not a link"} {
		_, _, err := (&EngineRouteService{}).resolve(EngineRoute{Via: RouteLink, Value: bad})
		if err == nil || !strings.Contains(err.Error(), "link") {
			t.Fatalf("resolve(%q) err = %v, want a refusal that talks about the link", bad, err)
		}
	}
}

func TestAnOutboundOfThePanelIsReachedThroughATemporaryCoreThatKnowsTheOthersToo(t *testing.T) {
	torExit := map[string]any{"tag": "tor-de", "protocol": "socks", "settings": map[string]any{}}
	direct := map[string]any{"tag": "direct", "protocol": "freedom"}
	opened := stubEngineRouting(t, []any{direct, torExit})

	got, closeRoute, err := (&EngineRouteService{}).resolve(EngineRoute{Via: RouteOutbound, Value: "tor-de"})
	if err != nil || got != "socks5://127.0.0.1:5555" {
		t.Fatalf("resolve = %q, %v", got, err)
	}
	closeRoute()
	if opened.chosen["tag"] != "tor-de" || len(opened.all) != 2 {
		t.Fatalf("core got chosen=%v all=%d, want tor-de and the whole outbound list", opened.chosen, len(opened.all))
	}

	_, _, err = (&EngineRouteService{}).resolve(EngineRoute{Via: RouteOutbound, Value: "tor-xx"})
	if err == nil || !strings.Contains(err.Error(), "tor-xx") {
		t.Fatalf("an unknown outbound gave err = %v, want one that names it", err)
	}
}

func TestAnXrayTunnelIsReachedThroughItsProbePort(t *testing.T) {
	svc := newTunnelDB(t)
	stubEngineRouting(t, nil)
	portal, err := svc.CreatePortal(tunnel.PortalSpec{Name: "de1", Address: "203.0.113.10", Port: 8443, Security: model.TunnelSecurityNone}, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, closeRoute, err := (&EngineRouteService{}).resolve(EngineRoute{Via: RouteOutbound, Value: "tunnel-de1"})
	if err != nil {
		t.Fatal(err)
	}
	closeRoute()
	if want := fmt.Sprintf("socks5://127.0.0.1:%d", xrayreverse.ProbePort(*portal)); got != want {
		t.Fatalf("proxy = %q, want %q", got, want)
	}
}

func TestAnUnknownWayOfDownloadingIsRefused(t *testing.T) {
	_, _, err := (&EngineRouteService{}).Client(EngineRoute{Via: "carrier-pigeon"}, 0)
	if err == nil || !errors.Is(err, errUnknownRoute) {
		t.Fatalf("err = %v, want errUnknownRoute", err)
	}
}

func TestChoicesListDirectTheOutboundsAndTheXrayTunnelsButNotInternalOnes(t *testing.T) {
	svc := newTunnelDB(t)
	if _, err := svc.CreatePortal(tunnel.PortalSpec{Name: "de1", Address: "203.0.113.10", Port: 8443, Security: model.TunnelSecurityNone}, nil); err != nil {
		t.Fatal(err)
	}
	stubEngineRouting(t, []any{
		map[string]any{"tag": "direct", "protocol": "freedom"},
		map[string]any{"tag": "blocked", "protocol": "blackhole"},
		map[string]any{"tag": "tunnel-bp1", "protocol": "socks"},
		map[string]any{"tag": "tor-de", "protocol": "socks"},
		map[string]any{"tag": "warp", "protocol": "wireguard"},
	})
	got, err := (&EngineRouteService{}).Choices()
	if err != nil {
		t.Fatal(err)
	}
	var flat []string
	for _, c := range got {
		flat = append(flat, c.Via+":"+c.Value+":"+c.Kind)
	}
	want := "panel::panel,direct::direct,outbound:tunnel-bp1:tunnel,outbound:tor-de:tor,outbound:warp:outbound,outbound:tunnel-de1:tunnel"
	if strings.Join(flat, ",") != want {
		t.Fatalf("choices = %v\nwant     %s", flat, want)
	}
}
