// Package benchmark tries tunnel methods against one node and ranks them. It owns the order of the steps and
// the cleanup; making, probing and removing a tunnel are supplied by the caller.
package benchmark

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	// silentLimit is how many methods in a row may get no answer from the exit server before the run stops.
	silentLimit = 3
	// NamePrefix marks the temporary tunnels, so leftovers of an interrupted run can be told apart.
	NamePrefix = "dalan-bench-"

	defaultConnectTimeout = 45 * time.Second
	defaultPollEvery      = time.Second
)

// ErrRunning is returned by Start while a run is going on.
var ErrRunning = errors.New("a benchmark is already running")

// ErrUnanswered marks a measurement that failed because the exit server gave no answer at all.
var ErrUnanswered = errors.New("the exit server gave no answer")

type unanswered struct{ err error }

func (u unanswered) Error() string        { return u.err.Error() }
func (u unanswered) Unwrap() error        { return u.err }
func (u unanswered) Is(target error) bool { return target == ErrUnanswered }

// Unanswered wraps err so that errors.Is(err, ErrUnanswered) holds while the message stays err's own.
func Unanswered(err error) error { return unanswered{err} }

// Candidate is one method to try: an engine, one of its transports and the direction of the tunnel.
type Candidate struct {
	Engine    string `json:"engine" example:"gost"`
	Transport string `json:"transport" example:"tls"`
	Mode      string `json:"mode" example:"reverse"`
}

// MethodResult is the outcome for one candidate; Score is zero for a failure.
type MethodResult struct {
	Candidate
	Measurement
	OK        bool    `json:"ok" example:"true"`
	Error     string  `json:"error" example:""`
	ConnectMs int64   `json:"connectMs" example:"2100"`
	Score     float64 `json:"score" example:"88.4"`
	Leftover  string  `json:"leftover" example:""`
}

// Phase is the step a run is in for its current candidate.
type Phase string

const (
	PhaseCreate  Phase = "create"
	PhaseConnect Phase = "connect"
	PhaseMeasure Phase = "measure"
	PhaseRemove  Phase = "remove"
)

// RunStatus is a snapshot that is safe to hold while a run goes on.
type RunStatus struct {
	Running   bool           `json:"running" example:"true"`
	Cancelled bool           `json:"cancelled" example:"false"`
	Objective Objective      `json:"objective" example:"balanced"`
	Total     int            `json:"total" example:"12"`
	Done      int            `json:"done" example:"4"`
	Current   *Candidate     `json:"current"`
	Phase     Phase          `json:"phase" example:"measure"`
	Results   []MethodResult `json:"results"`
	Best      *Candidate     `json:"best"`
}

// Runner runs one benchmark at a time. The function fields are how it reaches the servers.
type Runner struct {
	// Create makes the temporary tunnel on both servers and returns its id.
	Create func(ctx context.Context, name string, c Candidate) (int, error)
	// Connected reports whether the tunnel carries traffic; detail says why not.
	Connected func(ctx context.Context, id int) (ok bool, detail string)
	// Measure runs a speed test through the tunnel.
	Measure func(ctx context.Context, id int) (Measurement, error)
	// Remove deletes the tunnel on both servers.
	Remove func(id int) error

	ConnectTimeout time.Duration
	PollEvery      time.Duration

	mu     sync.Mutex
	status RunStatus
	cancel context.CancelFunc
}

