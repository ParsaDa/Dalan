package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel"
	"github.com/mhsanaei/3x-ui/v3/internal/web/runtime"
)

func TestARiderSharesItsTunnelsPortAndIdentityWithItsOwnClient(t *testing.T) {
	svc := newTunnelDB(t)
	parent, err := svc.CreatePortal(tunnel.PortalSpec{
		Name: "tr1", Address: "203.0.113.10", Port: 8443,
		Security: model.TunnelSecurityReality, ServerName: "www.example.com",
	}, nil)
	if err != nil {
		t.Fatalf("CreatePortal: %v", err)
	}

	rider, err := svc.CreateManagementRider(parent.Id, "node-de1")
	if err != nil {
		t.Fatalf("CreateManagementRider: %v", err)
	}
	if rider.ParentId != parent.Id || rider.Port != parent.Port || rider.Address != parent.Address ||
		rider.ShortId != parent.ShortId || rider.RealityPublicKey != parent.RealityPublicKey ||
		rider.ServerName != parent.ServerName {
		t.Fatalf("rider = %+v, want the port and REALITY identity of %+v", rider, parent)
	}
	if rider.UUID == "" || rider.UUID == parent.UUID || rider.ManagePort == 0 {
		t.Fatalf("rider uuid %q / manage port %d: want its own client and a loopback port", rider.UUID, rider.ManagePort)
	}
	if rider.RealityPrivateKey != "" {
		t.Fatal("the rider holds a copy of the tunnel's REALITY private key")
	}
	link, err := tunnel.SetupLink(*rider)
	if err != nil {
		t.Fatalf("SetupLink: %v", err)
	}
	bridge, err := tunnel.ParseSetupLink(link)
	if err != nil || bridge.UUID != rider.UUID || bridge.Port != parent.Port || bridge.Name != "node-de1" {
		t.Fatalf("the node's end = %+v (%v), want the rider's client on the tunnel's port", bridge, err)
	}

	tags := inboundTags(t)
	var links int
	for _, tag := range tags {
		if strings.HasSuffix(tag, "-link") {
			links++
		}
	}
	if links != 1 {
		t.Fatalf("inbounds %v: want one tunnel listener shared by both clients", tags)
	}
}

func TestATunnelCarryingANodeCannotBeDeletedAndAnotherPortalCannotCarryOne(t *testing.T) {
	svc := newTunnelDB(t)
	parent, err := svc.CreatePortal(tunnel.PortalSpec{
		Name: "tr1", Address: "203.0.113.10", Port: 8443, Security: model.TunnelSecurityNone,
	}, nil)
	if err != nil {
		t.Fatalf("CreatePortal: %v", err)
	}
	rider, err := svc.CreateManagementRider(parent.Id, "node-de1")
	if err != nil {
		t.Fatalf("CreateManagementRider: %v", err)
	}
	if err := svc.Delete(parent.Id); err == nil || !strings.Contains(err.Error(), "node-de1") {
		t.Fatalf("deleting a tunnel that carries a node: err = %v, want a refusal naming node-de1", err)
	}
	if _, err := svc.CreateManagementRider(rider.Id, "node-x"); err == nil {
		t.Fatal("a management channel was allowed to carry another channel")
	}
	if _, err := svc.CreateManagementRider(999, "node-y"); err == nil {
		t.Fatal("a management channel was allowed on a tunnel that does not exist")
	}
	if err := svc.DeleteManagement(rider.Id); err != nil {
		t.Fatalf("DeleteManagement: %v", err)
	}
	if err := svc.Delete(parent.Id); err != nil {
		t.Fatalf("once the node's channel is gone the tunnel can be deleted: %v", err)
	}
	var left int64
	database.GetDB().Model(&model.ServerTunnel{}).Count(&left)
	if left != 0 {
		t.Fatalf("%d tunnel rows left", left)
	}
}

// fakeTunnelNode answers the three tunnel calls of a node's panel and remembers its stored ends.
type fakeTunnelNode struct {
	srv   *httptest.Server
	mu    sync.Mutex
	ends  map[int]string
	next  int
	fail  bool
	calls []string
}

