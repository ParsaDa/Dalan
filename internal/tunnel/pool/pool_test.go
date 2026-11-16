package pool

import "testing"

func up(name string, ms int) Reading { return Reading{Name: name, Up: true, LatencyMs: ms} }
func down(name string) Reading       { return Reading{Name: name} }

// run feeds the rounds one by one and returns what each round decided.
func run(m *Monitor, rounds [][]Reading) []string {
	out := make([]string, len(rounds))
	for i, r := range rounds {
		out[i] = m.Observe(r)
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestADownActiveTunnelIsLeftOnlyAfterSeveralRoundsInARow(t *testing.T) {
	m := New("a")
	round := []Reading{down("a"), up("b", 120)}
	got := run(m, [][]Reading{round, round, round, round})
	if want := []string{"", "", "b", ""}; !equal(got, want) {
		t.Fatalf("decisions = %q, want %q", got, want)
	}
	if m.Active() != "b" {
		t.Fatalf("active = %q, want b", m.Active())
	}
}

func TestABriefDipDoesNotMoveTheTraffic(t *testing.T) {
	m := New("a")
	bad := []Reading{down("a"), up("b", 120)}
	good := []Reading{up("a", 100), up("b", 120)}
	got := run(m, [][]Reading{bad, bad, good, bad, bad, good})
	if want := []string{"", "", "", "", "", ""}; !equal(got, want) {
		t.Fatalf("decisions = %q, want no switch", got)
	}
	if m.Active() != "a" {
		t.Fatalf("active = %q, want a", m.Active())
	}
}

func TestTheStreakRestartsWhenADifferentTunnelBecomesTheBest(t *testing.T) {
	m := New("a")
	toB := []Reading{down("a"), up("b", 100), up("c", 300)}
	toC := []Reading{down("a"), up("b", 400), up("c", 100)}
	got := run(m, [][]Reading{toB, toB, toC, toC, toC})
	if want := []string{"", "", "", "", "c"}; !equal(got, want) {
		t.Fatalf("decisions = %q, want a switch to c only after three rounds of c", got)
	}
}

func TestAWorkingTunnelIsLeftOnlyForAClearlyFasterOne(t *testing.T) {
	cases := []struct {
		name   string
		active int
		other  int
		moves  bool
	}{
		{"a few milliseconds faster", 200, 190, false},
		{"faster by under a third", 300, 230, false},
		{"fast enough in ratio but not by 50 ms", 100, 60, false},
		{"clearly faster", 400, 150, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := New("a")
			round := []Reading{up("a", tc.active), up("b", tc.other)}
			got := run(m, [][]Reading{round, round, round})
			if moved := got[2] == "b"; moved != tc.moves {
				t.Fatalf("decisions = %q, moves = %v, want %v", got, moved, tc.moves)
			}
		})
	}
}

func TestNothingMovesWhileEveryTunnelIsDown(t *testing.T) {
	m := New("a")
	round := []Reading{down("a"), down("b")}
	if got := run(m, [][]Reading{round, round, round, round}); !equal(got, []string{"", "", "", ""}) {
		t.Fatalf("decisions = %q, want none", got)
	}
}

func TestEqualLatencyKeepsThePreferredOrder(t *testing.T) {
	m := New("c")
	round := []Reading{up("a", 100), up("b", 100), down("c")}
	got := run(m, [][]Reading{round, round, round})
	if got[2] != "a" {
		t.Fatalf("decisions = %q, want a (first in the pool) on the third round", got)
	}
}

func TestAnActiveTunnelThatLeftThePoolIsReplacedLikeADownOne(t *testing.T) {
	m := New("gone")
	round := []Reading{up("a", 100), up("b", 200)}
	got := run(m, [][]Reading{round, round, round})
	if got[2] != "a" {
		t.Fatalf("decisions = %q, want a on the third round", got)
	}
}

func TestATunnelWithoutAMeasuredLatencyOnlyTakesOverFromADownOne(t *testing.T) {
	round := []Reading{up("a", 300), {Name: "b", Up: true}}
	m := New("a")
	if got := run(m, [][]Reading{round, round, round, round}); !equal(got, []string{"", "", "", ""}) {
		t.Fatalf("a working tunnel was left for one with no measurement: %q", got)
	}

	failed := []Reading{down("a"), {Name: "b", Up: true}, up("c", 200)}
	m = New("a")
	if got := run(m, [][]Reading{failed, failed, failed}); got[2] != "c" {
		t.Fatalf("decisions = %q, want the measured tunnel c to win over the unmeasured b", got)
	}
}
