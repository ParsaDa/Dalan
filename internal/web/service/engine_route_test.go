package service

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"testing"
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

func TestAnUnknownWayOfDownloadingIsRefused(t *testing.T) {
	_, _, err := (&EngineRouteService{}).Client(EngineRoute{Via: "carrier-pigeon"}, 0)
	if err == nil || !errors.Is(err, errUnknownRoute) {
		t.Fatalf("err = %v, want errUnknownRoute", err)
	}
}

func TestChoicesListDirectAndTheOutboundsButNotInternalOnes(t *testing.T) {
	newTunnelDB(t)
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
	want := "panel::panel,direct::direct,outbound:tunnel-bp1:tunnel,outbound:tor-de:tor,outbound:warp:outbound"
	if strings.Join(flat, ",") != want {
		t.Fatalf("choices = %v\nwant     %s", flat, want)
	}
}

// recordingProxy is an HTTP proxy that notes where it was asked to go and then refuses.
func recordingProxy(t *testing.T) (*httptest.Server, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var hosts []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hosts = append(hosts, r.Host)
		mu.Unlock()
		http.Error(w, "refused by the test proxy", http.StatusBadGateway)
	}))
	t.Cleanup(server.Close)
	return server, func() []string { mu.Lock(); defer mu.Unlock(); return append([]string(nil), hosts...) }
}

func TestATunnelEngineIsDownloadedThroughTheRouteTheUserChose(t *testing.T) {
	svc := newTunnelDB(t)
	proxy, seen := recordingProxy(t)
	err := svc.InstallEngine("rakhsh", EngineRoute{Via: RouteLink, Value: proxy.URL})
	if err == nil {
		t.Fatal("the proxy refused every request, yet the install succeeded")
	}
	if hosts := seen(); len(hosts) == 0 || !strings.Contains(hosts[0], "github.com") {
		t.Fatalf("the proxy saw %v, want the GitHub release server", hosts)
	}
}

func TestTheTorBundleIsDownloadedThroughTheRouteTheUserChose(t *testing.T) {
	svc := newTorDB(t)
	useTorLayout(t)
	origin := torBundleServer(t)
	var mu sync.Mutex
	var viaProxy []string
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		viaProxy = append(viaProxy, r.Host)
		mu.Unlock()
		target, _ := url.Parse(origin.URL)
		httputil.NewSingleHostReverseProxy(target).ServeHTTP(w, r)
	}))
	t.Cleanup(proxy.Close)
	oldBase := torReleaseBase
	torReleaseBase = "http://tor-release.invalid"
	t.Cleanup(func() { torReleaseBase = oldBase })

	if err := svc.InstallEngine(EngineRoute{Via: RouteLink, Value: proxy.URL}); err != nil {
		t.Fatalf("InstallEngine through the proxy: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(viaProxy) == 0 || viaProxy[0] != "tor-release.invalid" {
		t.Fatalf("the proxy saw %v, want the Tor release host", viaProxy)
	}
}

func TestAnUnusableRouteStopsTheInstallBeforeAnythingIsDownloaded(t *testing.T) {
	svc := newTunnelDB(t)
	err := svc.InstallEngine("rakhsh", EngineRoute{Via: RouteLink, Value: "not a link"})
	if err == nil || !strings.Contains(err.Error(), "link") {
		t.Fatalf("err = %v, want a refusal that talks about the link", err)
	}
}
