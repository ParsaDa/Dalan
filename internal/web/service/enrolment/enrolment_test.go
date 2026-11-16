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
		Security: model.TunnelSecurityNone,
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
	if !strings.Contains(created.Command, created.Link) {
		t.Fatalf("command %q does not carry the link", created.Command)
	}

	grant, err := redeem(t, svc, link, enrol.NodeInfo{
		Token: link.Token, Port: 2053, BasePath: "/x/", Scheme: "https", APIToken: "node-token-1",
	})
	if err != nil {
		t.Fatalf("Redeem: %v", err)
	}
	bridge, err := tunnel.ParseSetupLink(grant.Tunnel)
	if err != nil {
		t.Fatalf("granted tunnel link: %v", err)
	}
	if bridge.UUID != portal.UUID || bridge.Port != 8443 || bridge.Address != "203.0.113.10" {
		t.Fatalf("granted bridge %+v does not dial the portal %+v", bridge, portal)
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
	if got := tunnelByName(t, "node-de1"); got != nil {
		t.Fatalf("the expired enrolment's tunnel still listens: %+v", got)
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
	told = &enrol.NodeInfo{}
	var offer enrol.Offer
	portal, err := tunnel.NewPortal(tunnel.PortalSpec{Name: "node-de1", Address: "203.0.113.10", Port: 8443, Security: model.TunnelSecurityNone})
	if err != nil {
		t.Fatal(err)
	}
	setup, _ := tunnel.SetupLink(portal)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		info, seal, err := enrol.OpenRequest(offer.PrivateKey, body)
		if err != nil || refuse {
			http.Error(w, "refused", http.StatusGone)
			return
		}
		*told = info
		reply, _ := seal(enrol.Grant{Tunnel: setup})
		_, _ = w.Write(reply)
	}))
	t.Cleanup(srv.Close)
	offer, err = enrol.NewOffer(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	return offer.Link, told
}

func TestJoiningAMainPanelStoresTheManagementBridgeAndATokenThePanelHolds(t *testing.T) {
	newDB(t)
	link, told := fakeMainPanel(t, false)
	if err := (&Service{}).Join(context.Background(), link); err != nil {
		t.Fatalf("Join: %v", err)
	}
	port, err := (&service.SettingService{}).GetPort()
	if err != nil {
		t.Fatal(err)
	}
	bridge := tunnelByName(t, "node-de1")
	if bridge == nil || bridge.Role != model.TunnelRoleBridge || bridge.ManagePort != port {
		t.Fatalf("bridge = %+v, want a bridge delivering to this panel's port %d", bridge, port)
	}
	if told.Port != port || told.Scheme != "http" || told.BasePath == "" {
		t.Fatalf("the panel was told %+v, want this panel's port %d over http with its base path", told, port)
	}
	row, ok := (&panel.ApiTokenService{}).MatchToken(told.APIToken)
	if !ok || row.Scope != model.ApiScopeNodeAdmin {
		t.Fatalf("the token sent to the main panel matches %+v (ok=%v), want a node-admin token", row, ok)
	}

	// Joining again replaces the old channel: one management bridge, one token.
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
	if got := tunnelByName(t, "node-de1"); got != nil {
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
	if got := tunnelByName(t, "node-de1"); got == nil {
		t.Fatalf("a refused join removed the current management tunnel")
	}
}
