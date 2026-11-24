package service

import (
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

func storeTunnel(t *testing.T, tun model.ServerTunnel) model.ServerTunnel {
	t.Helper()
	if err := database.GetDB().Create(&tun).Error; err != nil {
		t.Fatal(err)
	}
	return tun
}

func TestSpeedtestDialAcceptsOnlyAnEnabledEntryTunnelWithARelay(t *testing.T) {
	svc := newTunnelDB(t)
	good := storeTunnel(t, model.ServerTunnel{Name: "de1", Role: model.TunnelRolePortal, Engine: model.TunnelEngineBackpack, Enable: true, RelayPort: 63500, UUID: "u"})
	exit := storeTunnel(t, model.ServerTunnel{Name: "de2", Role: model.TunnelRoleBridge, Engine: model.TunnelEngineBackpack, Enable: true, RelayPort: 63501, UUID: "u"})
	direct := storeTunnel(t, model.ServerTunnel{Name: "de3", Role: model.TunnelRolePortal, Engine: model.TunnelEngineBackpack, Enable: true, Mode: model.TunnelModeDirect, UUID: "u"})
	managed := storeTunnel(t, model.ServerTunnel{Name: "de4", Role: model.TunnelRolePortal, Engine: model.TunnelEngineBackpack, Enable: true, RelayPort: 63502, ManagePort: 61001, UUID: "u"})
	off := storeTunnel(t, model.ServerTunnel{Name: "de5", Role: model.TunnelRolePortal, Engine: model.TunnelEngineBackpack, RelayPort: 63503, UUID: "u"})
	if err := database.GetDB().Model(&off).Update("enable", false).Error; err != nil {
		t.Fatal(err)
	}

	if dial, err := svc.SpeedtestDial(good.Id); err != nil || dial == nil {
		t.Fatalf("entry tunnel with a relay: dial = %v, err = %v", dial != nil, err)
	}
	for name, tc := range map[string]struct {
		id   int
		want string
	}{
		"exit end":          {exit.Id, "exit end"},
		"direct tunnel":     {direct.Id, "no relay"},
		"management tunnel": {managed.Id, "management"},
		"disabled":          {off.Id, "disabled"},
		"missing":           {9999, "does not exist"},
	} {
		if _, err := svc.SpeedtestDial(tc.id); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err = %v, want it to mention %q", name, err, tc.want)
		}
	}
}
