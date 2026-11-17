package tor

import (
	"fmt"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/tunnel"
)

// The states an exit can be in, as the panel shows them.
const (
	StateNotInstalled = "not_installed"
	StateStopped      = "stopped"
	StateStarting     = "starting"
	StateConnected    = "connected"
	StateFailed       = "failed"
)

// ProbeTimeout bounds the request sent through an exit to learn whether it works.
const ProbeTimeout = 8 * time.Second

// Status is what the panel knows about one exit. Connected means a real request went through it.
type Status struct {
	State     string `json:"state" example:"connected"`
	Progress  int    `json:"progress" example:"100"`
	LatencyMs int64  `json:"latencyMs" example:"1400"`
	ExitIP    string `json:"exitIp" example:"203.0.113.9"`
	ExitLoc   string `json:"exitLoc" example:"DE"`
	Error     string `json:"error" example:""`
}

// Observation is what the panel sees of one exit before it asks Tor anything.
type Observation struct {
	Port        int
	Installed   bool
	Running     bool
	Progress    int
	HasProgress bool
}

// Prober sends one request through a SOCKS address; tunnel.ProbeVia is the real one.
type Prober func(socksAddr, url string, timeout time.Duration) tunnel.Status

// StatusOf judges one exit; only a fully bootstrapped Tor is asked to carry a request.
func StatusOf(o Observation, probe Prober) Status {
	switch {
	case !o.Installed:
		return Status{State: StateNotInstalled}
	case !o.Running:
		return Status{State: StateStopped}
	case !o.HasProgress || o.Progress < 100:
		return Status{State: StateStarting, Progress: o.Progress}
	}
	result := probe(fmt.Sprintf("127.0.0.1:%d", o.Port), tunnel.DefaultProbeURL, ProbeTimeout)
	if !result.Connected {
		return Status{State: StateFailed, Progress: o.Progress, Error: result.Error}
	}
	return Status{State: StateConnected, Progress: o.Progress, LatencyMs: result.LatencyMs, ExitIP: result.ExitIP, ExitLoc: result.ExitLoc}
}
