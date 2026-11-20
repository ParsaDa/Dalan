package enrolment

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/dbtest"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/enrol"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service/panel"
)

func newDB(t *testing.T) {
	t.Helper()
	dbtest.InitDB(t, filepath.Join(t.TempDir(), "x-ui.db"))
}

func enrolmentSpec(name string, port int) Spec {
	return Spec{
		Name: name, PanelURL: "http://203.0.113.10:2053/", Address: "203.0.113.10", Port: port,
	}
}

func createEnrolment(t *testing.T, svc *Service, name string, port int) (*Invitation, enrol.Link) {
	t.Helper()
	created, err := svc.Create(enrolmentSpec(name, port))
	if err != nil {
		t.Fatalf("Create(%s): %v", name, err)
	}
	link, err := enrol.ParseLink(created.Link)
	if err != nil {
		t.Fatalf("ParseLink: %v", err)
	}
	return created, link
}

func redeem(t *testing.T, svc *Service, link enrol.Link, info enrol.NodeInfo) (enrol.Grant, error) {
	t.Helper()
	body, openReply, err := link.Seal(info)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	reply, err := svc.Redeem(body, "198.51.100.9")
	if err != nil {
		return enrol.Grant{}, err
	}
	grant, err := openReply(reply)
	if err != nil {
		t.Fatalf("open reply: %v", err)
	}
	return grant, nil
}

func nodeCount(t *testing.T) int64 {
	t.Helper()
	var n int64
	if err := database.GetDB().Model(&model.Node{}).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func tunnelByName(t *testing.T, name string) *model.ServerTunnel {
	t.Helper()
	var rows []model.ServerTunnel
	if err := database.GetDB().Where("name = ?", name).Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		return nil
	}
	return &rows[0]
}

func TestRedeemingAnEnrolmentRegistersTheNodeOnItsManagementPort(t *testing.T) {
	newDB(t)
	svc := &Service{}
	created, link := createEnrolment(t, svc, "de1", 8443)

	portal := tunnelByName(t, "node-de1")
	if portal == nil || portal.Role != model.TunnelRolePortal || portal.ManagePort == 0 || portal.Port != 8443 {
		t.Fatalf("management tunnel after Create = %+v, want a portal on 8443 with a management port", portal)
	}
	data := tunnelByName(t, "de1")
	if !strings.Contains(created.Command, created.Link) {
		t.Fatalf("command %q does not carry the link", created.Command)
	}
	wantInstall := "XUI_ENROL='" + created.Link + "' bash <(curl -Ls https://raw.githubusercontent.com/ParsaDa/dalan/main/install.sh)"
	if created.InstallCommand != wantInstall {
		t.Fatalf("install command = %q, want %q", created.InstallCommand, wantInstall)
	}

	grant, err := redeem(t, svc, link, enrol.NodeInfo{
		Token: link.Token, Port: 2053, BasePath: "/x/", Scheme: "https", APIToken: "node-token-1",
	})
	if err != nil {
		t.Fatalf("Redeem: %v", err)
	}
	bridge, err := tunnel.ParseSetupLink(grant.DataTunnel)
	if err != nil {
		t.Fatalf("granted tunnel link: %v", err)
	}
	if bridge.UUID != data.UUID || bridge.Port != 8443 || bridge.Address != "203.0.113.10" {
		t.Fatalf("granted bridge %+v does not dial the node's tunnel %+v", bridge, data)
	}

	var row model.Node
	if err := database.GetDB().Where("name = ?", "de1").First(&row).Error; err != nil {
		t.Fatalf("node row: %v", err)
	}
	node, err := (&service.NodeService{}).GetById(row.Id)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if node.Address != "127.0.0.1" || node.Port != portal.ManagePort || !node.AllowPrivateAddress {
		t.Fatalf("node dials %s:%d private=%v, want 127.0.0.1:%d with private addresses allowed",
			node.Address, node.Port, node.AllowPrivateAddress, portal.ManagePort)
	}
	if node.PublicAddress != "198.51.100.9" {
		t.Fatalf("PublicAddress = %q, want the address the node enrolled from, 198.51.100.9", node.PublicAddress)
	}
	if node.ApiToken != "node-token-1" || node.BasePath != "/x/" || node.Scheme != "https" || node.TlsVerifyMode != "skip" || !node.Enable {
		t.Fatalf("node = token %q path %q scheme %q tls %q enable %v", node.ApiToken, node.BasePath, node.Scheme, node.TlsVerifyMode, node.Enable)
	}
}

