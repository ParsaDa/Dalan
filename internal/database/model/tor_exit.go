package model

// TorExit is one country Tor leaves through: Dalan runs a Tor for it on a loopback SOCKS port that Xray
// reaches as the outbound "tor-<country>".
type TorExit struct {
	Id      int    `json:"id" gorm:"primaryKey;autoIncrement" example:"1"`
	Country string `json:"country" gorm:"uniqueIndex" example:"de"`
	Port    int    `json:"port" example:"62700"`
	Enable  bool   `json:"enable" gorm:"default:true" example:"true"`
}
