package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/xrayreverse"

	"gorm.io/gorm"
)

// TunnelService stores the tunnel ends this panel owns and schedules Xray to pick up changes.
type TunnelService struct {
	xrayService XrayService
}

func (s *TunnelService) GetAll() ([]model.ServerTunnel, error) {
	var tunnels []model.ServerTunnel
	err := database.GetDB().Order("id asc").Find(&tunnels).Error
	return tunnels, err
}

func (s *TunnelService) get(id int) (*model.ServerTunnel, error) {
	var t model.ServerTunnel
	if err := database.GetDB().First(&t, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("tunnel %d does not exist", id)
		}
		return nil, err
	}
	return &t, nil
}

func (s *TunnelService) create(t *model.ServerTunnel) error {
	db := database.GetDB()
	var count int64
	if err := db.Model(&model.ServerTunnel{}).Where("name = ?", t.Name).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("a tunnel with the name %q already exists", t.Name)
	}
	if t.Role == model.TunnelRolePortal {
		if err := s.checkPortFree(t.Port); err != nil {
			return err
		}
	}
	if err := db.Create(t).Error; err != nil {
		return err
	}
	s.xrayService.SetToNeedRestart()
	return nil
}

// A portal listens on its port, so it cannot share it with an inbound or another portal on this panel.
func (s *TunnelService) checkPortFree(port int) error {
	db := database.GetDB()
	var count int64
	if err := db.Model(&model.ServerTunnel{}).
		Where("role = ? AND port = ?", model.TunnelRolePortal, port).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		if err := db.Model(&model.Inbound{}).Where("node_id IS NULL AND port = ?", port).Count(&count).Error; err != nil {
			return err
		}
	}
	if count > 0 {
		return fmt.Errorf("port %d is already used on this server", port)
	}
	return nil
}

// CreatePortal makes the listening end of a tunnel; routeInbounds are the local inbound tags that leave through it.
func (s *TunnelService) CreatePortal(spec tunnel.PortalSpec, routeInbounds []string) (*model.ServerTunnel, error) {
	portal, err := tunnel.NewPortal(spec)
	if err != nil {
		return nil, err
	}
	portal.RouteInbounds = routeInbounds
	if err := s.create(&portal); err != nil {
		return nil, err
	}
	return &portal, nil
}

// Import makes the dialing end of a tunnel from the other panel's setup link.
func (s *TunnelService) Import(link, exitOutbound string) (*model.ServerTunnel, error) {
	bridge, err := tunnel.ParseSetupLink(link)
	if err != nil {
		return nil, err
	}
	bridge.ExitOutbound = exitOutbound
	if err := s.create(&bridge); err != nil {
		return nil, err
	}
	return &bridge, nil
}

func (s *TunnelService) SetupLink(id int) (string, error) {
	t, err := s.get(id)
	if err != nil {
		return "", err
	}
	return tunnel.SetupLink(*t)
}

func (s *TunnelService) SetEnable(id int, enable bool) error {
	if _, err := s.get(id); err != nil {
		return err
	}
	if err := database.GetDB().Model(&model.ServerTunnel{}).Where("id = ?", id).Update("enable", enable).Error; err != nil {
		return err
	}
	s.xrayService.SetToNeedRestart()
	return nil
}

func (s *TunnelService) Delete(id int) error {
	if _, err := s.get(id); err != nil {
		return err
	}
	if err := database.GetDB().Delete(&model.ServerTunnel{}, id).Error; err != nil {
		return err
	}
	s.xrayService.SetToNeedRestart()
	return nil
}

const tunnelProbeTimeout = 8 * time.Second

// Status sends one request through an entry tunnel and reports whether it came back, how fast and from where.
func (s *TunnelService) Status(id int) (tunnel.Status, error) {
	t, err := s.get(id)
	if err != nil {
		return tunnel.Status{}, err
	}
	if t.Role != model.TunnelRolePortal {
		return tunnel.Status{}, errors.New("only the entry end of a tunnel can be tested from this server")
	}
	addr := fmt.Sprintf("127.0.0.1:%d", xrayreverse.ProbePort(*t))
	return tunnel.ProbeVia(addr, tunnel.DefaultProbeURL, tunnelProbeTimeout), nil
}
