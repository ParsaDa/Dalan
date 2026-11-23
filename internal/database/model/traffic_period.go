package model

// TrafficPeriod counts this server's network bytes for the current billing month.
// LastCounter is the OS counter at the previous sample, so a reboot (counter reset) adds nothing twice.
type TrafficPeriod struct {
	Id          int    `json:"-" gorm:"primaryKey"`
	PeriodStart int64  `json:"periodStart"`
	Bytes       uint64 `json:"bytes"`
	LastCounter uint64 `json:"-"`
}
