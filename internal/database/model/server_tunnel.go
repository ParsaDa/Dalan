package model

const (
	TunnelRolePortal = "portal"
	TunnelRoleBridge = "bridge"

	TunnelEngineXray = "xray"

	TunnelSecurityNone    = "none"
	TunnelSecurityReality = "reality"
)

// ServerTunnel is one end of a tunnel between two panels. The portal listens and
// takes user traffic; the bridge dials the portal and is where traffic leaves.
type ServerTunnel struct {
	Id      int    `json:"id" gorm:"primaryKey;autoIncrement" example:"1"`
	Name    string `json:"name" gorm:"uniqueIndex" example:"de1"`
	Role    string `json:"role" example:"portal"`
	Engine  string `json:"engine" gorm:"default:xray" example:"xray"`
	Enable  bool   `json:"enable" gorm:"default:true" example:"true"`
	Address string `json:"address" example:"203.0.113.10"`
	Port    int    `json:"port" example:"8443"`
	UUID    string `json:"uuid" gorm:"column:uuid" example:"8d6f3a0e-5c1b-4f0e-9a51-2f6d1c7b9e10"`

	Security          string `json:"security" gorm:"default:reality" example:"reality"`
	ServerName        string `json:"serverName" gorm:"column:server_name" example:"www.example.com"`
	RealityPrivateKey string `json:"-" gorm:"column:reality_private_key"`
	RealityPublicKey  string `json:"realityPublicKey" gorm:"column:reality_public_key"`
	ShortId           string `json:"shortId" gorm:"column:short_id" example:"a1b2c3d4"`

	// RouteInbounds (portal) lists the local inbound tags whose traffic leaves through this tunnel.
	RouteInbounds []string `json:"routeInbounds" gorm:"serializer:json;column:route_inbounds"`
	// ExitOutbound (bridge) is the local outbound tunnel traffic leaves from; empty means "direct".
	ExitOutbound string `json:"exitOutbound" gorm:"column:exit_outbound" example:"direct"`
}
