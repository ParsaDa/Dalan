// Package pool decides which tunnel of a pool carries the traffic.
package pool

const (
	// SwitchAfter is how many rounds in a row the same tunnel must be better before traffic moves to it.
	SwitchAfter = 3
	// minGainMs and maxRatio together say how much faster a working tunnel has to be to take over.
	minGainMs = 50
	maxRatio  = 0.7
)

// Reading is one health measurement of a pool member.
type Reading struct {
	Name string
	Up   bool
	// LatencyMs is 0 when the engine reports a working tunnel without measuring it.
	LatencyMs int
}

// Monitor keeps the state between rounds for one pool.
type Monitor struct {
	active    string
	candidate string
	streak    int
}

func New(active string) *Monitor { return &Monitor{active: active} }

func (m *Monitor) Active() string { return m.active }

// Observe takes one round of readings, in the pool's order of preference, and returns the
// tunnel to move the traffic to, or "" to stay.
func (m *Monitor) Observe(readings []Reading) string {
	best := m.better(readings)
	if best == "" {
		m.candidate, m.streak = "", 0
		return ""
	}
	if best != m.candidate {
		m.candidate, m.streak = best, 0
	}
	m.streak++
	if m.streak < SwitchAfter {
		return ""
	}
	m.active, m.candidate, m.streak = best, "", 0
	return best
}

// better names the member worth moving to this round, or "" when the active one should stay.
func (m *Monitor) better(readings []Reading) string {
	var active *Reading
	best := -1
	for i := range readings {
		r := readings[i]
		if r.Name == m.active {
			active = &readings[i]
		}
		if r.Up && (best < 0 || measuredFaster(r.LatencyMs, readings[best].LatencyMs)) {
			best = i
		}
	}
	if best < 0 || readings[best].Name == m.active {
		return ""
	}
	if active != nil && active.Up && !clearlyFaster(readings[best].LatencyMs, active.LatencyMs) {
		return ""
	}
	return readings[best].Name
}

// measuredFaster ranks tunnels with a measurement ahead of those without, and the earlier one wins a tie.
func measuredFaster(a, b int) bool {
	return a > 0 && (b == 0 || a < b)
}

func clearlyFaster(candidate, active int) bool {
	return candidate > 0 && active > 0 && active-candidate >= minGainMs && float64(candidate) <= float64(active)*maxRatio
}
