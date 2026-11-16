package service

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	managePortFirst = 61000
	managePortLast  = 61999
)

var errManagementTunnel = errors.New("this tunnel manages a node; delete the node or its enrolment instead")

// freeManagePort picks a loopback port no stored inbound or tunnel claims and nothing is listening on now.
func (s *TunnelService) freeManagePort() (int, error) {
	db := database.GetDB()
	taken := map[int]bool{}
	var ports []int
	if err := db.Model(&model.ServerTunnel{}).Where("manage_port > 0").Pluck("manage_port", &ports).Error; err != nil {
		return 0, err
	}
	for _, p := range ports {
		taken[p] = true
	}
	ports = nil
	if err := db.Model(&model.Inbound{}).Where("node_id IS NULL AND port BETWEEN ? AND ?", managePortFirst, managePortLast).
		Pluck("port", &ports).Error; err != nil {
		return 0, err
	}
	for _, p := range ports {
		taken[p] = true
	}
	for port := managePortFirst; port <= managePortLast; port++ {
		if taken[port] {
			continue
		}
		l, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			continue
		}
		_ = l.Close()
		return port, nil
	}
	return 0, fmt.Errorf("no free management port between %d and %d", managePortFirst, managePortLast)
}

// CreateManagementPortal makes the listening end of a node's management tunnel on a free loopback port.
func (s *TunnelService) CreateManagementPortal(spec tunnel.PortalSpec) (*model.ServerTunnel, error) {
	portal, err := tunnel.NewPortal(spec)
	if err != nil {
		return nil, err
	}
	if portal.Engine != model.TunnelEngineXray {
		return nil, errors.New("a management tunnel runs on the Xray engine")
	}
	if portal.ManagePort, err = s.freeManagePort(); err != nil {
		return nil, err
	}
	if err := s.create(&portal); err != nil {
		return nil, err
	}
	return &portal, nil
}

// CreateManagementRider opens a node's management channel on an existing tunnel: same port and
// identity, a client and a loopback port of its own, so no second address gets exposed.
func (s *TunnelService) CreateManagementRider(parentId int, name string) (*model.ServerTunnel, error) {
	parent, err := s.get(parentId)
	if err != nil {
		return nil, err
	}
	if parent.Role != model.TunnelRolePortal || parent.ManagePort > 0 || parent.ParentId > 0 {
		return nil, errors.New("a node's management can only ride on the entry end of an ordinary tunnel")
	}
	rider := model.ServerTunnel{
		Name: name, Role: model.TunnelRolePortal, Engine: parent.Engine, Enable: true,
		Address: parent.Address, Port: parent.Port, UUID: uuid.NewString(), Security: parent.Security,
		ServerName: parent.ServerName, RealityPublicKey: parent.RealityPublicKey, ShortId: parent.ShortId,
		ParentId: parent.Id,
	}
	if rider.ManagePort, err = s.freeManagePort(); err != nil {
		return nil, err
	}
	if err := s.create(&rider); err != nil {
		return nil, err
	}
	return &rider, nil
}

// ImportManagementBridge stores the dialling end that delivers to this panel's port, replacing any earlier one.
func (s *TunnelService) ImportManagementBridge(link string, panelPort int) (*model.ServerTunnel, error) {
	bridge, err := tunnel.ParseSetupLink(link)
	if err != nil {
		return nil, err
	}
	if bridge.Engine != model.TunnelEngineXray {
		return nil, errors.New("a management tunnel runs on the Xray engine")
	}
	bridge.ManagePort = panelPort
	// A node answers to one main panel, so the older management channel is replaced, or kept if this one fails.
	err = database.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role = ? AND manage_port > 0", model.TunnelRoleBridge).Delete(&model.ServerTunnel{}).Error; err != nil {
			return err
		}
		var sameName int64
		if err := tx.Model(&model.ServerTunnel{}).Where("name = ?", bridge.Name).Count(&sameName).Error; err != nil {
			return err
		}
		if sameName > 0 {
			return fmt.Errorf("a tunnel with the name %q already exists", bridge.Name)
		}
		return tx.Create(&bridge).Error
	})
	if err != nil {
		return nil, err
	}
	s.applyChange()
	return &bridge, nil
}

// DeleteManagement removes a management tunnel; a tunnel already gone is not an error.
func (s *TunnelService) DeleteManagement(id int) error {
	err := database.GetDB().Where("id = ? AND manage_port > 0", id).Delete(&model.ServerTunnel{}).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	s.applyChange()
	return nil
}
