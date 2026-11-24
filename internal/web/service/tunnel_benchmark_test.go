package service

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/speedtest"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/benchmark"
)

type fakeBenchTunnels struct {
	mu          sync.Mutex
	made        []tunnel.PortalSpec
	nodes       []int
	deleted     []int
	deletedHere []int
	existing    []model.ServerTunnel
	deleteErr   error
}

func (f *fakeBenchTunnels) CreateOnNode(nodeID int, spec tunnel.PortalSpec, _ []string, _ string) (*model.ServerTunnel, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.made = append(f.made, spec)
	f.nodes = append(f.nodes, nodeID)
	return &model.ServerTunnel{Id: 100 + len(f.made), Name: spec.Name}, nil
}

func (f *fakeBenchTunnels) Status(int) (tunnel.Status, error) {
	return tunnel.Status{Connected: true}, nil
}

func (f *fakeBenchTunnels) Delete(id int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deleted = append(f.deleted, id)
	return nil
}

func (f *fakeBenchTunnels) DeleteHere(id int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deletedHere = append(f.deletedHere, id)
	return nil
}

func (f *fakeBenchTunnels) GetAll() ([]model.ServerTunnel, error) { return f.existing, nil }

func newBenchmarkForTest(f *fakeBenchTunnels) *BenchmarkService {
	return &BenchmarkService{
		tunnels: f,
		measure: func(context.Context, int) (speedtest.Result, error) {
			return speedtest.Result{PingMs: 40, JitterMs: 4, DownMbps: 90, UpMbps: 30}, nil
		},
	}
}

func waitBenchmarkDone(t *testing.T, s *BenchmarkService) benchmark.RunStatus {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if st := s.Status(); !st.Running {
			return st
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("the benchmark never finished")
	return benchmark.RunStatus{}
}

func TestBenchmarkStartNeedsANodeAndAnAddressForReverseMethods(t *testing.T) {
	s := newBenchmarkForTest(&fakeBenchTunnels{})
	reverse := []benchmark.Candidate{{Engine: "gost", Transport: "tls", Mode: "reverse"}}

	if err := s.Start(BenchmarkRequest{Objective: benchmark.Speed, Candidates: reverse, Address: "203.0.113.10"}); err == nil || !strings.Contains(err.Error(), "node") {
		t.Fatalf("no node: err = %v", err)
	}
	if err := s.Start(BenchmarkRequest{NodeId: 3, Objective: benchmark.Speed, Candidates: reverse}); err == nil || !strings.Contains(err.Error(), "address of this server") {
		t.Fatalf("no address: err = %v", err)
	}
	if st := s.Status(); st.Running || st.Results == nil {
		t.Fatalf("status after refusals = %+v, want an idle empty run", st)
	}
}

func TestBenchmarkMakesATemporaryTunnelOnTheNodeMeasuresItAndRemovesIt(t *testing.T) {
	f := &fakeBenchTunnels{}
	s := newBenchmarkForTest(f)
	err := s.Start(BenchmarkRequest{
		NodeId: 3, Address: "203.0.113.10", Objective: benchmark.Speed,
		Candidates: []benchmark.Candidate{
			{Engine: "gost", Transport: "tls", Mode: "reverse"},
			{Engine: "singbox", Transport: "wss", Mode: "direct"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	st := waitBenchmarkDone(t, s)

	if len(f.made) != 2 || f.nodes[0] != 3 || f.nodes[1] != 3 {
		t.Fatalf("made %+v on nodes %v, want two tunnels on node 3", f.made, f.nodes)
	}
	first, second := f.made[0], f.made[1]
	if first.Name != "dalan-bench-1" || first.Engine != "gost" || first.Transport != "tls" || first.Mode != "reverse" || first.Address != "203.0.113.10" || first.Port == 0 {
		t.Fatalf("first spec = %+v", first)
	}
	if second.Address != "" || second.Mode != "direct" {
		t.Fatalf("second spec = %+v, a direct method takes the node's own address", second)
	}
	if len(f.deleted) != 2 || f.deleted[0] != 101 || f.deleted[1] != 102 {
		t.Fatalf("deleted %v, want both temporary tunnels removed", f.deleted)
	}
	if got := st.Results[0]; !got.OK || got.DownMbps != 90 || got.UpMbps != 30 || got.PingMs != 40 {
		t.Fatalf("result = %+v, want the speed test numbers", got)
	}
	if st.Best == nil {
		t.Fatal("no best method named")
	}
}

func TestBenchmarkRemovesOnlyThisEndAndSaysSoWhenTheNodeCannotBeAsked(t *testing.T) {
	f := &fakeBenchTunnels{deleteErr: errors.New("the node does not answer")}
	s := newBenchmarkForTest(f)
	err := s.Start(BenchmarkRequest{
		NodeId: 3, Address: "203.0.113.10", Objective: benchmark.Speed,
		Candidates: []benchmark.Candidate{{Engine: "gost", Transport: "tls", Mode: "reverse"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	st := waitBenchmarkDone(t, s)

	if len(f.deletedHere) != 1 || f.deletedHere[0] != 101 {
		t.Fatalf("deleted here = %v, want this end removed anyway", f.deletedHere)
	}
	if got := st.Results[0].Leftover; !strings.Contains(got, "node's end is still there") || !strings.Contains(got, "the node does not answer") {
		t.Fatalf("leftover = %q", got)
	}
}

func TestRemoveLeftoversTouchesOnlyEntryTunnelsWithTheReservedPrefix(t *testing.T) {
	f := &fakeBenchTunnels{existing: []model.ServerTunnel{
		{Id: 1, Name: "dalan-bench-1", Role: model.TunnelRolePortal},
		{Id: 2, Name: "dalan-bench-2", Role: model.TunnelRoleBridge},
		{Id: 3, Name: "de1", Role: model.TunnelRolePortal},
		{Id: 4, Name: "bench-notes", Role: model.TunnelRolePortal},
	}}
	newBenchmarkForTest(f).RemoveLeftovers()

	if len(f.deleted) != 1 || f.deleted[0] != 1 {
		t.Fatalf("deleted %v, want only tunnel 1", f.deleted)
	}
}
