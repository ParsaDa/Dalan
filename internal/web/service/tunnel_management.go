package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/logger"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel"
	"github.com/mhsanaei/3x-ui/v3/internal/web/runtime"
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

const nodeCallTimeout = 20 * time.Second

func (s *TunnelService) remoteFor(nodeID int) (*runtime.Remote, error) {
	node, err := (&NodeService{}).GetById(nodeID)
	if err != nil || node == nil {
		return nil, fmt.Errorf("node %d does not exist", nodeID)
	}
	if !node.Enable {
		return nil, fmt.Errorf("node %q is disabled", node.Name)
	}
	mgr := runtime.GetManager()
	if mgr == nil {
		return nil, errors.New("runtime manager unavailable")
	}
	return mgr.RemoteFor(node)
}

// CreateOnNode makes both ends of a tunnel from this panel: the listening end here and the dialling end on a node.
func (s *TunnelService) CreateOnNode(nodeID int, spec tunnel.PortalSpec, routeInbounds []string, exitOutbound string) (*model.ServerTunnel, error) {
	remote, err := s.remoteFor(nodeID)
	if err != nil {
		return nil, err
	}
	portal, err := s.CreatePortal(spec, routeInbounds)
	if err != nil {
		return nil, err
	}
	link, err := tunnel.SetupLink(*portal)
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), nodeCallTimeout)
		defer cancel()
		_, err = remote.ImportTunnel(ctx, link, exitOutbound)
	}
	if err == nil {
		err = database.GetDB().Model(&model.ServerTunnel{}).Where("id = ?", portal.Id).Update("node_id", nodeID).Error
		portal.NodeId = nodeID
	}
	if err != nil {
		if delErr := s.deleteLocal(portal); delErr != nil {
			logger.Warning("tunnels: removing the entry end of a tunnel the node did not take:", delErr)
		}
		return nil, fmt.Errorf("the node did not take its end of tunnel %q: %w", spec.Name, err)
	}
	return portal, nil
}

// deleteNodeEnd removes the dialling end a node holds for this tunnel; an end already gone is not an error.
func (s *TunnelService) deleteNodeEnd(t *model.ServerTunnel) error {
	if node, err := (&NodeService{}).GetById(t.NodeId); err != nil || node == nil {
		return nil
	}
	remote, err := s.remoteFor(t.NodeId)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), nodeCallTimeout)
	defer cancel()
	ends, err := remote.ListTunnels(ctx)
	if err != nil {
		return err
	}
	for _, end := range ends {
		if end.Name == t.Name && end.Role == model.TunnelRoleBridge {
			return remote.DeleteTunnel(ctx, end.Id)
		}
	}
	return nil
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

// DeleteNodeTunnel removes the entry tunnel made for a node's traffic; one already gone is not an error.
func (s *TunnelService) DeleteNodeTunnel(id int) error {
	err := database.GetDB().Where("id = ? AND manage_port = 0 AND role = ?", id, model.TunnelRolePortal).
		Delete(&model.ServerTunnel{}).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	s.applyChange()
	return nil
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

// routeTestHitWait is how long the listeners get to report a connection after the node says it connected.
const routeTestHitWait = 500 * time.Millisecond

// RouteTest finds out which ports a node abroad can reach on this server: it listens on each port,
// has the node dial them, and counts a port only when the connection arrived here.
func (s *TunnelService) RouteTest(nodeID int, address string, ports []int) ([]tunnel.PortResult, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return nil, errors.New("a route test needs this server's address as the node sees it")
	}
	if err := tunnel.ValidatePorts(ports); err != nil {
		return nil, err
	}
	remote, err := s.remoteFor(nodeID)
	if err != nil {
		return nil, err
	}
	results := make([]tunnel.PortResult, len(ports))
	hits := make(map[int]*atomic.Int32, len(ports))
	var listening []int
	for i, port := range ports {
		results[i] = tunnel.PortResult{Port: port}
		l, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			results[i].Error = "this server cannot listen on the port: " + err.Error()
			continue
		}
		defer l.Close()
		hits[port] = new(atomic.Int32)
		listening = append(listening, port)
		go func() {
			for {
				conn, err := l.Accept()
				if err != nil {
					return
				}
				hits[port].Add(1)
				_ = conn.Close()
			}
		}()
	}
	if len(listening) == 0 {
		return results, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), nodeCallTimeout)
	defer cancel()
	dialed, err := remote.DialPorts(ctx, address, listening)
	if err != nil {
		return nil, err
	}
	byPort := map[int]tunnel.PortResult{}
	for _, d := range dialed {
		byPort[d.Port] = d
	}
	deadline := time.Now().Add(routeTestHitWait)
	for i := range results {
		counter, ok := hits[results[i].Port]
		if !ok {
			continue
		}
		d := byPort[results[i].Port]
		for d.Reachable && counter.Load() == 0 && time.Now().Before(deadline) {
			time.Sleep(10 * time.Millisecond)
		}
		results[i].LatencyMs, results[i].Error = d.LatencyMs, d.Error
		results[i].Reachable = d.Reachable && counter.Load() > 0
		if d.Reachable && !results[i].Reachable {
			results[i].Error = "something other than this panel answered on that address and port"
		}
	}
	return results, nil
}