func TestAnEnrolmentLinkWorksOnceAndOnlyWithItsToken(t *testing.T) {
	newDB(t)
	svc := &Service{}
	_, link := createEnrolment(t, svc, "de1", 8443)
	info := enrol.NodeInfo{Token: "not-the-token", Port: 2053, BasePath: "/", Scheme: "http", APIToken: "tok"}

	if _, err := redeem(t, svc, link, info); !errors.Is(err, ErrGone) {
		t.Fatalf("wrong token: err = %v, want ErrGone", err)
	}
	if n := nodeCount(t); n != 0 {
		t.Fatalf("a wrong token registered %d node(s)", n)
	}
	info.Token = link.Token
	if _, err := redeem(t, svc, link, info); err != nil {
		t.Fatalf("first redemption: %v", err)
	}
	if _, err := redeem(t, svc, link, info); !errors.Is(err, ErrGone) {
		t.Fatalf("second redemption: err = %v, want ErrGone", err)
	}
	if n := nodeCount(t); n != 1 {
		t.Fatalf("%d nodes after two redemptions, want 1", n)
	}
}

func TestAnExpiredEnrolmentIsRefusedAndItsTunnelRemoved(t *testing.T) {
	newDB(t)
	svc := &Service{}
	created, link := createEnrolment(t, svc, "de1", 8443)
	past := time.Now().Add(-time.Minute).Unix()
	if err := database.GetDB().Model(&model.NodeEnrolment{}).Where("id = ?", created.Id).Update("expires_at", past).Error; err != nil {
		t.Fatal(err)
	}
	_, err := redeem(t, svc, link, enrol.NodeInfo{Token: link.Token, Port: 2053, BasePath: "/", Scheme: "http", APIToken: "tok"})
	if !errors.Is(err, ErrGone) {
		t.Fatalf("expired link: err = %v, want ErrGone", err)
	}
	if n := nodeCount(t); n != 0 {
		t.Fatalf("an expired link registered %d node(s)", n)
	}
	if tunnelByName(t, "node-de1") != nil || tunnelByName(t, "de1") != nil {
		t.Fatal("the expired enrolment's tunnel still listens")
	}
}

func TestDeletingANodeOrAPendingEnrolmentRemovesItsManagementTunnel(t *testing.T) {
	newDB(t)
	svc := &Service{}
	_, link := createEnrolment(t, svc, "de1", 8443)
	pending, _ := createEnrolment(t, svc, "nl1", 8444)

	if _, err := redeem(t, svc, link, enrol.NodeInfo{Token: link.Token, Port: 2053, BasePath: "/", Scheme: "http", APIToken: "tok"}); err != nil {
		t.Fatalf("Redeem: %v", err)
	}
	var node model.Node
	if err := database.GetDB().Where("name = ?", "de1").First(&node).Error; err != nil {
		t.Fatal(err)
	}
	if err := (&service.TunnelService{}).Delete(tunnelByName(t, "node-de1").Id); err == nil {
		t.Fatalf("a management tunnel was deleted from the tunnel list while its node still exists")
	}
	if err := (&service.NodeService{}).Delete(node.Id); err != nil {
		t.Fatalf("delete node: %v", err)
	}
	if err := svc.NodeDeleted(node.Id); err != nil {
		t.Fatalf("NodeDeleted: %v", err)
	}
	if got := tunnelByName(t, "node-de1"); got != nil {
		t.Fatalf("tunnel of the deleted node remains: %+v", got)
	}

	if err := svc.Delete(pending.Id); err != nil {
		t.Fatalf("Delete pending: %v", err)
	}
	if got := tunnelByName(t, "node-nl1"); got != nil {
		t.Fatalf("tunnel of the cancelled enrolment remains: %+v", got)
	}
	left, err := svc.List()
	if err != nil || len(left) != 0 {
		t.Fatalf("List after both deletions = %+v, %v; want none", left, err)
	}
}

