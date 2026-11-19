package tor

import "testing"

func TestParseBootstrap(t *testing.T) {
	cases := []struct {
		line string
		pct  int
		ok   bool
	}{
		{"Nov 17 09:30:01.000 [notice] Bootstrapped 45% (loading_descriptors): Loading relay descriptors", 45, true},
		{"Bootstrapped 100% (done): Done", 100, true},
		{"Bootstrapped 0% (starting): Starting", 0, true},
		{"[notice] Opening Socks listener on 127.0.0.1:62700", 0, false},
		{"Bootstrapped lots", 0, false},
		{"", 0, false},
	}
	for _, c := range cases {
		t.Run(c.line, func(t *testing.T) {
			pct, ok := ParseBootstrap(c.line)
			if pct != c.pct || ok != c.ok {
				t.Fatalf("ParseBootstrap(%q) = %d, %v; want %d, %v", c.line, pct, ok, c.pct, c.ok)
			}
		})
	}
}

func TestProgressKeepsEachCountrySeparateAndForgetsOnReset(t *testing.T) {
	var p Progress
	if _, ok := p.Get("de"); ok {
		t.Fatal("an empty Progress knows de")
	}
	p.Set("de", 40)
	p.Set("nl", 100)
	if got, _ := p.Get("de"); got != 40 {
		t.Fatalf("de = %d, want 40", got)
	}
	p.Reset("de")
	if _, ok := p.Get("de"); ok {
		t.Fatal("de survived Reset")
	}
	if got, ok := p.Get("nl"); !ok || got != 100 {
		t.Fatalf("Reset of de changed nl to %d, %v", got, ok)
	}
}
