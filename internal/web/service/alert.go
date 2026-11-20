package service

import (
	"fmt"
	"sort"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

// Load alerts open at alertLoadOpen percent and close only below alertLoadClear, so a
// value hovering at the line does not open and resolve an alert every sample.
const (
	alertLoadOpen  = 85.0
	alertLoadClear = 80.0
	alertLocal     = "local"
	expiringWindow = 7 * 24 * time.Hour
)

// AlertCondition is one condition that holds right now.
type AlertCondition struct {
	Kind    string
	Subject string
	NodeId  int
	Value   float64
}

// AlertSnapshot is everything the evaluator looks at in one pass.
type AlertSnapshot struct {
	CPUPercent      float64
	HaveCPU         bool
	MemPercent      float64
	HaveMem         bool
	XrayRunning     bool
	HaveXray        bool
	Nodes           []*model.Node
	ExpiringClients int
	HaveClients     bool
}

type AlertService struct{}

func alertKey(kind, subject string) string { return kind + "\x00" + subject }

func nodeSubject(id int) string { return fmt.Sprintf("node:%d", id) }

// DesiredAlerts turns a snapshot into the conditions that should be open, using the
// alerts already open for hysteresis. It also returns the kinds the snapshot covered.
func DesiredAlerts(snap AlertSnapshot, open map[string]bool) ([]AlertCondition, []string) {
	var out []AlertCondition
	var kinds []string
	load := func(kind, subject string, nodeId int, pct float64) {
		line := alertLoadOpen
		if open[alertKey(kind, subject)] {
			line = alertLoadClear
		}
		if pct >= line {
			out = append(out, AlertCondition{Kind: kind, Subject: subject, NodeId: nodeId, Value: pct})
		}
	}
	if snap.HaveCPU {
		kinds = append(kinds, model.AlertCPUHigh)
		load(model.AlertCPUHigh, alertLocal, 0, snap.CPUPercent)
	}
	if snap.HaveMem {
		kinds = append(kinds, model.AlertMemoryHigh)
		load(model.AlertMemoryHigh, alertLocal, 0, snap.MemPercent)
	}
	if snap.HaveXray {
		kinds = append(kinds, model.AlertXrayDown)
		if !snap.XrayRunning {
			out = append(out, AlertCondition{Kind: model.AlertXrayDown, Subject: alertLocal})
		}
	}
	if snap.Nodes != nil {
		kinds = append(kinds, model.AlertNodeOffline, model.AlertNodeHighLoad)
		for _, n := range snap.Nodes {
			if n == nil || !n.Enable {
				continue
			}
			subject := nodeSubject(n.Id)
			switch n.Status {
			case "offline":
				out = append(out, AlertCondition{Kind: model.AlertNodeOffline, Subject: subject, NodeId: n.Id})
			case "online":
				load(model.AlertNodeHighLoad, subject, n.Id, max(n.CpuPct, n.MemPct))
			}
		}
	}
	if snap.HaveClients {
		kinds = append(kinds, model.AlertClientsExpiring)
		if snap.ExpiringClients > 0 {
			out = append(out, AlertCondition{Kind: model.AlertClientsExpiring, Subject: alertLocal, Value: float64(snap.ExpiringClients)})
		}
	}
	return out, kinds
}

// OpenKeys returns the Kind+Subject keys of every open alert.
func (s *AlertService) OpenKeys() (map[string]bool, error) {
	var rows []model.Alert
	if err := database.GetDB().Where("resolved_at = 0").Find(&rows).Error; err != nil {
		return nil, err
	}
	keys := make(map[string]bool, len(rows))
	for _, r := range rows {
		keys[alertKey(r.Kind, r.Subject)] = true
	}
	return keys, nil
}

// Reconcile makes the open alerts of the given kinds match observed: new conditions open
// a row, held ones refresh their value, and open rows no longer observed are resolved.
func (s *AlertService) Reconcile(kinds []string, observed []AlertCondition, now int64) error {
	if len(kinds) == 0 {
		return nil
	}
	db := database.GetDB()
	var open []model.Alert
	if err := db.Where("resolved_at = 0 AND kind IN ?", kinds).Find(&open).Error; err != nil {
		return err
	}
	byKey := make(map[string]*model.Alert, len(open))
	for i := range open {
		byKey[alertKey(open[i].Kind, open[i].Subject)] = &open[i]
	}
	seen := make(map[string]bool, len(observed))
	for _, c := range observed {
		k := alertKey(c.Kind, c.Subject)
		seen[k] = true
		if row, ok := byKey[k]; ok {
			if err := db.Model(&model.Alert{}).Where("id = ?", row.Id).
				Updates(map[string]any{"value": c.Value, "updated_at": now}).Error; err != nil {
				return err
			}
			continue
		}
		row := model.Alert{Kind: c.Kind, Subject: c.Subject, NodeId: c.NodeId, Value: c.Value, StartedAt: now, UpdatedAt: now}
		if err := db.Create(&row).Error; err != nil {
			return err
		}
	}
	for k, row := range byKey {
		if seen[k] {
			continue
		}
		if err := db.Model(&model.Alert{}).Where("id = ?", row.Id).
			Updates(map[string]any{"resolved_at": now, "updated_at": now}).Error; err != nil {
			return err
		}
	}
	return nil
}

// Apply evaluates a snapshot against the open alerts and reconciles them.
func (s *AlertService) Apply(snap AlertSnapshot, now int64) error {
	open, err := s.OpenKeys()
	if err != nil {
		return err
	}
	observed, kinds := DesiredAlerts(snap, open)
	return s.Reconcile(kinds, observed, now)
}

var alertSeverity = map[string]int{
	model.AlertNodeOffline:     0,
	model.AlertXrayDown:        0,
	model.AlertNodeHighLoad:    1,
	model.AlertCPUHigh:         1,
	model.AlertMemoryHigh:      1,
	model.AlertClientsExpiring: 2,
}

// ListOpen returns open alerts, the most serious first, then the oldest first.
func (s *AlertService) ListOpen() ([]model.Alert, error) {
	var rows []model.Alert
	if err := database.GetDB().Where("resolved_at = 0").Order("started_at asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	sort.SliceStable(rows, func(i, j int) bool {
		return alertSeverity[rows[i].Kind] < alertSeverity[rows[j].Kind]
	})
	return rows, nil
}

// CountExpiringClients counts enabled clients whose fixed expiry falls within the next week.
func (s *AlertService) CountExpiringClients(now time.Time) (int, error) {
	from := now.UnixMilli()
	to := now.Add(expiringWindow).UnixMilli()
	var n int64
	err := database.GetDB().Model(&model.ClientRecord{}).
		Where("enable = ? AND expiry_time > ? AND expiry_time <= ?", true, from, to).
		Count(&n).Error
	return int(n), err
}
