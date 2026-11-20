package service

import (
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

func TestPeriodStart(t *testing.T) {
	utc := func(y int, m time.Month, d, h int) time.Time { return time.Date(y, m, d, h, 0, 0, 0, time.UTC) }
	tests := []struct {
		name string
		now  time.Time
		day  int
		want time.Time
	}{
		{"after this month's reset", utc(2026, 11, 20, 9), 15, utc(2026, 11, 15, 0)},
		{"before this month's reset", utc(2026, 11, 10, 9), 15, utc(2026, 10, 15, 0)},
		{"reset day past a short month", utc(2026, 3, 10, 9), 31, utc(2026, 2, 28, 0)},
		{"on the reset day itself", utc(2026, 11, 1, 0), 1, utc(2026, 11, 1, 0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PeriodStart(tt.now, tt.day); !got.Equal(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAdvanceTrafficPeriod(t *testing.T) {
	nov := time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC)
	p := AdvanceTrafficPeriod(model.TrafficPeriod{}, 5000, nov, 1)
	if p.Bytes != 0 || p.LastCounter != 5000 {
		t.Fatalf("first sample = %+v, want a baseline with no bytes counted", p)
	}
	p = AdvanceTrafficPeriod(p, 8000, nov.Add(time.Minute), 1)
	if p.Bytes != 3000 {
		t.Fatalf("growth = %d, want 3000", p.Bytes)
	}
	p = AdvanceTrafficPeriod(p, 700, nov.Add(2*time.Minute), 1)
	if p.Bytes != 3700 {
		t.Fatalf("after a reboot = %d, want 3700 (the new counter counts from zero)", p.Bytes)
	}
	p = AdvanceTrafficPeriod(p, 900, time.Date(2026, 12, 1, 0, 5, 0, 0, time.UTC), 1)
	if p.Bytes != 200 {
		t.Fatalf("new month = %d, want only this month's 200", p.Bytes)
	}
}
