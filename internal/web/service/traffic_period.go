package service

import (
	"errors"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/net"

	"gorm.io/gorm"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

// PeriodStart is the most recent reset boundary at or before now, in now's zone.
// A reset day past the month's end (31 in February) falls on its last day.
func PeriodStart(now time.Time, resetDay int) time.Time {
	if resetDay < 1 {
		resetDay = 1
	}
	at := func(y int, m time.Month) time.Time {
		last := time.Date(y, m+1, 0, 0, 0, 0, 0, now.Location()).Day()
		return time.Date(y, m, min(resetDay, last), 0, 0, 0, 0, now.Location())
	}
	start := at(now.Year(), now.Month())
	if start.After(now) {
		start = at(now.Year(), now.Month()-1)
	}
	return start
}

// AdvanceTrafficPeriod folds one OS byte-counter sample into the period. A counter that
// went backwards means the OS restarted, so the new counter itself is the traffic since.
func AdvanceTrafficPeriod(p model.TrafficPeriod, counter uint64, now time.Time, resetDay int) model.TrafficPeriod {
	start := PeriodStart(now, resetDay).Unix()
	if p.PeriodStart != start {
		p.PeriodStart = start
		p.Bytes = 0
	}
	switch {
	case p.LastCounter == 0:
	case counter >= p.LastCounter:
		p.Bytes += counter - p.LastCounter
	default:
		p.Bytes += counter
	}
	p.LastCounter = counter
	return p
}

type TrafficPeriodService struct {
	settingService SettingService
}

// Sample records one counter reading and returns the updated period.
func (s *TrafficPeriodService) Sample(counter uint64, now time.Time) (model.TrafficPeriod, error) {
	_, day, err := s.settingService.GetTrafficCap()
	if err != nil {
		return model.TrafficPeriod{}, err
	}
	db := database.GetDB()
	var p model.TrafficPeriod
	if err := db.First(&p, 1).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return p, err
	}
	p = AdvanceTrafficPeriod(p, counter, now, day)
	p.Id = 1
	return p, db.Save(&p).Error
}

// TrafficUsage is this server's traffic against its monthly plan.
type TrafficUsage struct {
	CapGB       int    `json:"capGB" example:"10000"`
	ResetDay    int    `json:"resetDay" example:"1"`
	Bytes       uint64 `json:"bytes" example:"6300000000000"`
	PeriodStart int64  `json:"periodStart" example:"1761955200"`
	NextReset   int64  `json:"nextReset" example:"1764547200"`
}

func (s *TrafficPeriodService) Usage(now time.Time) (TrafficUsage, error) {
	capGB, day, err := s.settingService.GetTrafficCap()
	if err != nil {
		return TrafficUsage{}, err
	}
	var p model.TrafficPeriod
	if err := database.GetDB().First(&p, 1).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return TrafficUsage{}, err
	}
	start := PeriodStart(now, day)
	if p.PeriodStart != start.Unix() {
		p.Bytes = 0
	}
	next := PeriodStart(start.AddDate(0, 1, 7), day)
	return TrafficUsage{CapGB: capGB, ResetDay: day, Bytes: p.Bytes, PeriodStart: start.Unix(), NextReset: next.Unix()}, nil
}

// PhysicalNetBytes sums bytes sent and received on real interfaces, as the status card does.
func PhysicalNetBytes() (uint64, error) {
	stats, err := net.IOCounters(true)
	if err != nil {
		return 0, err
	}
	var total uint64
	for _, iface := range stats {
		if isVirtualInterface(strings.ToLower(iface.Name)) {
			continue
		}
		total += iface.BytesSent + iface.BytesRecv
	}
	return total, nil
}
