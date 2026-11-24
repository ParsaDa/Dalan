package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/logger"
	"github.com/mhsanaei/3x-ui/v3/internal/speedtest"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/benchmark"
)

// BenchmarkRequest asks for the given methods to be tried against one node.
type BenchmarkRequest struct {
	NodeId     int                   `json:"nodeId" example:"3"`
	Address    string                `json:"address" example:"203.0.113.10"`
	Objective  benchmark.Objective   `json:"objective" example:"balanced"`
	Candidates []benchmark.Candidate `json:"candidates"`
}

// benchmarkTunnels is what a benchmark needs of the tunnel service.
type benchmarkTunnels interface {
	CreateOnNode(nodeID int, spec tunnel.PortalSpec, routeInbounds []string, exitOutbound string) (*model.ServerTunnel, error)
	Status(id int) (tunnel.Status, error)
	Delete(id int) error
	DeleteHere(id int) error
	GetAll() ([]model.ServerTunnel, error)
}

// BenchmarkService tries tunnel methods one after another against a node, each on a temporary tunnel that is
// removed again, and keeps the last run for the panel to show. One run goes on at a time.
type BenchmarkService struct {
	tunnels benchmarkTunnels
	measure func(ctx context.Context, tunnelId int) (speedtest.Result, error)

	mu     sync.Mutex
	runner *benchmark.Runner
}

var (
	sharedBenchmark     *BenchmarkService
	sharedBenchmarkOnce sync.Once
)

// NewBenchmarkService returns the one service every controller shares, so a second run is refused panel-wide.
func NewBenchmarkService() *BenchmarkService {
	sharedBenchmarkOnce.Do(func() {
		sharedBenchmark = &BenchmarkService{
			tunnels: &TunnelService{},
			measure: NewSpeedtestService().Measure,
		}
	})
	return sharedBenchmark
}

// Start checks the request and begins the run in the background; poll Status for progress.
func (s *BenchmarkService) Start(req BenchmarkRequest) error {
	if req.NodeId <= 0 {
		return errors.New("choose the node to test against")
	}
	if err := benchmark.CheckAddress(req.Address, req.Candidates); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.runner != nil && s.runner.Status().Running {
		return benchmark.ErrRunning
	}
	s.RemoveLeftovers()
	runner := s.newRunner(req.NodeId, strings.TrimSpace(req.Address))
	if err := runner.Start(req.Objective, req.Candidates); err != nil {
		return err
	}
	s.runner = runner
	return nil
}

// Cancel stops the run after the tunnel being measured is removed.
func (s *BenchmarkService) Cancel() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.runner != nil {
		s.runner.Cancel()
	}
}

// Status is the current or last run; before any run it is an empty one.
func (s *BenchmarkService) Status() benchmark.RunStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.runner == nil {
		return benchmark.RunStatus{Results: []benchmark.MethodResult{}}
	}
	return s.runner.Status()
}

// RemoveLeftovers deletes the temporary tunnels an interrupted run left on this server and on their nodes.
func (s *BenchmarkService) RemoveLeftovers() {
	all, err := s.tunnels.GetAll()
	if err != nil {
		logger.Warning("benchmark: could not list tunnels to remove leftovers:", err)
		return
	}
	for _, t := range all {
		if t.Role != model.TunnelRolePortal || !strings.HasPrefix(t.Name, benchmark.NamePrefix) {
			continue
		}
		if err := s.removeTunnel(t.Id); err != nil {
			logger.Warningf("benchmark: could not remove the leftover tunnel %s: %v", t.Name, err)
		}
	}
}

func (s *BenchmarkService) newRunner(nodeId int, address string) *benchmark.Runner {
	return &benchmark.Runner{
		Create: func(_ context.Context, name string, c benchmark.Candidate) (int, error) {
			port, err := benchmark.FreePort()
			if err != nil {
				return 0, err
			}
			spec := benchmark.SpecFor(name, c, address, port)
			made, err := s.tunnels.CreateOnNode(nodeId, tunnel.PortalSpec{
				Name: spec.Name, Address: spec.Address, Port: spec.Port,
				Engine: spec.Engine, Transport: spec.Transport, Mode: spec.Mode,
			}, nil, "")
			if err != nil {
				return 0, err
			}
			return made.Id, nil
		},
		Connected: func(_ context.Context, id int) (bool, string) {
			st, err := s.tunnels.Status(id)
			if err != nil {
				return false, err.Error()
			}
			return st.Connected, st.Error
		},
		Measure: func(ctx context.Context, id int) (benchmark.Measurement, error) {
			res, err := s.measure(ctx, id)
			if err != nil {
				return benchmark.Measurement{}, err
			}
			return benchmark.Measurement{PingMs: res.PingMs, JitterMs: res.JitterMs, DownMbps: res.DownMbps, UpMbps: res.UpMbps}, nil
		},
		Remove: s.removeTunnel,
	}
}

// removeTunnel asks the node to drop its end first; when it cannot, this end still goes and the error says so.
func (s *BenchmarkService) removeTunnel(id int) error {
	err := s.tunnels.Delete(id)
	if err == nil {
		return nil
	}
	if hereErr := s.tunnels.DeleteHere(id); hereErr != nil {
		return fmt.Errorf("%w; this server could not remove its end either: %w", err, hereErr)
	}
	return fmt.Errorf("removed on this server only, the node's end is still there: %w", err)
}
