// Package speedtest measures this machine's ping, download and upload against a speedtest.net server,
// or against the exit end of a tunnel.
package speedtest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"path"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

const (
	defaultServersURL = "https://www.speedtest.net/api/js/servers"
	downloadFile      = "random4000x4000.jpg"
	uploadChunk       = 64 << 10
	uploadBody        = 8 << 20
	tick              = 200 * time.Millisecond
	probeTimeout      = 4 * time.Second
	userAgent         = "Mozilla/5.0 (compatible; dalan-speedtest)"
)

// Phase names the stage a run is in.
type Phase string

const (
	PhaseServers  Phase = "servers"
	PhasePing     Phase = "ping"
	PhaseDownload Phase = "download"
	PhaseUpload   Phase = "upload"
)

// Config tunes a run; DefaultConfig holds the production values.
type Config struct {
	ServersURL   string
	AllowPrivate bool
	PingSamples  int
	Phase        time.Duration
	Warmup       time.Duration
	Connections  int
	Candidates   int
	// Dial, when set, carries every connection (for example through a tunnel); the far end then refuses internal targets.
	Dial func(ctx context.Context, network, addr string) (net.Conn, error)
	// LinkAddr, when set, measures the link to the responder at this address (dialed through Dial) and
	// leaves speedtest.net out, so the result is the speed of the path to that end.
	LinkAddr string
}

// DefaultConfig is what the panel runs with.
func DefaultConfig() Config {
	return Config{
		ServersURL: defaultServersURL, PingSamples: 8, Phase: 10 * time.Second,
		Warmup: 2 * time.Second, Connections: 4, Candidates: 5,
	}
}

// ServerInfo is the speedtest.net host a run measured against.
type ServerInfo struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Sponsor  string  `json:"sponsor"`
	Country  string  `json:"country"`
	Host     string  `json:"host"`
	Distance float64 `json:"distance"`

	url string
}

// Update is a progress report: Mbps is the live rate while a speed phase runs.
type Update struct {
	Phase    Phase
	Progress float64
	Mbps     float64
	PingMs   float64
}

// Result is one finished run.
type Result struct {
	Server   ServerInfo `json:"server"`
	PingMs   float64    `json:"pingMs"`
	JitterMs float64    `json:"jitterMs"`
	DownMbps float64    `json:"downMbps"`
	UpMbps   float64    `json:"upMbps"`
	At       int64      `json:"at"`
}

var errNoServer = errors.New("no usable speedtest server")

// Run measures against the closest responsive server; onUpdate may be nil.
func Run(ctx context.Context, cfg Config, onUpdate func(Update)) (Result, error) {
	cfg = normalize(cfg)
	emit := func(u Update) {
		if onUpdate != nil {
			onUpdate(u)
		}
	}
	if cfg.LinkAddr != "" {
		return runLink(ctx, cfg, emit)
	}
	client := newClient(cfg, cfg.AllowPrivate)
	defer client.CloseIdleConnections()
	directory := newClient(cfg, true)
	defer directory.CloseIdleConnections()

	emit(Update{Phase: PhaseServers})
	servers, err := fetchServers(ctx, directory, cfg)
	if err != nil {
		return Result{}, fmt.Errorf("could not get the server list from speedtest.net: %w", err)
	}
	best, err := pickServer(ctx, client, servers)
	if err != nil {
		return Result{}, err
	}

	res := Result{Server: best}
	emit(Update{Phase: PhasePing})
	samples, err := pingSamples(ctx, client, best, cfg.PingSamples, func(ms float64) {
		emit(Update{Phase: PhasePing, PingMs: ms})
	})
	if err != nil {
		return Result{}, err
	}
	res.PingMs = minOf(samples)
	res.JitterMs = jitter(samples)

	emit(Update{Phase: PhaseDownload})
	if res.DownMbps, err = measure(ctx, cfg, PhaseDownload, downloadWorker(client, best), emit); err != nil {
		return Result{}, err
	}
	emit(Update{Phase: PhaseUpload})
	if res.UpMbps, err = measure(ctx, cfg, PhaseUpload, uploadWorker(client, best), emit); err != nil {
		return Result{}, err
	}
	res.At = time.Now().UnixMilli()
	return res, nil
}