// Start validates the request and begins a run in the background; poll Status for progress.
func (r *Runner) Start(objective Objective, candidates []Candidate) error {
	if !objective.valid() {
		return fmt.Errorf("unknown objective %q", objective)
	}
	if len(candidates) == 0 {
		return errors.New("a benchmark needs at least one method")
	}
	for _, c := range candidates {
		if c.Engine == "" || c.Transport == "" || (c.Mode != "reverse" && c.Mode != "direct") {
			return fmt.Errorf("incomplete method %+v: an engine, a transport and a mode of reverse or direct are required", c)
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.status.Running {
		return ErrRunning
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	r.status = RunStatus{Running: true, Objective: objective, Total: len(candidates), Results: []MethodResult{}}
	go r.run(ctx, objective, append([]Candidate(nil), candidates...))
	return nil
}

// Cancel stops the run after the tunnel being tested is removed.
func (r *Runner) Cancel() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.status.Running && r.cancel != nil {
		r.cancel()
	}
}

// Status returns a copy of the current or last run.
func (r *Runner) Status() RunStatus {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.status
	out.Results = append([]MethodResult(nil), r.status.Results...)
	if r.status.Current != nil {
		c := *r.status.Current
		out.Current = &c
	}
	if r.status.Best != nil {
		b := *r.status.Best
		out.Best = &b
	}
	return out
}

func (r *Runner) update(change func(*RunStatus)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	change(&r.status)
}

func (r *Runner) run(ctx context.Context, objective Objective, candidates []Candidate) {
	defer r.cancel()
	silent := 0
	for i, c := range candidates {
		if ctx.Err() != nil {
			break
		}
		c := c
		r.update(func(s *RunStatus) { s.Current, s.Phase = &c, PhaseCreate })
		result, noAnswer := r.try(ctx, fmt.Sprintf("%s%d", NamePrefix, i+1), c)
		stop := false
		if noAnswer {
			silent++
			if silent >= silentLimit && i < len(candidates)-1 {
				result.Error += fmt.Sprintf(" (the run was stopped: the exit server gave no answer for %d methods in a row)", silent)
				stop = true
			}
		} else {
			silent = 0
		}
		r.update(func(s *RunStatus) {
			s.Results = append(s.Results, result)
			s.Done = len(s.Results)
			scoreAll(s.Results, objective)
		})
		if stop {
			break
		}
	}
	r.update(func(s *RunStatus) {
		s.Running, s.Current, s.Phase = false, nil, ""
		s.Cancelled = ctx.Err() != nil
		if b, ok := best(s.Results); ok {
			s.Best = &b
		}
	})
}

// try makes the tunnel, measures it and always removes it again; noAnswer is true when the exit server
// gave the measurement no answer at all.
func (r *Runner) try(ctx context.Context, name string, c Candidate) (result MethodResult, noAnswer bool) {
	result = MethodResult{Candidate: c}
	started := time.Now()
	id, err := r.Create(ctx, name, c)
	if err != nil {
		result.Error = err.Error()
		return result, false
	}
	defer func() {
		r.update(func(s *RunStatus) { s.Phase = PhaseRemove })
		if err := r.Remove(id); err != nil {
			result.Leftover = err.Error()
		}
	}()

	r.update(func(s *RunStatus) { s.Phase = PhaseConnect })
	if detail, ok := r.waitConnected(ctx, id); !ok {
		result.Error = detail
		return result, false
	}
	result.ConnectMs = time.Since(started).Milliseconds()

	r.update(func(s *RunStatus) { s.Phase = PhaseMeasure })
	m, err := r.Measure(ctx, id)
	if err != nil {
		result.Error = err.Error()
		return result, errors.Is(err, ErrUnanswered)
	}
	result.Measurement, result.OK = m, true
	return result, false
}

func (r *Runner) waitConnected(ctx context.Context, id int) (string, bool) {
	timeout, every := r.ConnectTimeout, r.PollEvery
	if timeout <= 0 {
		timeout = defaultConnectTimeout
	}
	if every <= 0 {
		every = defaultPollEvery
	}
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	tick := time.NewTicker(every)
	defer tick.Stop()
	detail := ""
	for {
		ok, why := r.Connected(ctx, id)
		if ok {
			return "", true
		}
		detail = why
		select {
		case <-ctx.Done():
			return "cancelled", false
		case <-deadline.C:
			if detail == "" {
				detail = "did not connect"
			}
			return fmt.Sprintf("%s (waited %s)", detail, timeout), false
		case <-tick.C:
		}
	}
}
