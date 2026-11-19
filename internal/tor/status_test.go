package tor

import (
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/tunnel"
)

func TestStatusOf(t *testing.T) {
	probed := ""
	prober := func(addr, url string, timeout time.Duration) tunnel.Status {
		probed = addr
		if addr == "127.0.0.1:62701" {
			return tunnel.Status{Error: "socks connect: general failure"}
		}
		return tunnel.Status{Connected: true, LatencyMs: 1400, ExitIP: "203.0.113.9", ExitLoc: "DE"}
	}
	cases := []struct {
		name string
		in   Observation
		want Status
		call bool
	}{
		{"no bundle", Observation{Port: 62700, Installed: false, Running: false}, Status{State: StateNotInstalled}, false},
		{"not running", Observation{Port: 62700, Installed: true, Running: false}, Status{State: StateStopped}, false},
		{"no log line yet", Observation{Port: 62700, Installed: true, Running: true}, Status{State: StateStarting}, false},
		{"bootstrapping", Observation{Port: 62700, Installed: true, Running: true, Progress: 40, HasProgress: true}, Status{State: StateStarting, Progress: 40}, false},
		{"bootstrapped and reachable", Observation{Port: 62700, Installed: true, Running: true, Progress: 100, HasProgress: true}, Status{State: StateConnected, Progress: 100, LatencyMs: 1400, ExitIP: "203.0.113.9", ExitLoc: "DE"}, true},
		{"bootstrapped but the request fails", Observation{Port: 62701, Installed: true, Running: true, Progress: 100, HasProgress: true}, Status{State: StateFailed, Progress: 100, Error: "socks connect: general failure"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			probed = ""
			got := StatusOf(c.in, prober)
			if got != c.want {
				t.Fatalf("StatusOf = %+v, want %+v", got, c.want)
			}
			if (probed != "") != c.call {
				t.Fatalf("probed %q, want a probe: %v", probed, c.call)
			}
		})
	}
}
