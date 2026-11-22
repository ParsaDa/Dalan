package tunnel

import (
	"testing"
	"time"
)

func TestHistoryKeepsTheLastHourOfProbesPerTunnel(t *testing.T) {
	h := NewHistory()
	start := time.Unix(1_700_000_000, 0)
	h.Add(1, Status{Connected: true, LatencyMs: 40}, start)
	h.Add(2, Status{Connected: true, LatencyMs: 90}, start)
	h.Add(1, Status{Connected: false}, start.Add(30*time.Minute))
	got := h.Add(1, Status{Connected: true, LatencyMs: 55}, start.Add(61*time.Minute))
	// The first probe is over an hour old, and a failed probe is kept as 0 so the gap shows.
	want := []int64{0, 55}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("history = %v, want %v", got, want)
	}
	if other := h.Add(2, Status{Connected: true, LatencyMs: 80}, start.Add(time.Minute)); len(other) != 2 {
		t.Fatalf("tunnel 2 history = %v, want two probes", other)
	}
}

func TestHistoryForgetsARemovedTunnel(t *testing.T) {
	h := NewHistory()
	now := time.Unix(1_700_000_000, 0)
	h.Add(3, Status{Connected: true, LatencyMs: 10}, now)
	h.Forget(3)
	if got := h.Add(3, Status{Connected: true, LatencyMs: 20}, now); len(got) != 1 {
		t.Fatalf("history after forget = %v, want one probe", got)
	}
}
