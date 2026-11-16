package engine

import (
	"context"
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

const (
	defaultRestartDelay = 3 * time.Second
	stopGrace           = 5 * time.Second
)

type process struct {
	cmd    *exec.Cmd
	config string
	exited chan struct{}
	output *outputLog
}

// Supervisor keeps one engine process running per tunnel it wants, restarting a crashed one.
type Supervisor struct {
	// Name is the engine as the panel log and errors call it, such as "BackPack".
	Name         string
	Binary       string
	Dir          string
	RestartDelay time.Duration
	// LogLine receives each line an engine prints; nil sends it to the panel log.
	LogLine func(tunnel, line string)
	// Wants says whether a tunnel row is one this engine runs; disabled rows are skipped before it is asked.
	Wants func(model.ServerTunnel) bool
	// Render writes the configuration file of a tunnel; Args turns that file's path into the command line.
	Render func(model.ServerTunnel) (string, error)
	Args   func(configPath string) []string
	// NotInstalled is what Reconcile returns when tunnels are wanted but Binary is missing.
	NotInstalled error
	// Leftovers lists the files an engine writes for a tunnel besides its configuration; they go when it stops.
	Leftovers func(dir, name string) []string

	mu        sync.Mutex
	processes map[string]*process
}

func (s *Supervisor) configPath(name string) string { return filepath.Join(s.Dir, name+".toml") }

func (s *Supervisor) removeLeftovers(name string) {
	if s.Leftovers == nil {
		return
	}
	for _, path := range s.Leftovers(s.Dir, name) {
		_ = os.Remove(path)
	}
}

// Reconcile starts, restarts and stops engines until the running set matches the given tunnels.
func (s *Supervisor) Reconcile(tunnels []model.ServerTunnel) error {
	desired := map[string]string{}
	var firstErr error
	for _, t := range tunnels {
		if !t.Enable || !s.Wants(t) {
			continue
		}
		rendered, err := s.Render(t)
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
	if s.processes == nil {
		s.processes = map[string]*process{}
	}
	for name, p := range s.processes {
		if rendered, ok := desired[name]; !ok || rendered != p.config {
			s.stopLocked(name, p)
			if !ok {
				_ = os.Remove(s.configPath(name))
			}
		}
	}
	if len(desired) == 0 {
		return firstErr
	}
	if _, err := os.Stat(s.Binary); err != nil {
		return s.NotInstalled
	}
	for name, rendered := range desired {
		if _, running := s.processes[name]; running {
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
	// The file holds the tunnel's secrets, so it is written owner-only before the engine ever reads it.
	if err := os.WriteFile(s.configPath(name), []byte(rendered), 0o600); err != nil {
		return err
	}
	// The engine outlives any one request; it is stopped by signal, not by a context.
	// cmd.Dir is set, so a relative engine path such as the default "bin/backpack" would be looked up inside Dir.
	binary, err := filepath.Abs(s.Binary)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(context.Background(), binary, s.Args(s.configPath(name))...)
	cmd.Dir = s.Dir
	output := s.outputFor(name)
	cmd.Stdout, cmd.Stderr = output, output
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s for tunnel %q: %w", s.Name, name, err)
	}
	p := &process{cmd: cmd, config: rendered, exited: make(chan struct{}), output: output}
	s.processes[name] = p
	go s.watch(name, p)
	return nil
}

// watch restarts an engine that exited by itself; one stopped on purpose is no longer the current one.
func (s *Supervisor) watch(name string, p *process) {
	err := p.cmd.Wait()
	close(p.exited)

	delay := s.RestartDelay
	if delay <= 0 {
		delay = defaultRestartDelay
	}
	time.Sleep(delay)

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.processes[name] != p {
		return
	}
	logger.Warningf("%s engine for tunnel %q exited (%v); starting it again. Last output: %s", s.Name, name, err, p.output.finish())
	delete(s.processes, name)
	if err := s.startLocked(name, p.config); err != nil {
		logger.Warningf("%s engine for tunnel %q could not be restarted: %v", s.Name, name, err)
	}
}

func (s *Supervisor) stopLocked(name string, p *process) {
	delete(s.processes, name)
	// A stopped engine's status goes stale at once and would read as connected for a while.
	defer s.removeLeftovers(name)
	_ = p.cmd.Process.Signal(syscall.SIGTERM)
	select {
	case <-p.exited:
	case <-time.After(stopGrace):
		_ = p.cmd.Process.Kill()
		<-p.exited
	}
}

// StopAll stops every engine and leaves their configuration on disk for the next start.
func (s *Supervisor) StopAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for name, p := range s.processes {
		s.stopLocked(name, p)
	}
}
