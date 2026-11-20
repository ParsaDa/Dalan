package model

// Alert is a condition that needs the admin's attention. It stays open while the
// condition holds and gets ResolvedAt when it clears; one open row per Kind+Subject.
type Alert struct {
	Id         int     `json:"id" gorm:"primaryKey;autoIncrement" example:"1"`
	Kind       string  `json:"kind" gorm:"index:idx_alert_open,priority:1" example:"node.offline"`
	Subject    string  `json:"subject" gorm:"index:idx_alert_open,priority:2" example:"node:3"`
	NodeId     int     `json:"nodeId" example:"3"`
	Value      float64 `json:"value" example:"91.5"`
	StartedAt  int64   `json:"startedAt" example:"1763600000"`
	UpdatedAt  int64   `json:"updatedAt" example:"1763600720"`
	ResolvedAt int64   `json:"resolvedAt" gorm:"index:idx_alert_open,priority:3" example:"0"`
}

const (
	AlertNodeOffline     = "node.offline"
	AlertNodeHighLoad    = "node.load"
	AlertCPUHigh         = "cpu.high"
	AlertMemoryHigh      = "memory.high"
	AlertXrayDown        = "xray.down"
	AlertClientsExpiring = "clients.expiring"
)
