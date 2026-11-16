package service

import (
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

func TestTheFrontDoorTableFollowsThePanelSettingsAndTheStoredInbounds(t *testing.T) {
	newTunnelDB(t)
	t.Setenv("XUI_PORT", "20530")
	previous := frontDoorTTL
	frontDoorTTL = 0
	t.Cleanup(func() { frontDoorTTL = previous })

	var door FrontDoorService
	if err := door.settings.setString("subPath", "/mysub/"); err != nil {
		t.Fatalf("set subPath: %v", err)
	}
	if err := door.settings.SetBasePath("/secret/"); err != nil {
		t.Fatalf("SetBasePath: %v", err)
	}
	table := door.Table()
	if table.PanelPort != 20530 || table.PanelPath != "/secret/" {
		t.Fatalf("panel = %d %q, want the XUI_PORT override on the panel's base path", table.PanelPort, table.PanelPath)
	}
	if table.SubPort != 2096 || len(table.SubPaths) == 0 || table.SubPaths[0] != "/mysub/" {
		t.Fatalf("subscription = %d %v, want the default port and the stored path", table.SubPort, table.SubPaths)
	}
	if len(table.Inbounds) != 0 {
		t.Fatalf("inbounds = %+v before any exists", table.Inbounds)
	}

	in := model.Inbound{
		Enable: true, Port: 20868, Protocol: model.VLESS, Tag: "in-20868",
		Settings:       `{"clients":[]}`,
		StreamSettings: `{"network":"ws","security":"none","wsSettings":{"path":"/zedone"}}`,
	}
	if err := database.GetDB().Create(&in).Error; err != nil {
		t.Fatalf("create inbound: %v", err)
	}
	table = door.Table()
	if len(table.Inbounds) != 1 || table.Inbounds[0].Prefix != "/zedone" || table.Inbounds[0].Port != 20868 {
		t.Fatalf("inbounds = %+v, want the WebSocket inbound on /zedone", table.Inbounds)
	}
}