func TestTwoEnrolmentsGetDifferentManagementPortsAndNamesMustBeFree(t *testing.T) {
	newDB(t)
	svc := &Service{}
	createEnrolment(t, svc, "de1", 8443)
	createEnrolment(t, svc, "nl1", 8444)
	if a, b := tunnelByName(t, "node-de1").ManagePort, tunnelByName(t, "node-nl1").ManagePort; a == b {
		t.Fatalf("both enrolments got management port %d", a)
	}
	if _, err := svc.Create(enrolmentSpec("de1", 8445)); err == nil {
		t.Fatalf("a second enrolment named de1 was accepted")
	}
	if got := tunnelByName(t, "node-de1"); got == nil || got.Port != 8443 {
		t.Fatalf("the refused duplicate disturbed the first tunnel: %+v", got)
	}
}

// fakeMainPanel answers an enrolment the way a main panel does and records what the node told it.
func fakeMainPanel(t *testing.T, refuse bool) (link string, told *enrol.NodeInfo) {
	t.Helper()
	portal, err := tunnel.NewPortal(tunnel.PortalSpec{
		Name: "de1", Address: "203.0.113.10", Port: 8443, Engine: model.TunnelEngineBackpack, Transport: "wssmux",
	})
	if err != nil {
		t.Fatal(err)
	}
	setup, _ := tunnel.SetupLink(portal)
	return fakeMainPanelGranting(t, refuse, enrol.Grant{DataTunnel: setup})
}

func fakeMainPanelGranting(t *testing.T, refuse bool, grant enrol.Grant) (link string, told *enrol.NodeInfo) {
	t.Helper()
	told = &enrol.NodeInfo{}
	var offer enrol.Offer
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		info, seal, err := enrol.OpenRequest(offer.PrivateKey, body)
		if err != nil || refuse {
			http.Error(w, "refused", http.StatusGone)
			return
		}
		*told = info
		reply, _ := seal(grant)
		_, _ = w.Write(reply)
	}))
	t.Cleanup(srv.Close)
	var err error
	offer, err = enrol.NewOffer(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	return offer.Link, told
}

func TestJoiningAMainPanelStoresItsTunnelAndATokenThePanelHolds(t *testing.T) {
	newDB(t)
	link, told := fakeMainPanel(t, false)
	if err := (&Service{}).Join(context.Background(), link); err != nil {
		t.Fatalf("Join: %v", err)
	}
	port, err := (&service.SettingService{}).GetPort()
	if err != nil {
		t.Fatal(err)
	}
	bridge := tunnelByName(t, "de1")
	if bridge == nil || bridge.Role != model.TunnelRoleBridge || bridge.Engine != model.TunnelEngineBackpack || bridge.Transport != "wssmux" || bridge.ManagePort != 0 {
		t.Fatalf("bridge = %+v, want an ordinary BackPack bridge on wssmux", bridge)
	}
	if told.Port != port || told.Scheme != "http" || told.BasePath == "" {
		t.Fatalf("the panel was told %+v, want this panel's port %d over http with its base path", told, port)
	}
	row, ok := (&panel.ApiTokenService{}).MatchToken(told.APIToken)
	if !ok || row.Scope != model.ApiScopeNodeAdmin {
		t.Fatalf("the token sent to the main panel matches %+v (ok=%v), want a node-admin token", row, ok)
	}

	// Joining another panel updates the same tunnel and leaves one token.
	link2, told2 := fakeMainPanel(t, false)
	if err := (&Service{}).Join(context.Background(), link2); err != nil {
		t.Fatalf("second Join: %v", err)
	}
	if _, ok := (&panel.ApiTokenService{}).MatchToken(told.APIToken); ok {
		t.Fatalf("the first main panel's token still works after joining another")
	}
	if _, ok := (&panel.ApiTokenService{}).MatchToken(told2.APIToken); !ok {
		t.Fatalf("the second main panel's token does not work")
	}
}

