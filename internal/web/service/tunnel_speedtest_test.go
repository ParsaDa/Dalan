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

func TestSpeedtestLinkPointsAtTheExitEndsResponderBesideTheRelayPort(t *testing.T) {
	svc := newTunnelDB(t)
	good := storeTunnel(t, model.ServerTunnel{Name: "de1", Role: model.TunnelRolePortal, Engine: model.TunnelEngineBackpack, Enable: true, RelayPort: 63500, UUID: "u"})

	dial, addr, err := svc.SpeedtestLink(good.Id)
	if err != nil || dial == nil {
		t.Fatalf("dial = %v, err = %v", dial != nil, err)
	}
	if addr != "127.0.0.1:64500" {
		t.Fatalf("addr = %q, want the loopback port 1000 above the relay port", addr)
	}
	if _, _, err := svc.SpeedtestLink(9999); err == nil {
		t.Fatal("a missing tunnel must be refused")
	}
}

func TestExitSpeedtestAddrsAreTheEnabledExitEndsWithARelay(t *testing.T) {
	tunnels := []model.ServerTunnel{
		{Name: "exit", Role: model.TunnelRoleBridge, Engine: model.TunnelEngineBackpack, Enable: true, RelayPort: 63501},
		{Name: "entry", Role: model.TunnelRolePortal, Engine: model.TunnelEngineBackpack, Enable: true, RelayPort: 63502},
		{Name: "off", Role: model.TunnelRoleBridge, Engine: model.TunnelEngineBackpack, Enable: false, RelayPort: 63503},
		{Name: "norelay", Role: model.TunnelRoleBridge, Engine: model.TunnelEngineBackpack, Enable: true},
		{Name: "child", Role: model.TunnelRoleBridge, Engine: model.TunnelEngineBackpack, Enable: true, RelayPort: 63504, ParentId: 7},
		{Name: "exit2", Role: model.TunnelRoleBridge, Engine: model.TunnelEngineRakhsh, Enable: true, RelayPort: 63505},
	}

	got := exitSpeedtestAddrs(tunnels)

	want := []string{"127.0.0.1:64501", "127.0.0.1:64505"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("addrs = %v, want %v", got, want)
	}
}