func newFakeTunnelNode(t *testing.T) (*fakeTunnelNode, *model.Node) {
	t.Helper()
	f := &fakeTunnelNode{ends: map[int]string{}, next: 40}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.calls = append(f.calls, r.Method+" "+r.URL.Path)
		reply := func(obj any) {
			raw, _ := json.Marshal(obj)
			_, _ = fmt.Fprintf(w, `{"success":true,"msg":"","obj":%s}`, raw)
		}
		if f.fail {
			http.Error(w, "down", http.StatusBadGateway)
			return
		}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/panel/api/tunnels/import":
			var body struct {
				Link         string `json:"link"`
				ExitOutbound string `json:"exitOutbound"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			bridge, err := tunnel.ParseSetupLink(body.Link)
			if err != nil {
				_, _ = fmt.Fprintf(w, `{"success":false,"msg":%q}`, err.Error())
				return
			}
			f.next++
			bridge.Id = f.next
			f.ends[bridge.Id] = bridge.Name
			reply(bridge)
		case r.Method == http.MethodGet && r.URL.Path == "/panel/api/tunnels/list":
			list := []model.ServerTunnel{}
			for id, name := range f.ends {
				list = append(list, model.ServerTunnel{Id: id, Name: name, Role: model.TunnelRoleBridge})
			}
			reply(list)
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/panel/api/tunnels/del/"):
			id, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/panel/api/tunnels/del/"))
			delete(f.ends, id)
			reply(nil)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(f.srv.Close)
	u, _ := url.Parse(f.srv.URL)
	port, _ := strconv.Atoi(u.Port())
	node := &model.Node{
		Name: "de1", Scheme: "http", Address: u.Hostname(), Port: port, BasePath: "/", ApiToken: "t",
		Enable: true, AllowPrivateAddress: true,
	}
	if err := database.GetDB().Create(node).Error; err != nil {
		t.Fatalf("create node: %v", err)
	}
	prev := runtime.GetManager()
	runtime.SetManager(runtime.NewManager(runtime.LocalDeps{APIPort: func() int { return 0 }, SetNeedRestart: func() {}}))
	t.Cleanup(func() { runtime.SetManager(prev) })
	return f, node
}

func TestATunnelMadeForANodeIsStoredOnBothEndsAndDeletedFromBoth(t *testing.T) {
	svc := newTunnelDB(t)
	fake, node := newFakeTunnelNode(t)

	portal, err := svc.CreateOnNode(node.Id, tunnel.PortalSpec{
		Name: "tr1", Address: "203.0.113.10", Port: 8443, Security: model.TunnelSecurityNone,
	}, []string{"in-1"}, "")
	if err != nil {
		t.Fatalf("CreateOnNode: %v", err)
	}
	if portal.NodeId != node.Id || portal.Role != model.TunnelRolePortal {
		t.Fatalf("local end = %+v, want a portal tied to node %d", portal, node.Id)
	}
	if len(fake.ends) != 1 {
		t.Fatalf("node holds %d tunnel ends, want the bridge of tr1", len(fake.ends))
	}

	if err := svc.Delete(portal.Id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	var left int64
	database.GetDB().Model(&model.ServerTunnel{}).Count(&left)
	if left != 0 || len(fake.ends) != 0 {
		t.Fatalf("after Delete %d local rows and %d node ends remain, want none", left, len(fake.ends))
	}
}

func TestANodeThatCannotTakeItsEndLeavesNoHalfTunnelAndKeepsTheOneItHasUntilForced(t *testing.T) {
	svc := newTunnelDB(t)
	fake, node := newFakeTunnelNode(t)
	fake.fail = true
	if _, err := svc.CreateOnNode(node.Id, tunnel.PortalSpec{
		Name: "tr1", Address: "203.0.113.10", Port: 8443, Security: model.TunnelSecurityNone,
	}, nil, ""); err == nil {
		t.Fatal("CreateOnNode succeeded although the node refused its end")
	}
	var left int64
	database.GetDB().Model(&model.ServerTunnel{}).Count(&left)
	if left != 0 {
		t.Fatalf("%d local rows left after the node refused: a half tunnel", left)
	}

	fake.fail = false
	portal, err := svc.CreateOnNode(node.Id, tunnel.PortalSpec{
		Name: "tr1", Address: "203.0.113.10", Port: 8443, Security: model.TunnelSecurityNone,
	}, nil, "")
	if err != nil {
		t.Fatalf("CreateOnNode: %v", err)
	}
	fake.fail = true
	if err := svc.Delete(portal.Id); err == nil {
		t.Fatal("Delete removed the local end although the node could not drop its own")
	}
	if err := svc.DeleteHere(portal.Id); err != nil {
		t.Fatalf("DeleteHere: %v", err)
	}
	database.GetDB().Model(&model.ServerTunnel{}).Count(&left)
	if left != 0 {
		t.Fatalf("%d local rows left after DeleteHere", left)
	}
}

func TestADeleteRefusedForCarryingANodeLeavesTheNodesEndInPlace(t *testing.T) {
	svc := newTunnelDB(t)
	fake, node := newFakeTunnelNode(t)
	portal, err := svc.CreateOnNode(node.Id, tunnel.PortalSpec{
		Name: "tr1", Address: "203.0.113.10", Port: 8443, Security: model.TunnelSecurityNone,
	}, nil, "")
	if err != nil {
		t.Fatalf("CreateOnNode: %v", err)
	}
	if _, err := svc.CreateManagementRider(portal.Id, "node-de1"); err != nil {
		t.Fatalf("CreateManagementRider: %v", err)
	}

	if err := svc.Delete(portal.Id); err == nil {
		t.Fatal("a tunnel carrying a node's management was deleted")
	}
	if len(fake.ends) != 1 {
		t.Fatalf("the refused delete removed the node's end: %d ends left, want 1", len(fake.ends))
	}
}