func TestARefusedJoinLeavesNoTokenAndNoTunnel(t *testing.T) {
	newDB(t)
	link, _ := fakeMainPanel(t, true)
	err := (&Service{}).Join(context.Background(), link)
	if err == nil || !strings.Contains(err.Error(), "refused") {
		t.Fatalf("Join = %v, want the panel's refusal", err)
	}
	if got := tunnelByName(t, "de1"); got != nil {
		t.Fatalf("a refused join stored a tunnel: %+v", got)
	}
	tokens, err := (&panel.ApiTokenService{}).List()
	if err != nil || len(tokens) != 0 {
		t.Fatalf("a refused join left %d API token(s), err %v", len(tokens), err)
	}
}

func TestARefusedJoinKeepsTheChannelToTheCurrentMainPanel(t *testing.T) {
	newDB(t)
	link, told := fakeMainPanel(t, false)
	if err := (&Service{}).Join(context.Background(), link); err != nil {
		t.Fatalf("Join: %v", err)
	}
	refusing, _ := fakeMainPanel(t, true)
	if err := (&Service{}).Join(context.Background(), refusing); err == nil {
		t.Fatalf("the refused join reported success")
	}
	// The same link again must fail for the panel's reason, not over a leftover of the first attempt.
	if err := (&Service{}).Join(context.Background(), refusing); err == nil || !strings.Contains(err.Error(), "refused") {
		t.Fatalf("repeating the refused join: err = %v, want the panel's refusal", err)
	}
	if _, ok := (&panel.ApiTokenService{}).MatchToken(told.APIToken); !ok {
		t.Fatalf("a refused join revoked the current main panel's token")
	}
	if got := tunnelByName(t, "de1"); got == nil {
		t.Fatalf("a refused join removed the current tunnel to the main panel")
	}
}

func TestAnEnrolmentOnABackpackTunnelForwardsToTheNodesPanelPort(t *testing.T) {
	newDB(t)
	svc := &Service{}
	parent := model.ServerTunnel{
		Name: "bp1", Role: model.TunnelRolePortal, Engine: model.TunnelEngineBackpack, Enable: true,
		Address: "203.0.113.10", Port: 8443, UUID: "8d6f3a0e-5c1b-4f0e-9a51-2f6d1c7b9e10", Transport: "tcpmux",
		RelayPort: 63500,
	}
	if err := database.GetDB().Create(&parent).Error; err != nil {
		t.Fatalf("create parent: %v", err)
	}
	created, err := svc.Create(Spec{Name: "de1", PanelURL: "http://203.0.113.10:2053/", TunnelId: parent.Id})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	link, _ := enrol.ParseLink(created.Link)

	grant, err := redeem(t, svc, link, enrol.NodeInfo{
		Token: link.Token, Port: 2053, BasePath: "/x/", Scheme: "https", APIToken: "node-token-1",
	})
	if err != nil {
		t.Fatalf("Redeem: %v", err)
	}
	if grant.DataTunnel != "" {
		t.Fatalf("granted %q, want no tunnel to import: the node already has the BackPack tunnel", grant.DataTunnel)
	}
	rider := tunnelByName(t, "node-de1")
	if rider == nil || rider.ParentId != parent.Id || rider.ManageTarget != 2053 || rider.ManagePort == 0 {
		t.Fatalf("channel after Redeem = %+v, want a rider on tunnel %d aimed at the node's panel port 2053", rider, parent.Id)
	}
	var node model.Node
	if err := database.GetDB().Where("name = ?", "de1").First(&node).Error; err != nil || node.Port != rider.ManagePort {
		t.Fatalf("node = %+v (%v), want it behind the rider's management port %d", node, err, rider.ManagePort)
	}
	if err := svc.NodeDeleted(node.Id); err != nil {
		t.Fatalf("NodeDeleted: %v", err)
	}
	if tunnelByName(t, "node-de1") != nil || tunnelByName(t, "bp1") == nil {
		t.Fatal("removing the node must close its channel and keep the tunnel it rode on")
	}
}

