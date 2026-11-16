package service

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/logger"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/pool"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/xrayreverse"
)

var poolNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,25}$`)

// TunnelPoolService stores tunnel pools and moves their traffic between members.
type TunnelPoolService struct {
	tunnels TunnelService
}

// List returns the pools with Active set to the member that carries traffic now.
func (s *TunnelPoolService) List() ([]model.TunnelPool, error) {
	var pools []model.TunnelPool
	if err := database.GetDB().Order("id asc").Find(&pools).Error; err != nil {
		return nil, err
	}
	tunnels, err := s.tunnels.GetAll()
	if err != nil {
		return nil, err
	}
	for i := range pools {
		pools[i].Active = xrayreverse.ActiveMember(pools[i], tunnels)
	}
	return pools, nil
}

// Create stores a pool of two or more ordinary entry tunnels; the first one carries traffic to begin with.
func (s *TunnelPoolService) Create(name string, members, routeInbounds []string) (*model.TunnelPool, error) {
	name = strings.TrimSpace(name)
	if !poolNamePattern.MatchString(name) {
		return nil, fmt.Errorf("pool name %q must be lower-case letters, digits and dashes, at most 26 characters", name)
	}
	if len(members) < 2 {
		return nil, errors.New("a pool needs at least two tunnels")
	}
	tunnels, err := s.tunnels.GetAll()
	if err != nil {
		return nil, err
	}
	byName := map[string]model.ServerTunnel{}
	for _, t := range tunnels {
		byName[t.Name] = t
	}
	seen := map[string]bool{}
	for _, m := range members {
		if seen[m] {
			return nil, fmt.Errorf("tunnel %q is listed twice", m)
		}
		seen[m] = true
		t, ok := byName[m]
		switch {
		case !ok:
			return nil, fmt.Errorf("tunnel %q does not exist", m)
		case t.ManagePort > 0:
			return nil, fmt.Errorf("tunnel %q is a management tunnel; a pool takes ordinary entry tunnels", m)
		case t.Role != model.TunnelRolePortal:
			return nil, fmt.Errorf("tunnel %q is not an entry tunnel", m)
		}
	}
	row := model.TunnelPool{Name: name, Members: members, RouteInbounds: routeInbounds, Active: members[0]}
	var taken int64
	if err := database.GetDB().Model(&model.TunnelPool{}).Where("name = ?", name).Count(&taken).Error; err != nil {
		return nil, err
	}
	if taken > 0 {
		return nil, fmt.Errorf("a pool named %q already exists", name)
	}
	if err := database.GetDB().Create(&row).Error; err != nil {
		return nil, err
	}
	s.tunnels.applyChange()
	return &row, nil
}

func (s *TunnelPoolService) Delete(id int) error {
	if err := database.GetDB().Delete(&model.TunnelPool{}, id).Error; err != nil {
		return err
	}
	s.tunnels.applyChange()
	return nil
}

// Probe measures one pool member the way the Tunnels page does.
func (s *TunnelPoolService) Probe(t model.ServerTunnel) pool.Reading {
	status, err := s.tunnels.Status(t.Id)
	if err != nil || !t.Enable {
		return pool.Reading{Name: t.Name}
	}
	return pool.Reading{Name: t.Name, Up: status.Connected, LatencyMs: int(status.LatencyMs)}
}

// Observe runs one round for every pool: it measures the members and moves the traffic when a
// monitor says so. The monitors belong to the caller and carry the streaks between rounds.
func (s *TunnelPoolService) Observe(monitors map[int]*pool.Monitor, probe func(model.ServerTunnel) pool.Reading) error {
	pools, err := s.List()
	if err != nil {
		return err
	}
	tunnels, err := s.tunnels.GetAll()
	if err != nil {
		return err
	}
	byName := map[string]model.ServerTunnel{}
	for _, t := range tunnels {
		byName[t.Name] = t
	}
	live := map[int]bool{}
	for _, p := range pools {
		live[p.Id] = true
		m := monitors[p.Id]
		if m == nil || m.Active() != p.Active {
			m = pool.New(p.Active)
			monitors[p.Id] = m
		}
		readings := readMembers(p.Members, byName, probe)
		if next := m.Observe(readings); next != "" {
			s.moveTo(p, next, byName)
		}
	}
	for id := range monitors {
		if !live[id] {
			delete(monitors, id)
		}
	}
	return nil
}

func readMembers(members []string, byName map[string]model.ServerTunnel, probe func(model.ServerTunnel) pool.Reading) []pool.Reading {
	readings := make([]pool.Reading, len(members))
	var wg sync.WaitGroup
	for i, name := range members {
		t, ok := byName[name]
		if !ok {
			readings[i] = pool.Reading{Name: name}
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			readings[i] = probe(t)
		}()
	}
	wg.Wait()
	return readings
}

// moveTo stores the new active member and, when Xray is running, points the live balancer at it.
func (s *TunnelPoolService) moveTo(p model.TunnelPool, member string, byName map[string]model.ServerTunnel) {
	if err := database.GetDB().Model(&model.TunnelPool{}).Where("id = ?", p.Id).Update("active", member).Error; err != nil {
		logger.Warning("tunnel pool", p.Name, "could not store its new active tunnel:", err)
		return
	}
	logger.Infof("tunnel pool %s: traffic moves from %s to %s", p.Name, p.Active, member)
	err := s.tunnels.xrayService.OverrideBalancer(xrayreverse.PoolBalancerTag(p.Name), xrayreverse.PortalOutboundTag(member))
	if err != nil && !errors.Is(err, ErrXrayNotRunning) {
		logger.Warning("tunnel pool", p.Name, "could not move the live balancer, it follows at the next Xray start:", err)
	}
}
