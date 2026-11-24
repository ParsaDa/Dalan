package model

const (
	TunnelRolePortal = "portal"
	TunnelRoleBridge = "bridge"

	TunnelEngineBackpack = "backpack"
	TunnelEngineRakhsh   = "rakhsh"
	TunnelEngineGost     = "gost"
	TunnelEngineRathole  = "rathole"
	TunnelEngineBackhaul = "backhaul"
	TunnelEngineSingbox  = "singbox"
	TunnelEngineXrayCore = "xraycore"
	TunnelEngineKariz    = "kariz"
	TunnelEngineCarrier  = "carrier"

	TunnelModeReverse = "reverse"
	TunnelModeDirect  = "direct"
)

// ExternalEngine reports whether an engine runs as its own process and reaches Xray through a loopback relay.
func ExternalEngine(engine string) bool {
	switch engine {
	case TunnelEngineBackpack, TunnelEngineRakhsh, TunnelEngineGost, TunnelEngineRathole, TunnelEngineBackhaul,
		TunnelEngineSingbox, TunnelEngineXrayCore, TunnelEngineKariz, TunnelEngineCarrier:
		return true
	}
	return false
}

// ServerTunnel is one end of a tunnel between two panels. The portal listens and
// takes user traffic; the bridge dials the portal and is where traffic leaves.
type ServerTunnel struct {
	Id      int    `json:"id" gorm:"primaryKey;autoIncrement" example:"1"`
	Name    string `json:"name" gorm:"uniqueIndex" example:"de1"`
	Role    string `json:"role" example:"portal"`
	Engine  string `json:"engine" gorm:"default:backpack" example:"backpack"`
	Enable  bool   `json:"enable" gorm:"default:true" example:"true"`
	Address string `json:"address" example:"203.0.113.10"`
	Port    int    `json:"port" example:"8443"`
	UUID    string `json:"uuid" gorm:"column:uuid" example:"8d6f3a0e-5c1b-4f0e-9a51-2f6d1c7b9e10"`

	// Transport, FallbackTransports, ForwardPorts and AcceptUDP belong to the BackPack engine.
	Transport          string   `json:"transport" example:"wssmux"`
	FallbackTransports []string `json:"fallbackTransports" gorm:"serializer:json;column:fallback_transports"`
	// ForwardPorts (portal) are ports opened on this server and carried to the other end, like "443=127.0.0.1:2096".
	ForwardPorts []string `json:"forwardPorts" gorm:"serializer:json;column:forward_ports"`
	AcceptUDP    bool     `json:"acceptUdp" gorm:"column:accept_udp"`
	// RelayPort is the loopback port both ends of a BackPack tunnel use to carry Xray traffic; zero means none.
	RelayPort int `json:"relayPort" gorm:"column:relay_port;default:0" example:"63500"`

	// Mode is "reverse" (the exit server dials the entry server) or "direct" (the entry server dials the
	// exit server, a BackPack layer-3 link). An empty value reads as reverse.
	Mode string `json:"mode" gorm:"default:reverse" example:"reverse"`
	// Options are BackPack settings beyond the fields above, keyed as the engine spells them; each end
	// writes the ones that apply to its role.
	Options map[string]any `json:"options" gorm:"serializer:json;column:options"`

	// RakhshSecret is this end's private key for the Rakhsh handshake; RakhshPeerPublic pins the other end's.
	RakhshSecret     string `json:"-" gorm:"column:rakhsh_secret"`
	RakhshPeerPublic string `json:"rakhshPeerPublic" gorm:"column:rakhsh_peer_public" example:"MzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzMzM="`

	// EngineSecret is the token or password both ends of a GOST, Rathole, Backhaul, sing-box, Xray or
	// Kariz tunnel share; the setup link carries it.
	EngineSecret string `json:"-" gorm:"column:engine_secret"`

	// RouteInbounds (portal) lists the local inbound tags whose traffic leaves through this tunnel.
	RouteInbounds []string `json:"routeInbounds" gorm:"serializer:json;column:route_inbounds"`
	// ExitOutbound (bridge) is the local outbound tunnel traffic leaves from; empty means "direct".
	ExitOutbound string `json:"exitOutbound" gorm:"column:exit_outbound" example:"direct"`

	// ManagePort marks a management tunnel. Portal: the loopback port opened here. Bridge: this
	// server's panel port, the only place the tunnel's traffic may go. Zero is an ordinary tunnel.
	ManagePort int `json:"managePort" gorm:"column:manage_port;default:0" example:"0"`
	// ParentId (management portal) is the tunnel whose port and REALITY identity this channel shares.
	ParentId int `json:"parentId" gorm:"column:parent_id;default:0" example:"0"`
	// NodeId (portal) names the node whose panel holds the other end, set when this panel created both ends.
	NodeId int `json:"nodeId" gorm:"column:node_id;default:0" example:"0"`
	// Up and Down are bytes that left and came back through the entry outbound; read from the stats, never stored.
	Up   int64 `json:"up" gorm:"-" example:"1048576"`
	Down int64 `json:"down" gorm:"-" example:"5242880"`
	// ManageTarget is the node panel port a BackPack parent delivers its management forward to.
	ManageTarget int `json:"-" gorm:"column:manage_target;default:0"`
}