func TestANodeWithATunnelGetsAnOrdinaryTunnelThatItsManagementRidesOn(t *testing.T) {
	newDB(t)
	svc := &Service{}
	created, err := svc.Create(enrolmentSpec("de1", 8443))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	link, err := enrol.ParseLink(created.Link)
	if err != nil {
		t.Fatal(err)
	}
	data, channel := tunnelByName(t, "de1"), tunnelByName(t, "node-de1")
	if data == nil || data.ManagePort != 0 || data.ParentId != 0 || data.Port != 8443 {
		t.Fatalf("data tunnel after Create = %+v, want an ordinary entry tunnel on 8443", data)
	}
	if channel == nil || channel.ParentId != data.Id || channel.ManagePort == 0 || channel.Port != 8443 {
		t.Fatalf("management channel = %+v, want one riding on the data tunnel %d", channel, data.Id)
	}

	grant, err := redeem(t, svc, link, enrol.NodeInfo{
		Token: link.Token, Port: 2053, BasePath: "/x/", Scheme: "https", APIToken: "node-token-1",
	})
	if err != nil {
		t.Fatalf("Redeem: %v", err)
	}
	dataBridge, err := tunnel.ParseSetupLink(grant.DataTunnel)
	if err != nil || dataBridge.UUID != data.UUID || dataBridge.Name != "de1" {
		t.Fatalf("granted data bridge = %+v (%v), want the data tunnel's own client", dataBridge, err)
	}
	var node model.Node
	if err := database.GetDB().Where("name = ?", "de1").First(&node).Error; err != nil {
		t.Fatal(err)
	}
	if got := tunnelByName(t, "de1"); got.NodeId != node.Id {
		t.Fatalf("data tunnel NodeId = %d, want the node %d so deleting it reaches the node's end", got.NodeId, node.Id)
	}

	if err := svc.NodeDeleted(node.Id); err != nil {
		t.Fatalf("NodeDeleted: %v", err)
	}
	if tunnelByName(t, "de1") != nil || tunnelByName(t, "node-de1") != nil {
		t.Fatal("removing the node must remove its channel and its data tunnel")
	}
}

