package service

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/dbtest"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/tor"
)

func newTorDB(t *testing.T) *TorService {
	t.Helper()
	dbtest.InitDB(t, filepath.Join(t.TempDir(), "x-ui.db"))
	return &TorService{}
}

func TestAddAssignsPortsInRange(t *testing.T) {
	svc := newTorDB(t)
	db := database.GetDB()

	first, err := svc.Add("DE")
	if err != nil {
		t.Fatalf("Add(DE): %v", err)
	}
	second, err := svc.Add("nl")
	if err != nil {
		t.Fatalf("Add(nl): %v", err)
	}
	if first.Country != "de" || first.Port != tor.PortBase || second.Port != tor.PortBase+1 {
		t.Fatalf("got %s:%d and %s:%d, want de:%d and nl:%d", first.Country, first.Port, second.Country, second.Port, tor.PortBase, tor.PortBase+1)
	}

	if err := db.Create(&model.ServerTunnel{Name: "t1", Role: model.TunnelRolePortal, RelayPort: tor.PortBase + 2}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Inbound{Port: tor.PortBase + 3, Tag: "in-taken", Protocol: model.VLESS}).Error; err != nil {
		t.Fatal(err)
	}
	third, err := svc.Add("fr")
	if err != nil {
		t.Fatalf("Add(fr): %v", err)
	}
	if third.Port != tor.PortBase+4 {
		t.Fatalf("port = %d, want %d (the relay and the inbound ports are taken)", third.Port, tor.PortBase+4)
	}
}

func TestAddRefusesDuplicateAndBadCountry(t *testing.T) {
	svc := newTorDB(t)
	if _, err := svc.Add("de"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Add(" DE "); err == nil || !strings.Contains(err.Error(), "already") {
		t.Fatalf("second de: err = %v, want one saying it already exists", err)
	}
	if _, err := svc.Add("deu"); err == nil {
		t.Fatal("a three-letter country was accepted")
	}
}

func TestAddRefusesAnEleventhEnabledExit(t *testing.T) {
	svc := newTorDB(t)
	codes := []string{"de", "nl", "fr", "se", "ch", "at", "fi", "no", "dk", "pl"}
	var firstID int
	for i, cc := range codes {
		exit, err := svc.Add(cc)
		if err != nil {
			t.Fatalf("Add(%s): %v", cc, err)
		}
		if i == 0 {
			firstID = exit.Id
		}
	}
	if _, err := svc.Add("it"); err == nil || !strings.Contains(err.Error(), fmt.Sprint(tor.MaxExits)) {
		t.Fatalf("eleventh exit: err = %v, want one naming the limit of %d", err, tor.MaxExits)
	}
	if err := svc.SetEnable(firstID, false); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Add("it"); err != nil {
		t.Fatalf("with one exit disabled the eleventh must fit: %v", err)
	}
	if err := svc.SetEnable(firstID, true); err == nil {
		t.Fatal("re-enabling past the limit was accepted")
	}
}

func TestDeleteRefusedWhileRoutingNamesTheTag(t *testing.T) {
	cases := []struct {
		name     string
		template string
		panelOut string
		refused  string
	}{
		{"routing rule", `{"routing":{"rules":[{"type":"field","outboundTag":"tor-de","domain":["example.com"]}]}}`, "", "routing rule"},
		{"balancer selector", `{"routing":{"balancers":[{"tag":"b","selector":["proxy","tor-de"]}]}}`, "", "balancer"},
		{"panel egress", `{"routing":{}}`, "tor-de", "panel"},
		{"unrelated tag", `{"routing":{"rules":[{"type":"field","outboundTag":"tor-dex"}],"balancers":[{"tag":"b","selector":["tor-nl"]}]}}`, "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc := newTorDB(t)
			exit, err := svc.Add("de")
			if err != nil {
				t.Fatal(err)
			}
			settings := &SettingService{}
			if err := settings.saveSetting("xrayTemplateConfig", c.template); err != nil {
				t.Fatal(err)
			}
			if err := settings.SetPanelOutbound(c.panelOut); err != nil {
				t.Fatal(err)
			}

			err = svc.Delete(exit.Id)
			rows, listErr := svc.List()
			if listErr != nil {
				t.Fatal(listErr)
			}
			if c.refused == "" {
				if err != nil || len(rows) != 0 {
					t.Fatalf("delete of an unreferenced exit: err = %v, rows = %d", err, len(rows))
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), c.refused) || len(rows) != 1 {
				t.Fatalf("delete: err = %v, rows = %d; want a refusal mentioning %q and the exit kept", err, len(rows), c.refused)
			}
		})
	}
}

func TestBridgesAreValidatedBeforeTheyAreStored(t *testing.T) {
	svc := newTorDB(t)
	if err := svc.SetBridges("obfs4 192.0.2.4:443 AAA cert=x iat-mode=0\nExitPolicy accept *:*"); err == nil {
		t.Fatal("a directive was accepted as a bridge")
	}
	if got, _ := svc.Bridges(); got != "" {
		t.Fatalf("a refused save changed the stored bridges to %q", got)
	}
	if err := svc.SetBridges("  obfs4 192.0.2.4:443 AAA cert=x iat-mode=0  \n\n"); err != nil {
		t.Fatal(err)
	}
	if got, _ := svc.Bridges(); got != "obfs4 192.0.2.4:443 AAA cert=x iat-mode=0" {
		t.Fatalf("stored bridges = %q", got)
	}
}
