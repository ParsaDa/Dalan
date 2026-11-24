package benchmark

import "testing"

func okResult(engine string, m Measurement) MethodResult {
	return MethodResult{Candidate: Candidate{Engine: engine, Transport: "t", Mode: "reverse"}, Measurement: m, OK: true}
}

func TestSpeedRanksByWeightedThroughputAndIgnoresPing(t *testing.T) {
	results := []MethodResult{
		okResult("slow", Measurement{PingMs: 10, DownMbps: 50, UpMbps: 50}),
		okResult("fast", Measurement{PingMs: 400, DownMbps: 200, UpMbps: 10}),
		{Candidate: Candidate{Engine: "broken"}},
	}
	scoreAll(results, Speed)

	if results[0].Score != 50 || results[1].Score != 143 || results[2].Score != 0 {
		t.Fatalf("scores = %v %v %v, want 50, 143 (0.7*200+0.3*10) and 0", results[0].Score, results[1].Score, results[2].Score)
	}
	if got, ok := best(results); !ok || got.Engine != "fast" {
		t.Fatalf("best = %+v, %v", got, ok)
	}
}

func TestBalancedPrefersAFastLinkOverAFasterOneWithAPoorPing(t *testing.T) {
	results := []MethodResult{
		okResult("steady", Measurement{PingMs: 40, JitterMs: 4, DownMbps: 90, UpMbps: 90}),
		okResult("fastest", Measurement{PingMs: 400, JitterMs: 200, DownMbps: 100, UpMbps: 100}),
	}
	scoreAll(results, Balanced)

	// steady: 0.5*0.9 + 0.3*1 + 0.2*0.9 = 0.93; fastest: 0.5*1 + 0.3*0.1 + 0.2*0.5 = 0.63.
	if results[0].Score != 93 || results[1].Score != 63 {
		t.Fatalf("scores = %v and %v, want 93 and 63", results[0].Score, results[1].Score)
	}
	if got, _ := best(results); got.Engine != "steady" {
		t.Fatalf("best = %+v, want the steady link", got)
	}
}

func TestBalancedDoesNotDivideByZeroWhenNothingMeasuredAThroughput(t *testing.T) {
	results := []MethodResult{okResult("idle", Measurement{PingMs: 30})}
	scoreAll(results, Balanced)
	if results[0].Score != 50 {
		t.Fatalf("score = %v, want 50 (ping and jitter only)", results[0].Score)
	}
}

func TestBestIsNotFoundWhenEveryMethodFailed(t *testing.T) {
	if _, ok := best([]MethodResult{{Candidate: Candidate{Engine: "a"}}, {Candidate: Candidate{Engine: "b"}}}); ok {
		t.Fatal("a best method was named although none worked")
	}
}

func TestBestKeepsTheFirstOfEqualScores(t *testing.T) {
	results := []MethodResult{
		okResult("first", Measurement{DownMbps: 10}),
		okResult("second", Measurement{DownMbps: 10}),
	}
	scoreAll(results, Speed)
	if got, _ := best(results); got.Engine != "first" {
		t.Fatalf("best = %+v, want the first of the tie", got)
	}
}