func TestCancellingAPendingEnrolmentWithATunnelRemovesBothTunnels(t *testing.T) {
	newDB(t)
	svc := &Service{}
	created, err := svc.Create(enrolmentSpec("de1", 8443))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.Delete(created.Id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if tunnelByName(t, "de1") != nil || tunnelByName(t, "node-de1") != nil {
		t.Fatal("a cancelled enrolment left a tunnel behind")
	}
}

func TestAnEnrolmentWithoutAnEngineMakesABackpackTunnelOnWssmux(t *testing.T) {
	newDB(t)
	if _, err := (&Service{}).Create(enrolmentSpec("de1", 8443)); err != nil {
		t.Fatalf("Create: %v", err)
	}
	data, rider := tunnelByName(t, "de1"), tunnelByName(t, "node-de1")
	if data == nil || data.Engine != model.TunnelEngineBackpack || data.Transport != "wssmux" || data.RelayPort == 0 {
		t.Fatalf("node tunnel = %+v, want BackPack on wssmux with a relay port", data)
	}
	if rider == nil || rider.ParentId != data.Id || rider.ManagePort == 0 || rider.Engine != model.TunnelEngineBackpack {
		t.Fatalf("channel = %+v, want a BackPack rider on tunnel %d with a management port", rider, data.Id)
	}
}

func TestAnEnrolmentOnRakhshMakesARakhshTunnel(t *testing.T) {
	newDB(t)
	spec := enrolmentSpec("de1", 8443)
	spec.Engine = model.TunnelEngineRakhsh
	if _, err := (&Service{}).Create(spec); err != nil {
		t.Fatalf("Create: %v", err)
	}
	data := tunnelByName(t, "de1")
	if data == nil || data.Engine != model.TunnelEngineRakhsh || data.RakhshSecret == "" {
		t.Fatalf("node tunnel = %+v, want a Rakhsh tunnel with its own key", data)
	}
	if rider := tunnelByName(t, "node-de1"); rider == nil || rider.ParentId != data.Id {
		t.Fatalf("channel = %+v, want a rider on the Rakhsh tunnel", rider)
	}
}

func TestAnEnrolmentRefusesAnUnknownEngineAndLeavesNothingBehind(t *testing.T) {
	newDB(t)
	spec := enrolmentSpec("de1", 8443)
	spec.Engine = "xray"
	_, err := (&Service{}).Create(spec)
	if err == nil || !strings.Contains(err.Error(), `tunnel engine "xray" is not supported`) {
		t.Fatalf("Create with the xray engine: err = %v, want the engine refused", err)
	}
	if tunnelByName(t, "de1") != nil || tunnelByName(t, "node-de1") != nil {
		t.Fatal("the refused enrolment left a tunnel behind")
	}
}

func TestANodeFlaggedToEnrolAgainKeepsItsRecordWhenItJoinsAgain(t *testing.T) {
	newDB(t)
	svc := &Service{}
	_, first := createEnrolment(t, svc, "de1", 8443)
	if _, err := redeem(t, svc, first, enrol.NodeInfo{Token: first.Token, Port: 2053, BasePath: "/", Scheme: "https", APIToken: "old-token"}); err != nil {
		t.Fatalf("first Redeem: %v", err)
	}
	var old model.Node
	if err := database.GetDB().Where("name = ?", "de1").First(&old).Error; err != nil {
		t.Fatal(err)
	}
	// What the Xray-removal migration leaves behind: no tunnels, and the enrolment flagged.
	if err := database.GetDB().Where("name IN ?", []string{"de1", "node-de1"}).Delete(&model.ServerTunnel{}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(&model.NodeEnrolment{}).Where("name = ?", "de1").Update("needs_reenrol", true).Error; err != nil {
		t.Fatal(err)
	}

	_, second := createEnrolment(t, svc, "de1", 8443)
	if _, err := redeem(t, svc, second, enrol.NodeInfo{Token: second.Token, Port: 2054, BasePath: "/", Scheme: "https", APIToken: "new-token"}); err != nil {
		t.Fatalf("second Redeem: %v", err)
	}

	if got := nodeCount(t); got != 1 {
		t.Fatalf("%d nodes after enrolling again, want the one record kept", got)
	}
	node, err := (&service.NodeService{}).GetById(old.Id)
	if err != nil {
		t.Fatalf("the old node record is gone: %v", err)
	}
	portal := tunnelByName(t, "node-de1")
	if node.Port != portal.ManagePort || node.ApiToken != "new-token" {
		t.Fatalf("node = port %d token %q, want the new management port %d and token", node.Port, node.ApiToken, portal.ManagePort)
	}
	var rows []model.NodeEnrolment
	if err := database.GetDB().Where("name = ?", "de1").Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].NodeId != old.Id || rows[0].NeedsReenrol {
		t.Fatalf("enrolments named de1 = %+v, want one tied to node %d and no longer flagged", rows, old.Id)
	}
}
