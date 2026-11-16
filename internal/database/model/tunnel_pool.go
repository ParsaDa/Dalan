package model

// TunnelPool is a set of entry tunnels that stand in for each other: traffic leaves through the
// active one, and the panel moves it to a spare when that one stays worse than the spare.
type TunnelPool struct {
	Id   int    `json:"id" gorm:"primaryKey;autoIncrement" example:"1"`
	Name string `json:"name" gorm:"uniqueIndex" example:"main"`
	// Members are tunnel names in order of preference.
	Members []string `json:"members" gorm:"serializer:json;column:members"`
	// RouteInbounds lists the local inbound tags whose traffic leaves through the pool.
	RouteInbounds []string `json:"routeInbounds" gorm:"serializer:json;column:route_inbounds"`
	// Active is the member carrying the traffic now; empty means the first working member.
	Active string `json:"active" example:"de1"`
}
