package speedtest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// fakeServer answers like a speedtest.net host: latency.txt, a random*.jpg to download and upload.php.
type fakeServer struct {
	*httptest.Server
	delay    time.Duration
	pings    atomic.Int64
	uploaded atomic.Int64
}

func newFakeServer(t *testing.T, delay time.Duration) *fakeServer {
	t.Helper()
	f := &fakeServer{delay: delay}
	mux := http.NewServeMux()
	mux.HandleFunc("/speedtest/latency.txt", func(w http.ResponseWriter, r *http.Request) {
		f.pings.Add(1)
		time.Sleep(f.delay)
		fmt.Fprint(w, "test=test\n")
	})
	mux.HandleFunc("/speedtest/random4000x4000.jpg", func(w http.ResponseWriter, r *http.Request) {
		chunk := make([]byte, 64<<10)
		for {
			if _, err := w.Write(chunk); err != nil {
				return
			}
		}
	})
	mux.HandleFunc("/speedtest/upload.php", func(w http.ResponseWriter, r *http.Request) {
		n, _ := io.Copy(io.Discard, r.Body)
		f.uploaded.Add(n)
		fmt.Fprintf(w, "size=%d\n", n)
	})
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	return f
}

func (f *fakeServer) entry(id string) map[string]any {
	return map[string]any{
		"url": f.URL + "/speedtest/upload.php", "name": "Testville", "country": "Iran",
		"sponsor": "Test ISP " + id, "id": id, "host": strings.TrimPrefix(f.URL, "http://"), "distance": 12.5,
	}
}

func listServer(t *testing.T, status int, servers ...map[string]any) *httptest.Server {
	t.Helper()
	l := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("engine") != "js" {
			http.Error(w, "engine=js is how the list is asked for", http.StatusBadRequest)
			return
		}
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(servers)
	}))
	t.Cleanup(l.Close)
	return l
}

func fastConfig(list *httptest.Server) Config {
	return Config{
		ServersURL: list.URL, AllowPrivate: true,
		PingSamples: 4, Phase: 400 * time.Millisecond, Warmup: 0, Connections: 2, Candidates: 3,
	}
}

func TestRunMeasuresPingDownloadAndUploadAgainstTheFastestServer(t *testing.T) {
	slow := newFakeServer(t, 60*time.Millisecond)
	fast := newFakeServer(t, 0)
	list := listServer(t, http.StatusOK, slow.entry("1"), fast.entry("2"))

	var phases []Phase
	res, err := Run(context.Background(), fastConfig(list), func(u Update) {
		if len(phases) == 0 || phases[len(phases)-1] != u.Phase {
			phases = append(phases, u.Phase)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Server.ID != "2" {
		t.Fatalf("tested server %q, want the quicker one (2)", res.Server.ID)
	}
	if res.PingMs <= 0 || res.DownMbps <= 0 || res.UpMbps <= 0 {
		t.Fatalf("result %+v lacks a measurement", res)
	}
	if fast.uploaded.Load() == 0 {
		t.Fatal("the upload phase sent nothing to the chosen server")
	}
	if slow.uploaded.Load() != 0 {
		t.Fatal("the slower server was used for the speed phases")
	}
	want := []Phase{PhaseServers, PhasePing, PhaseDownload, PhaseUpload}
	if fmt.Sprint(phases) != fmt.Sprint(want) {
		t.Fatalf("phases = %v, want %v", phases, want)
	}
}

func TestRunNeverTouchesAServerOnAPrivateAddress(t *testing.T) {
	inside := newFakeServer(t, 0)
	list := listServer(t, http.StatusOK, inside.entry("9"))
	cfg := fastConfig(list)
	cfg.AllowPrivate = false

	_, err := Run(context.Background(), cfg, nil)
	if err == nil || !strings.Contains(err.Error(), "no usable speedtest server") {
		t.Fatalf("err = %v, want a refusal naming no usable server", err)
	}
	if inside.pings.Load() != 0 {
		t.Fatal("a loopback server listed by the directory was contacted")
	}
}

func TestRunSaysSpeedtestNetCouldNotBeReached(t *testing.T) {
	list := listServer(t, http.StatusServiceUnavailable)
	_, err := Run(context.Background(), fastConfig(list), nil)
	if err == nil || !strings.Contains(err.Error(), "could not get the server list from speedtest.net") {
		t.Fatalf("err = %v, want the server list failure", err)
	}
}

func TestRunStopsWhenItsContextIsCancelled(t *testing.T) {
	fast := newFakeServer(t, 0)
	list := listServer(t, http.StatusOK, fast.entry("2"))
	cfg := fastConfig(list)
	cfg.Phase = time.Minute
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(300 * time.Millisecond)
		cancel()
	}()

	started := time.Now()
	_, err := Run(ctx, cfg, nil)
	if err == nil {
		t.Fatal("a cancelled run reported success")
	}
	if took := time.Since(started); took > 3*time.Second {
		t.Fatalf("the run kept going %v after it was cancelled", took)
	}
}

func TestJitterIsTheMeanChangeBetweenNeighbouringPings(t *testing.T) {
	if got := jitter([]float64{10, 12, 9}); got != 2.5 {
		t.Fatalf("jitter = %v, want 2.5", got)
	}
	if got := jitter([]float64{7}); got != 0 {
		t.Fatalf("jitter of one sample = %v, want 0", got)
	}
}

func TestRunSendsEveryRequestThroughTheDialerItIsGiven(t *testing.T) {
	fast := newFakeServer(t, 0)
	list := listServer(t, http.StatusOK, fast.entry("2"))
	cfg := fastConfig(list)
	cfg.AllowPrivate = false
	var dials atomic.Int64
	var plain net.Dialer
	cfg.Dial = func(ctx context.Context, network, addr string) (net.Conn, error) {
		dials.Add(1)
		return plain.DialContext(ctx, network, addr)
	}

	res, err := Run(context.Background(), cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Server.ID != "2" {
		t.Fatalf("tested server %q, want 2", res.Server.ID)
	}
	// The list fetch, the probes and the transfers all open their connections through Dial.
	if dials.Load() < 3 {
		t.Fatalf("only %d connections went through the dialer", dials.Load())
	}
}
