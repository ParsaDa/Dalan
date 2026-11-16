package model

// NodeEnrolment is one invitation for a node abroad to dial into this panel, and afterwards the
// record tying that node to its management tunnel. Its secrets are cleared once it is used.
type NodeEnrolment struct {
	Id         int    `json:"id" gorm:"primaryKey;autoIncrement" example:"1"`
	Name       string `json:"name" gorm:"uniqueIndex" example:"de1"`
	PublicId   string `json:"-" gorm:"column:public_id;index"`
	TokenHash  string `json:"-" gorm:"column:token_hash"`
	PrivateKey string `json:"-" gorm:"column:private_key"`
	ExpiresAt  int64  `json:"expiresAt" gorm:"column:expires_at" example:"1700001800"`
	UsedAt     int64  `json:"usedAt" gorm:"column:used_at;default:0" example:"0"`
	TunnelId   int    `json:"tunnelId" gorm:"column:tunnel_id" example:"3"`
	NodeId     int    `json:"nodeId" gorm:"column:node_id;default:0" example:"0"`
	// DataTunnelId is the ordinary entry tunnel made for the node's traffic; 0 when none was asked for.
	DataTunnelId int `json:"dataTunnelId" gorm:"column:data_tunnel_id;default:0" example:"0"`
}
