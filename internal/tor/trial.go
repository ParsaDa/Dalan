package tor

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel"
)

// ErrTrialBusy means another country is being tested; one throwaway Tor at a time keeps the server light.
var ErrTrialBusy = errors.New("another country is being tested; wait for it to finish")

const (
	trialStopGrace = 3 * time.Second
	trialTailLines = 5
)

var trialMu sync.Mutex

// TrialResult is what a throwaway Tor learned about one country.
type TrialResult struct {
	OK        bool   `json:"ok" example:"true"`
	LatencyMs int64  `json:"latencyMs" example:"1400"`
	ExitIP    string `json:"exitIp" example:"203.0.113.9"`
	ExitLoc   string `json:"exitLoc" example:"DE"`
	Reason    string `json:"reason" example:""`
	Error     string `json:"error" example:""`
}

// Trial starts a Tor for one country in a throwaway folder, waits for it to connect, sends one request
// through it and always stops it and removes its folder. ctx bounds the whole test.
func Trial(ctx context.Context, country string, bridges []string, layout Layout, probe Prober) (TrialResult, error) {
	cc, ok := NormalizeCountry(country)
	if !ok {
		return TrialResult{}, errors.New("a country is two letters, such as de or nl")
	}
	if !trialMu.TryLock() {
		return TrialResult{}, ErrTrialBusy
	}
	defer trialMu.Unlock()
	if _, err := os.Stat(layout.Binary()); err != nil {
		return TrialResult{}, ErrNotInstalled
	}
	dir, err := os.MkdirTemp("", "dalan-tor-trial-")
	if err != nil {
		return TrialResult{}, err
	}
	defer os.RemoveAll(dir)
	port, err := freeLoopbackPort()
	if err != nil {
		return TrialResult{}, err
	}
	config, err := RenderConfig(model.TorExit{Country: cc, Port: port, Enable: true}, bridges, Layout{Root: dir, BundleDir: layout.BundleDir})
	if err != nil {
		return TrialResult{}, err
	}
	if err := os.MkdirAll(filepath.Join(dir, "data"), 0o700); err != nil {
		return TrialResult{}, err
	}
	configPath := filepath.Join(dir, "trial.torrc")
	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		return TrialResult{}, err
	}

	runCtx, stop := context.WithCancel(ctx)
	defer stop()
	cmd := exec.CommandContext(runCtx, layout.Binary(), "-f", configPath)
	cmd.Env = append(os.Environ(), "LD_LIBRARY_PATH="+layout.LibraryDir())
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
	cmd.WaitDelay = trialStopGrace
	reader, writer := io.Pipe()
	cmd.Stdout, cmd.Stderr = writer, writer
	if err := cmd.Start(); err != nil {
		return TrialResult{}, fmt.Errorf("start Tor: %w", err)
	}
	seen := newTrialLog()
	go func() {
		seen.scan(reader)
		close(seen.scanned)
	}()
	exited := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		_ = writer.Close()
		close(exited)
	}()

	result := TrialResult{}
	select {
	case <-seen.booted:
		status := probe(fmt.Sprintf("127.0.0.1:%d", port), tunnel.DefaultProbeURL, ProbeTimeout)
		if status.Connected {
			result = TrialResult{OK: true, LatencyMs: status.LatencyMs, ExitIP: status.ExitIP, ExitLoc: status.ExitLoc}
		} else {
			result = TrialResult{Error: status.Error, Reason: seen.probeFailureReason()}
		}
	case <-exited:
		<-seen.scanned
		result = TrialResult{Reason: seen.stoppedReason()}
	case <-ctx.Done():
		result = TrialResult{Reason: seen.timeoutReason()}
	}
	stop()
	<-exited
	return result, nil
}

func freeLoopbackPort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

// trialLog keeps what a throwaway Tor printed: how far it got and its last few lines.
type trialLog struct {
	booted chan struct{}
	// scanned closes once every line Tor printed has been read.
	scanned chan struct{}
	once    sync.Once

	mu       sync.Mutex
	progress int
	tail     []string
}

func newTrialLog() *trialLog {
	return &trialLog{booted: make(chan struct{}), scanned: make(chan struct{})}
}

func (l *trialLog) scan(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		l.mu.Lock()
		l.tail = append(l.tail, line)
		if len(l.tail) > trialTailLines {
			l.tail = l.tail[1:]
		}
		if percent, ok := ParseBootstrap(line); ok {
			l.progress = percent
		}
		done := l.progress == 100
		l.mu.Unlock()
		if done {
			l.once.Do(func() { close(l.booted) })
		}
	}
}

func (l *trialLog) said() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return strings.Join(l.tail, " | ")
}

func (l *trialLog) probeFailureReason() string {
	said := l.said()
	for _, sign := range []string{"Failed to find node", "No Tor server allows exit", "no exit"} {
		if strings.Contains(said, sign) {
			return "no exit relays reachable in this country"
		}
	}
	return "Tor connected, but a request through it failed"
}

func (l *trialLog) stoppedReason() string {
	return "Tor stopped before it connected: " + l.said()
}

func (l *trialLog) timeoutReason() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return fmt.Sprintf("Tor did not finish connecting (it reached %d%%); if this server cannot reach Tor directly, add bridges", l.progress)
}
