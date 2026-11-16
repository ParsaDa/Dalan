package service

import (
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel"
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
