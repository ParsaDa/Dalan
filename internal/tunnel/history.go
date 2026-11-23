package tunnel

import (
	"sync"
	"time"
)

// historyLimit caps one tunnel's samples, so a page that probes often cannot grow memory.
const historyLimit = 120

type sample struct {
	at        time.Time
	latencyMs int64
}

// History keeps the probe results of the last hour per tunnel, in memory, for the latency chart.
type History struct {
	mu   sync.Mutex
	byID map[int][]sample
}

func NewHistory() *History { return &History{byID: map[int][]sample{}} }

// Add stores one probe and returns the latencies of the last hour, oldest first; a failed probe is 0.
func (h *History) Add(id int, st Status, now time.Time) []int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	ms := int64(0)
	if st.Connected {
		ms = st.LatencyMs
	}
	cut := now.Add(-time.Hour)
	kept := make([]sample, 0, len(h.byID[id])+1)
	for _, s := range h.byID[id] {
		if s.at.After(cut) {
			kept = append(kept, s)
		}
	}
	kept = append(kept, sample{at: now, latencyMs: ms})
	if len(kept) > historyLimit {
		kept = kept[len(kept)-historyLimit:]
	}
	h.byID[id] = kept
	out := make([]int64, len(kept))
	for i, s := range kept {
		out[i] = s.latencyMs
	}
	return out
}

func (h *History) Forget(id int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.byID, id)
}