func normalize(cfg Config) Config {
	if cfg.ServersURL == "" {
		cfg.ServersURL = defaultServersURL
	}
	if cfg.PingSamples < 1 {
		cfg.PingSamples = 1
	}
	if cfg.Connections < 1 {
		cfg.Connections = 1
	}
	if cfg.Candidates < 1 {
		cfg.Candidates = 1
	}
	if cfg.Phase <= 0 {
		cfg.Phase = 10 * time.Second
	}
	if cfg.Warmup < 0 || cfg.Warmup >= cfg.Phase {
		cfg.Warmup = cfg.Phase / 4
	}
	return cfg
}

// newClient guards test-server connections so a listed server cannot point the panel at an internal address.
func newClient(cfg Config, allowPrivate bool) *http.Client {
	dialer := &net.Dialer{Timeout: probeTimeout}
	if !allowPrivate && cfg.Dial == nil {
		dialer.Control = func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			ip, err := netip.ParseAddr(host)
			if err != nil {
				return err
			}
			if isInternal(ip.Unmap()) {
				return fmt.Errorf("refusing to connect to internal address %s", ip)
			}
			return nil
		}
	}
	dial := dialer.DialContext
	if cfg.Dial != nil {
		dial = cfg.Dial
	}
	return &http.Client{Transport: &http.Transport{
		DialContext: dial, DisableCompression: true,
		MaxIdleConnsPerHost: 16, TLSHandshakeTimeout: probeTimeout,
	}}
}

func isInternal(ip netip.Addr) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsUnspecified()
}

type listEntry struct {
	URL      string  `json:"url"`
	Name     string  `json:"name"`
	Country  string  `json:"country"`
	Sponsor  string  `json:"sponsor"`
	ID       string  `json:"id"`
	Host     string  `json:"host"`
	Distance float64 `json:"distance"`
}

func fetchServers(ctx context.Context, client *http.Client, cfg Config) ([]ServerInfo, error) {
	u, err := url.Parse(cfg.ServersURL)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("engine", "js")
	q.Set("limit", strconv.Itoa(cfg.Candidates*4))
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	var entries []listEntry
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&entries); err != nil {
		return nil, err
	}
	servers := make([]ServerInfo, 0, len(entries))
	for _, e := range entries {
		su, err := url.Parse(e.URL)
		if err != nil || (su.Scheme != "http" && su.Scheme != "https") || su.Host == "" {
			continue
		}
		servers = append(servers, ServerInfo{
			ID: e.ID, Name: e.Name, Sponsor: e.Sponsor, Country: e.Country,
			Host: e.Host, Distance: e.Distance, url: e.URL,
		})
	}
	return servers, nil
}

// fileURL points at a file served next to the upload endpoint, with a cache buster.
func (s ServerInfo) fileURL(name string) string {
	u, err := url.Parse(s.url)
	if err != nil {
		return ""
	}
	u.Path = path.Join(path.Dir(u.Path), name)
	u.RawQuery = "x=" + strconv.FormatInt(time.Now().UnixNano(), 10)
	return u.String()
}

func probe(ctx context.Context, client *http.Client, s ServerInfo) (float64, error) {
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.fileURL("latency.txt"), nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", userAgent)
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if _, err := io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<10)); err != nil {
		return 0, err
	}
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("status %d", resp.StatusCode)
	}
	return float64(time.Since(start).Microseconds()) / 1000, nil
}

func pickServer(ctx context.Context, client *http.Client, servers []ServerInfo) (ServerInfo, error) {
	var best ServerInfo
	bestMs := math.MaxFloat64
	for _, s := range servers {
		if ctx.Err() != nil {
			return ServerInfo{}, ctx.Err()
		}
		ms, err := probe(ctx, client, s)
		if err == nil && ms < bestMs {
			best, bestMs = s, ms
		}
	}
	if bestMs == math.MaxFloat64 {
		return ServerInfo{}, errNoServer
	}
	return best, nil
}

func pingSamples(ctx context.Context, client *http.Client, s ServerInfo, n int, each func(float64)) ([]float64, error) {
	samples := make([]float64, 0, n)
	for len(samples) < n {
		ms, err := probe(ctx, client, s)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, fmt.Errorf("ping to %s failed: %w", s.Host, err)
		}
		samples = append(samples, ms)
		each(ms)
	}
	return samples, nil
}

func minOf(v []float64) float64 {
	sorted := append([]float64(nil), v...)
	sort.Float64s(sorted)
	return sorted[0]
}

