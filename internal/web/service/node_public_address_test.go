package service

import (
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

func TestNodePublicAddressSurvivesCreateAndUpdate(t *testing.T) {
	setupConflictDB(t)

	svc := &NodeService{}
	token := "tok"
	view, err := svc.CreateFromRequest(&NodeMutationRequest{
		Name: "dialled-in", Scheme: "http", Address: "127.0.0.1", Port: 61000,
		ApiToken: &token, Enable: true, AllowPrivateAddress: true,
		PublicAddress: " 198.51.100.9 ",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if view.PublicAddress != "198.51.100.9" {
		t.Fatalf("created PublicAddress = %q, want 198.51.100.9", view.PublicAddress)
	}

	err = svc.UpdateFromRequest(view.Id, &NodeMutationRequest{
		Name: "dialled-in", Scheme: "http", Address: "127.0.0.1", Port: 61000,
		Enable: true, AllowPrivateAddress: true, PublicAddress: "node.example.com",
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	got, err := svc.GetViewById(view.Id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.PublicAddress != "node.example.com" {
		t.Fatalf("updated PublicAddress = %q, want node.example.com", got.PublicAddress)
	}
}

func TestNodePublicAddressMustBeAHost(t *testing.T) {
	n := &model.Node{Name: "n", Address: "example.com", Port: 2053, PublicAddress: "not a host!"}
	err := (&NodeService{}).normalize(n)
	if err == nil {
		t.Fatal("a public address that is not a host was accepted")
	}
}

func TestInboundOptionsGiveClientsTheNodePublicAddress(t *testing.T) {
	setupConflictDB(t)

	node := &model.Node{Name: "dialled-in", Address: "127.0.0.1", PublicAddress: "198.51.100.9", Port: 61000, Enable: true}
	if err := database.GetDB().Create(node).Error; err != nil {
		t.Fatalf("create node: %v", err)
	}
	in := &model.Inbound{
		UserId: 1, Tag: "in-443-tcp", Enable: true, Listen: "0.0.0.0", Port: 443, Protocol: model.VLESS,
		StreamSettings: `{"network":"tcp"}`, Settings: `{"clients":[]}`, NodeID: &node.Id,
	}
	if err := database.GetDB().Create(in).Error; err != nil {
		t.Fatalf("create inbound: %v", err)
	}

	options, err := (&InboundService{}).GetInboundOptions(1)
	if err != nil {
		t.Fatalf("GetInboundOptions: %v", err)
	}
	for _, o := range options {
		if o.Id == in.Id {
			if o.NodeAddress != "198.51.100.9" {
				t.Fatalf("NodeAddress = %q, want the public address 198.51.100.9", o.NodeAddress)
			}
			return
		}
	}
	t.Fatal("inbound missing from options")
}
