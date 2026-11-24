package benchmark

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeServers stands in for the two servers and records every call in order.
type fakeServers struct {
	mu      sync.Mutex
	calls   []string
	nextID  int
	connect func(id int) (bool, string)
	measure func(ctx context.Context, id int) (Measurement, error)
	create  func(name string, c Candidate) error
	remove  func(id int) error
	names   map[int]string
}

func newFakeServers() *fakeServers {
	return &fakeServers{
		names:   map[int]string{},
		connect: func(int) (bool, string) { return true, "" },
		measure: func(context.Context, int) (Measurement, error) {
			return Measurement{PingMs: 50, JitterMs: 5, DownMbps: 100, UpMbps: 50}, nil
		},
	}
}

func (f *fakeServers) log(format string, args ...any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, fmt.Sprintf(format, args...))
}

func (f *fakeServers) runner() *Runner {
	return &Runner{
		ConnectTimeout: 60 * time.Millisecond,
		PollEvery:      5 * time.Millisecond,
		Create: func(_ context.Context, name string, c Candidate) (int, error) {
			if f.create != nil {
				if err := f.create(name, c); err != nil {
					f.log("create-failed %s", name)
					return 0, err
				}
			}
			f.mu.Lock()
			f.nextID++
			id := f.nextID
			f.names[id] = name
			f.mu.Unlock()
			f.log("create %s %s/%s/%s", name, c.Engine, c.Transport, c.Mode)
			return id, nil
		},
		Connected: func(_ context.Context, id int) (bool, string) { return f.connect(id) },
		Measure: func(ctx context.Context, id int) (Measurement, error) {
			f.log("measure %s", f.names[id])
			return f.measure(ctx, id)
		},
		Remove: func(id int) error {
			f.log("remove %s", f.names[id])
			if f.remove != nil {
				return f.remove(id)
			}
			return nil
		},
	}
}

func (f *fakeServers) callList() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.calls...)
}

