package model

// Activity is one change made through the panel, kept for the activity log.
// NodeId 0 is this server; Actor is "you", "api" or "system".
type Activity struct {
	Id     int    `json:"id" gorm:"primaryKey;autoIncrement" example:"1"`
	At     int64  `json:"at" gorm:"index" example:"1763600000"`
	Actor  string `json:"actor" example:"you"`
	Action string `json:"action" gorm:"index" example:"client.add"`
	Target string `json:"target" example:"sara.k"`
	NodeId int    `json:"nodeId" gorm:"index" example:"0"`
}

const (
	ActorYou    = "you"
	ActorAPI    = "api"
	ActorSystem = "system"
)
