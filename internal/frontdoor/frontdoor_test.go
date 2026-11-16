package frontdoor

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

func inbound(port int, listen, stream string) model.Inbound {
	return model.Inbound{Enable: true, Port: port, Listen: listen, StreamSettings: stream}
}

func TestOnlyPlainWebSocketAndXhttpInboundsGetAPathRoute(t *testing.T) {
	ws := `{"network":"ws","security":"none","wsSettings":{"path":"/zedone"}}`
	xhttp := `{"network":"xhttp","security":"none","xhttpSettings":{"path":"/stream/"}}`
	upgrade := `{"network":"httpupgrade","security":"none","httpupgradeSettings":{"path":"/up"}}`
	disabled := inbound(20004, "", ws)
	disabled.Enable = false

	got := InboundRoutes([]model.Inbound{
		inbound(20001, "", ws),
		inbound(20002, "10.0.0.5", xhttp),
		inbound(20003, "0.0.0.0", upgrade),
		disabled,
		inbound(20005, "", `{"network":"ws","security":"tls","wsSettings":{"path":"/tls"}}`),
		inbound(20006, "", `{"network":"tcp","security":"none"}`),
		inbound(20007, "", `{"network":"ws","security":"none","wsSettings":{"path":"/"}}`),
		inbound(20008, "", `not json`),
	})

	want := []Route{
		{Prefix: "/zedone", Host: "127.0.0.1", Port: 20001},
		{Prefix: "/stream", Host: "10.0.0.5", Port: 20002},
		{Prefix: "/up", Host: "127.0.0.1", Port: 20003},
	}
	if len(got) != len(want) {
		t.Fatalf("routes = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("route %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestEachPathGoesToTheRightPortAndOnlyAtAPathBoundary(t *testing.T) {
	table := Table{
		Inbounds:  []Route{{Prefix: "/zed", Host: "127.0.0.1", Port: 1}, {Prefix: "/zed/deep", Host: "127.0.0.1", Port: 2}},
		SubPaths:  []string{"/sub/"},
		SubPort:   3,
		PanelPath: "/panel/",
		PanelPort: 4,
	}
	for path, want := range map[string]int{
		"/zed":              1,
		"/zed/session-id":   1,
		"/zed/deep/x":       2,
		"/zedone":           0,
		"/sub/abc":          3,
		"/panel/dashboard":  4,
		"/panel":            4,
		"/elsewhere":        0,
		"/":                 0,
		"/panelist/secrets": 0,
	} {
		_, port, ok := table.Resolve(path)
		if want == 0 && ok || want != 0 && (!ok || port != want) {
			t.Errorf("Resolve(%q) = %d, %v; want port %d", path, port, ok, want)
		}
	}
	root := Table{PanelPath: "/", PanelPort: 4}
	if _, port, ok := root.Resolve("/anything"); !ok || port != 4 {
		t.Errorf("a panel at the root should take every other path, got %d, %v", port, ok)
	}
}

func serve(t *testing.T, table Table) string {
	t.Helper()
	srv := httptest.NewServer(New(func() Table { return table }))
	t.Cleanup(srv.Close)
	return srv.URL
}

func portOf(t *testing.T, raw string) int {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	port, _ := strconv.Atoi(u.Port())
	return port
}

func TestTheRootAnswersHealthChecksWhenThePanelLivesUnderAPath(t *testing.T) {
	base := serve(t, Table{PanelPath: "/secret/", PanelPort: 1})
	for _, path := range []string{"/", "/healthz"} {
		resp, err := http.Get(base + path)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK || string(body) != "ok" {
			t.Errorf("GET %s = %d %q, want 200 ok", path, resp.StatusCode, body)
		}
	}
	resp, err := http.Get(base + "/unknown")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("GET /unknown = %d, want 404", resp.StatusCode)
	}
}

func TestHealthzAnswersEvenWhenThePanelOwnsTheRootPath(t *testing.T) {
	base := serve(t, Table{PanelPath: "/", PanelPort: 1})
	resp, err := http.Get(base + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /healthz = %d, want 200 from the front door itself", resp.StatusCode)
	}
}

func TestAResponseIsPassedOnAsItIsWrittenSoStreamingTransportsWork(t *testing.T) {
	release := make(chan struct{})
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A declared length makes the proxy buffer unless it is told to flush each write.
		w.Header().Set("Content-Length", "10")
		_, _ = io.WriteString(w, "first")
		w.(http.Flusher).Flush()
		<-release
	}))
	t.Cleanup(backend.Close)
	base := serve(t, Table{Inbounds: []Route{{Prefix: "/x", Host: "127.0.0.1", Port: portOf(t, backend.URL)}}})
	// Registered last so it runs first: the servers cannot close while a request is still waiting on it.
	t.Cleanup(func() { close(release) })

	got := make(chan string, 1)
	go func() {
		resp, err := http.Get(base + "/x/session")
		if err != nil {
			got <- err.Error()
			return
		}
		defer resp.Body.Close()
		buf := make([]byte, 5)
		_, _ = io.ReadFull(resp.Body, buf)
		got <- string(buf)
	}()
	select {
	case s := <-got:
		if s != "first" {
			t.Fatalf("first chunk = %q", s)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the first chunk was held back until the backend finished")
	}
}

func TestAnUpgradedConnectionCarriesBytesBothWays(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, rw, err := w.(http.Hijacker).Hijack()
		if err != nil {
			return
		}
		defer conn.Close()
		_, _ = rw.WriteString("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n")
		_ = rw.Flush()
		_, _ = io.Copy(conn, rw)
	}))
	t.Cleanup(backend.Close)
	base := serve(t, Table{Inbounds: []Route{{Prefix: "/ws", Host: "127.0.0.1", Port: portOf(t, backend.URL)}}})

	conn, err := net.Dial("tcp", strings.TrimPrefix(base, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	_, _ = io.WriteString(conn, "GET /ws HTTP/1.1\r\nHost: x\r\nConnection: Upgrade\r\nUpgrade: websocket\r\n\r\n")
	r := bufio.NewReader(conn)
	status, err := r.ReadString('\n')
	if err != nil || !strings.Contains(status, "101") {
		t.Fatalf("status line = %q (%v), want 101", status, err)
	}
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if line == "\r\n" {
			break
		}
	}
	_, _ = io.WriteString(conn, "ping")
	buf := make([]byte, 4)
	if _, err := io.ReadFull(r, buf); err != nil || string(buf) != "ping" {
		t.Fatalf("echo = %q (%v), want ping", buf, err)
	}
}