func waitDone(t *testing.T, r *Runner) RunStatus {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if s := r.Status(); !s.Running {
			return s
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("the benchmark did not finish")
	return RunStatus{}
}

func TestRunMakesMeasuresAndRemovesEachTunnelInTurn(t *testing.T) {
	f := newFakeServers()
	r := f.runner()
	candidates := []Candidate{
		{Engine: "gost", Transport: "tls", Mode: "reverse"},
		{Engine: "rathole", Transport: "tcp", Mode: "reverse"},
	}
	if err := r.Start(Balanced, candidates); err != nil {
		t.Fatal(err)
	}
	s := waitDone(t, r)

	want := []string{
		"create dalan-bench-1 gost/tls/reverse", "measure dalan-bench-1", "remove dalan-bench-1",
		"create dalan-bench-2 rathole/tcp/reverse", "measure dalan-bench-2", "remove dalan-bench-2",
	}
	if got := f.callList(); !reflect.DeepEqual(got, want) {
		t.Fatalf("calls = %v, want %v", got, want)
	}
	if s.Total != 2 || s.Done != 2 || s.Cancelled || s.Current != nil || len(s.Results) != 2 {
		t.Fatalf("status after the run = %+v", s)
	}
	if !s.Results[0].OK || s.Results[0].DownMbps != 100 || s.Results[0].Candidate != candidates[0] {
		t.Fatalf("first result = %+v", s.Results[0])
	}
}

func TestRunRemovesTheTunnelThatNeverConnectsAndGoesOn(t *testing.T) {
	f := newFakeServers()
	f.connect = func(id int) (bool, string) { return id == 2, "the exit does not answer" }
	r := f.runner()
	if err := r.Start(Speed, []Candidate{
		{Engine: "gost", Transport: "tls", Mode: "reverse"},
		{Engine: "gost", Transport: "ws", Mode: "reverse"},
	}); err != nil {
		t.Fatal(err)
	}
	s := waitDone(t, r)

	if first := s.Results[0]; first.OK || !strings.Contains(first.Error, "the exit does not answer") || first.Score != 0 {
		t.Fatalf("first result = %+v, want a failure that says why", first)
	}
	if !s.Results[1].OK {
		t.Fatalf("the run stopped at the first failure: %+v", s.Results[1])
	}
	calls := strings.Join(f.callList(), ",")
	if !strings.Contains(calls, "remove dalan-bench-1,create dalan-bench-2") {
		t.Fatalf("dalan-bench-1 was not removed before dalan-bench-2 was made: %s", calls)
	}
	if strings.Contains(calls, "measure dalan-bench-1") {
		t.Fatalf("a tunnel that never connected was measured: %s", calls)
	}
}

func TestRunRemovesTheTunnelWhenTheMeasurementFails(t *testing.T) {
	f := newFakeServers()
	f.measure = func(context.Context, int) (Measurement, error) {
		return Measurement{}, errors.New("no speedtest server answers")
	}
	r := f.runner()
	if err := r.Start(Speed, []Candidate{{Engine: "gost", Transport: "tls", Mode: "direct"}}); err != nil {
		t.Fatal(err)
	}
	s := waitDone(t, r)

	if got := s.Results[0]; got.OK || got.Error != "no speedtest server answers" {
		t.Fatalf("result = %+v", got)
	}
	if calls := f.callList(); calls[len(calls)-1] != "remove dalan-bench-1" {
		t.Fatalf("calls = %v, want the tunnel removed last", calls)
	}
}

func TestRunReportsACreateFailureAndHasNothingToRemove(t *testing.T) {
	f := newFakeServers()
	f.create = func(name string, _ Candidate) error { return errors.New("the gost engine is not installed") }
	r := f.runner()
	if err := r.Start(Speed, []Candidate{{Engine: "gost", Transport: "tls", Mode: "reverse"}}); err != nil {
		t.Fatal(err)
	}
	s := waitDone(t, r)

	if got := s.Results[0]; got.OK || got.Error != "the gost engine is not installed" {
		t.Fatalf("result = %+v", got)
	}
	if calls := f.callList(); !reflect.DeepEqual(calls, []string{"create-failed dalan-bench-1"}) {
		t.Fatalf("calls = %v, want only the failed create", calls)
	}
}

func TestRunKeepsWhyATunnelCouldNotBeRemoved(t *testing.T) {
	f := newFakeServers()
	f.remove = func(int) error { return errors.New("the node does not answer") }
	r := f.runner()
	if err := r.Start(Speed, []Candidate{{Engine: "gost", Transport: "tls", Mode: "reverse"}}); err != nil {
		t.Fatal(err)
	}
	s := waitDone(t, r)

	if got := s.Results[0]; !got.OK || got.Leftover != "the node does not answer" {
		t.Fatalf("result = %+v, want the measurement kept and the leftover named", got)
	}
}

func TestCancelRemovesTheTunnelBeingMeasuredAndMakesNoMore(t *testing.T) {
	f := newFakeServers()
	measuring := make(chan struct{})
	f.measure = func(ctx context.Context, _ int) (Measurement, error) {
		close(measuring)
		<-ctx.Done()
		return Measurement{}, ctx.Err()
	}
	r := f.runner()
	if err := r.Start(Speed, []Candidate{
		{Engine: "gost", Transport: "tls", Mode: "reverse"},
		{Engine: "gost", Transport: "ws", Mode: "reverse"},
	}); err != nil {
		t.Fatal(err)
	}
	<-measuring
	r.Cancel()
	s := waitDone(t, r)

	if !s.Cancelled || s.Best != nil {
		t.Fatalf("status = %+v, want a cancelled run with no best", s)
	}
	calls := strings.Join(f.callList(), ",")
	if !strings.HasSuffix(calls, "remove dalan-bench-1") || strings.Contains(calls, "dalan-bench-2") {
		t.Fatalf("calls = %s, want dalan-bench-1 removed and dalan-bench-2 never made", calls)
	}
}

func TestStartRefusesASecondRunWhileOneIsGoingOn(t *testing.T) {
	f := newFakeServers()
	release := make(chan struct{})
	f.measure = func(ctx context.Context, _ int) (Measurement, error) {
		<-release
		return Measurement{DownMbps: 1}, nil
	}
	r := f.runner()
	one := []Candidate{{Engine: "gost", Transport: "tls", Mode: "reverse"}}
	if err := r.Start(Speed, one); err != nil {
		t.Fatal(err)
	}
	if err := r.Start(Speed, one); !errors.Is(err, ErrRunning) {
		t.Fatalf("second Start = %v, want ErrRunning", err)
	}
	close(release)
	waitDone(t, r)
	if err := r.Start(Speed, one); err != nil {
		t.Fatalf("Start after the run finished = %v", err)
	}
	waitDone(t, r)
}

func TestStartRejectsARequestThatCannotRun(t *testing.T) {
	ok := Candidate{Engine: "gost", Transport: "tls", Mode: "reverse"}
	tooMany := make([]Candidate, MaxCandidates+1)
	for i := range tooMany {
		tooMany[i] = ok
	}
	for name, tc := range map[string]struct {
		objective  Objective
		candidates []Candidate
		want       string
	}{
		"unknown objective": {"fastest", []Candidate{ok}, "unknown objective"},
		"no methods":        {Speed, nil, "between 1 and"},
		"too many":          {Speed, tooMany, "between 1 and"},
		"no engine":         {Speed, []Candidate{{Transport: "tls", Mode: "reverse"}}, "incomplete method"},
		"no transport":      {Speed, []Candidate{{Engine: "gost", Mode: "reverse"}}, "incomplete method"},
		"bad mode":          {Speed, []Candidate{{Engine: "gost", Transport: "tls", Mode: "both"}}, "incomplete method"},
	} {
		r := newFakeServers().runner()
		if err := r.Start(tc.objective, tc.candidates); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err = %v, want it to mention %q", name, err, tc.want)
		}
		if r.Status().Running {
			t.Errorf("%s: a rejected request started a run", name)
		}
	}
}

func TestStatusIsACopyThatLaterProgressDoesNotChange(t *testing.T) {
	f := newFakeServers()
	r := f.runner()
	if err := r.Start(Speed, []Candidate{{Engine: "gost", Transport: "tls", Mode: "reverse"}}); err != nil {
		t.Fatal(err)
	}
	s := waitDone(t, r)
	s.Results[0].Error = "changed by the caller"
	if r.Status().Results[0].Error != "" {
		t.Fatal("editing a returned status changed the runner's own results")
	}
}
