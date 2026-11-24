package benchmark

import "math"

// Objective says what "best" means when the results are ranked.
type Objective string

const (
	// Speed ranks by throughput alone, download weighted over upload.
	Speed Objective = "speed"
	// Balanced weighs throughput, ping and jitter together.
	Balanced Objective = "balanced"
)

func (o Objective) valid() bool { return o == Speed || o == Balanced }

const (
	downloadShare = 0.7
	uploadShare   = 1 - downloadShare

	balancedThroughput = 0.5
	balancedPing       = 0.3
	balancedJitter     = 0.2
)

// Measurement is what one speed test through a tunnel reports.
type Measurement struct {
	PingMs   float64 `json:"pingMs" example:"84.5"`
	JitterMs float64 `json:"jitterMs" example:"3.2"`
	DownMbps float64 `json:"downMbps" example:"92.4"`
	UpMbps   float64 `json:"upMbps" example:"41.7"`
}

func (m Measurement) throughput() float64 {
	return downloadShare*m.DownMbps + uploadShare*m.UpMbps
}

// scoreAll fills Score on every successful result; higher is better. A failed result keeps score 0.
func scoreAll(results []MethodResult, objective Objective) {
	var topThroughput, bestPing float64
	for _, r := range results {
		if !r.OK {
			continue
		}
		topThroughput = math.Max(topThroughput, r.throughput())
		if r.PingMs > 0 && (bestPing == 0 || r.PingMs < bestPing) {
			bestPing = r.PingMs
		}
	}
	for i := range results {
		r := &results[i]
		if !r.OK {
			continue
		}
		if objective == Speed {
			r.Score = round2(r.throughput())
			continue
		}
		through := ratio(r.throughput(), topThroughput)
		ping := 1.0
		if r.PingMs > 0 {
			ping = ratio(bestPing, r.PingMs)
		}
		steady := 1.0
		if r.PingMs > 0 {
			steady = 1 - math.Min(1, r.JitterMs/r.PingMs)
		}
		r.Score = round2(100 * (balancedThroughput*through + balancedPing*ping + balancedJitter*steady))
	}
}

// best is the successful result with the highest score; the first one listed wins a tie.
func best(results []MethodResult) (Candidate, bool) {
	index := -1
	for i, r := range results {
		if r.OK && (index < 0 || r.Score > results[index].Score) {
			index = i
		}
	}
	if index < 0 {
		return Candidate{}, false
	}
	return results[index].Candidate, true
}

func ratio(part, whole float64) float64 {
	if whole <= 0 {
		return 0
	}
	return part / whole
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }
