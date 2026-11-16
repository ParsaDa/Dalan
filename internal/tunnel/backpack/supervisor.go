package backpack

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/logger"
)

// ErrNotInstalled means a BackPack tunnel exists but the engine binary is not on this server.
var ErrNotInstalled = errors.New("the BackPack engine is not installed on this server")

const (
	defaultRestartDelay = 3 * time.Second
	stopGrace           = 5 * time.Second
)

type engine struct {
	cmd    *exec.Cmd
	config string
	exited chan struct{}
	output *outputLog
}

// Supervisor keeps one BackPack process running per enabled BackPack tunnel.
type Supervisor struct {
	Binary       string
	Dir          string
	RestartDelay time.Duration
	// LogLine receives each line an engine prints; nil sends it to the panel log.
	LogLine func(tunnel, line string)

	mu      sync.Mutex
	engines map[string]*engine
}

func (s *Supervisor) configPath(name string) string { return filepath.Join(s.Dir, name+".toml") }

// Reconcile starts, restarts and stops engines until the running set matches the given tunnels.
func (s *Supervisor) Reconcile(tunnels []model.ServerTunnel) error {
	desired := map[string]string{}
	var firstErr error
	for _, t := range tunnels {
		if !t.Enable || t.Engine != model.TunnelEngineBackpack {
			continue
		}
		rendered, err := RenderConfig(t)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		desired[t.Name] = rendered
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.engines == nil {
		s.engines = map[string]*engine{}
	}
	for name, e := range s.engines {
		if rendered, ok := desired[name]; !ok || rendered != e.config {
			s.stopLocked(name, e)
			if !ok {
				_ = os.Remove(s.configPath(name))
				_ = os.Remove(snapshotPath(s.Dir, name))
			}
		}
	}
	if len(desired) == 0 {
		return firstErr
	}
	if _, err := os.Stat(s.Binary); err != nil {
		return ErrNotInstalled
	}
	for name, rendered := range desired {
		if _, running := s.engines[name]; running {
			continue
		}
		if err := s.startLocked(name, rendered); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (s *Supervisor) startLocked(name, rendered string) error {
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return err
	}
	// The file holds the tunnel token, so it is written owner-only before the engine ever reads it.
	if err := os.WriteFile(s.configPath(name), []byte(rendered), 0o600); err != nil {
		return err
	}
	// The engine outlives any one request; it is stopped by signal, not by a context.
	// cmd.Dir is set, so a relative engine path such as the default "bin/backpack" would be looked up inside Dir.
	binary, err := filepath.Abs(s.Binary)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(context.Background(), binary, "-c", s.configPath(name))
	cmd.Dir = s.Dir
	output := s.outputFor(name)
	cmd.Stdout, cmd.Stderr = output, output
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start BackPack for tunnel %q: %w", name, err)
	}
	e := &engine{cmd: cmd, config: rendered, exited: make(chan struct{}), output: output}
	s.engines[name] = e
	go s.watch(name, e)
	return nil
}

// watch restarts an engine that exited by itself; one stopped on purpose is no longer the current one.
func (s *Supervisor) watch(name string, e *engine) {
	err := e.cmd.Wait()
	close(e.exited)

	delay := s.RestartDelay
	if delay <= 0 {
		delay = defaultRestartDelay
	}
	time.Sleep(delay)

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.engines[name] != e {
		return
	}
	logger.Warningf("BackPack engine for tunnel %q exited (%v); starting it again. Last output: %s", name, err, e.output.finish())
	delete(s.engines, name)
	if err := s.startLocked(name, e.config); err != nil {
		logger.Warningf("BackPack engine for tunnel %q could not be restarted: %v", name, err)
	}
}

func (s *Supervisor) stopLocked(name string, e *engine) {
	delete(s.engines, name)
	// A stopped engine's status file goes stale at once and would read as connected for up to 30 seconds.
	defer func() { _ = os.Remove(snapshotPath(s.Dir, name)) }()
	_ = e.cmd.Process.Signal(syscall.SIGTERM)
	select {
	case <-e.exited:
	case <-time.After(stopGrace):
		_ = e.cmd.Process.Kill()
		<-e.exited
	}
}

// StopAll stops every engine and leaves their configuration on disk for the next start.
func (s *Supervisor) StopAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for name, e := range s.engines {
		s.stopLocked(name, e)
	}
}