// jitter is the mean absolute difference between neighbouring ping samples.
func jitter(samples []float64) float64 {
	if len(samples) < 2 {
		return 0
	}
	var sum float64
	for i := 1; i < len(samples); i++ {
		sum += math.Abs(samples[i] - samples[i-1])
	}
	return sum / float64(len(samples)-1)
}

// worker moves bytes until its context ends, adding each chunk to moved.
type worker func(ctx context.Context, moved *atomic.Int64)

func downloadWorker(client *http.Client, s ServerInfo) worker {
	return func(ctx context.Context, moved *atomic.Int64) {
		buf := make([]byte, 64<<10)
		for ctx.Err() == nil {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.fileURL(downloadFile), nil)
			if err != nil {
				return
			}
			req.Header.Set("User-Agent", userAgent)
			resp, err := client.Do(req)
			if err != nil {
				pause(ctx)
				continue
			}
			for {
				n, err := resp.Body.Read(buf)
				moved.Add(int64(n))
				if err != nil {
					break
				}
			}
			resp.Body.Close()
		}
	}
}

// countingBody yields random-looking bytes and counts what the transport pulls from it.
type countingBody struct {
	chunk []byte
	left  int
	moved *atomic.Int64
}

func (b *countingBody) Read(p []byte) (int, error) {
	if b.left <= 0 {
		return 0, io.EOF
	}
	n := min(len(p), len(b.chunk), b.left)
	copy(p, b.chunk[:n])
	b.left -= n
	b.moved.Add(int64(n))
	return n, nil
}

func uploadWorker(client *http.Client, s ServerInfo) worker {
	chunk := randomChunk()
	return func(ctx context.Context, moved *atomic.Int64) {
		for ctx.Err() == nil {
			body := &countingBody{chunk: chunk, left: uploadBody, moved: moved}
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url, body)
			if err != nil {
				return
			}
			req.ContentLength = uploadBody
			req.Header.Set("User-Agent", userAgent)
			req.Header.Set("Content-Type", "application/octet-stream")
			resp, err := client.Do(req)
			if err != nil {
				pause(ctx)
				continue
			}
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<10))
			resp.Body.Close()
		}
	}
}

func pause(ctx context.Context) {
	select {
	case <-ctx.Done():
	case <-time.After(100 * time.Millisecond):
	}
}

// measure runs cfg.Connections workers for cfg.Phase and reports the rate after the warm-up.
func measure(ctx context.Context, cfg Config, phase Phase, run worker, emit func(Update)) (float64, error) {
	phaseCtx, cancel := context.WithTimeout(ctx, cfg.Phase)
	defer cancel()
	var moved atomic.Int64
	var wg sync.WaitGroup
	for range cfg.Connections {
		wg.Add(1)
		go func() {
			defer wg.Done()
			run(phaseCtx, &moved)
		}()
	}

	start := time.Now()
	ticker := time.NewTicker(tick)
	defer ticker.Stop()
	lastAt, lastBytes := start, int64(0)
	var warmAt time.Time
	var warmBytes int64
loop:
	for {
		select {
		case <-phaseCtx.Done():
			break loop
		case now := <-ticker.C:
			total := moved.Load()
			emit(Update{
				Phase: phase, Progress: math.Min(1, now.Sub(start).Seconds()/cfg.Phase.Seconds()),
				Mbps: mbps(total-lastBytes, now.Sub(lastAt)),
			})
			lastAt, lastBytes = now, total
			if warmAt.IsZero() && now.Sub(start) >= cfg.Warmup {
				warmAt, warmBytes = now, total
			}
		}
	}
	end, total := time.Now(), moved.Load()
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if warmAt.IsZero() {
		warmAt, warmBytes = start, 0
	}
	if total <= warmBytes {
		return 0, fmt.Errorf("%s moved no data", phase)
	}
	return mbps(total-warmBytes, end.Sub(warmAt)), nil
}

func mbps(bytes int64, d time.Duration) float64 {
	if d <= 0 {
		return 0
	}
	return float64(bytes) * 8 / d.Seconds() / 1e6
}

func randomChunk() []byte {
	chunk := make([]byte, uploadChunk)
	x := uint64(time.Now().UnixNano()) | 1
	for i := range chunk {
		x ^= x << 13
		x ^= x >> 7
		x ^= x << 17
		chunk[i] = byte(x)
	}
	return chunk
}
