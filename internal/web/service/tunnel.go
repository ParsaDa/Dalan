package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/config"
	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/logger"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/backpack"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/xrayreverse"

	"gorm.io/gorm"
)

// tunnelEngines runs the BackPack processes of this panel; tests replace it with one on a fake engine.
var tunnelEngines = &backpack.Supervisor{
	Binary: filepath.Join(config.GetBinFolderPath(), "backpack"),
	Dir:    filepath.Join(config.GetDBFolderPath(), "tunnels"),
}

const engineInstallTimeout = 5 * time.Minute

// EngineInfo says whether the BackPack engine is on this server, which release is there (empty when
// it was placed by hand) and which one Dalan installs by default.
type EngineInfo struct {
	Installed bool   `json:"installed" example:"true"`
	Version   string `json:"version" example:"v1.9.0"`
	Pinned    string `json:"pinned" example:"v1.9.0"`
}

// TunnelService stores the tunnel ends this panel owns and schedules Xray to pick up changes.
type TunnelService struct {
	xrayService XrayService
}

// TunnelOutbound is an entry tunnel as Xray routing sees it: an outbound that rules and balancers can target.
type TunnelOutbound struct {
	Tag    string `json:"tag" example:"tunnel-de1"`
	Name   string `json:"name" example:"de1"`
	Engine string `json:"engine" example:"xray"`
}

// RoutableOutbounds lists the enabled entry tunnels that exist as outbounds in the generated Xray config.
func (s *TunnelService) RoutableOutbounds() ([]TunnelOutbound, error) {
	tunnels, err := s.GetAll()
	if err != nil {
		return nil, err
	}
	out := make([]TunnelOutbound, 0, len(tunnels))
	for _, t := range tunnels {
		if !xrayreverse.Routable(t) {
			continue
		}
		out = append(out, TunnelOutbound{Tag: xrayreverse.PortalOutboundTag(t.Name), Name: t.Name, Engine: t.Engine})
	}
	return out, nil
}

// ListWithTraffic returns every tunnel with the bytes its entry outbound has carried.
func (s *TunnelService) ListWithTraffic() ([]model.ServerTunnel, error) {
	tunnels, err := s.GetAll()
	if err != nil {
		return nil, err
	}
	var rows []model.OutboundTraffics
	if err := database.GetDB().Where("tag LIKE ?", xrayreverse.PortalOutboundTag("")+"%").Find(&rows).Error; err != nil {
		return nil, err
	}
	byTag := make(map[string]model.OutboundTraffics, len(rows))
	for _, r := range rows {
		byTag[r.Tag] = r
	}
	for i := range tunnels {
		if t := &tunnels[i]; xrayreverse.Routable(*t) {
			r := byTag[xrayreverse.PortalOutboundTag(t.Name)]
			t.Up, t.Down = r.Up, r.Down
		}
	}
	return tunnels, nil
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
	if listensOnPort(*t) {
		if err := s.checkPortFree(t.Port, 0); err != nil {
			return err
		}
	}
	if t.RelayPort > 0 {
		if err := s.checkRelayPortFree(t.RelayPort); err != nil {
			return err
		}
	}
	if t.Mode == model.TunnelModeDirect {
		if err := s.checkLinkAddressesFree(t, 0); err != nil {
			return err
		}
	}
	if err := db.Create(t).Error; err != nil {
		return err
	}
	s.applyChange()
	return nil
}

// listensOnPort tells whether this end opens its port here: a reverse portal does, and so does the exit end of
// a direct tunnel. A management rider shares its parent's port and opens none.
func listensOnPort(t model.ServerTunnel) bool {
	if t.Mode == model.TunnelModeDirect {
		return t.Role == model.TunnelRoleBridge
	}
	return t.Role == model.TunnelRolePortal && t.ParentId == 0
}

// A listening end cannot share its port with an inbound or another listening end on this panel; except is
// the tunnel being edited, which may keep its own port.
func (s *TunnelService) checkPortFree(port, except int) error {
	db := database.GetDB()
	var count int64
	if err := db.Model(&model.ServerTunnel{}).
		Where("id <> ? AND port = ? AND parent_id = 0 AND ((role = ? AND COALESCE(mode, '') <> ?) OR (role = ? AND mode = ?))",
			except, port, model.TunnelRolePortal, model.TunnelModeDirect, model.TunnelRoleBridge, model.TunnelModeDirect).
		Count(&count).Error; err != nil {
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

var errRelayPortTaken = errors.New("relay port is already used on this server")

// A relay port is a loopback listener on both ends, so it cannot share a port with an inbound or another relay.
func (s *TunnelService) checkRelayPortFree(port int) error {
	db := database.GetDB()
	var count int64
	if err := db.Model(&model.ServerTunnel{}).Where("relay_port = ?", port).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		if err := db.Model(&model.Inbound{}).Where("node_id IS NULL AND port = ?", port).Count(&count).Error; err != nil {
			return err
		}
	}
	if count > 0 {
		return fmt.Errorf("relay port %d: %w", port, errRelayPortTaken)
	}
	return nil
}

var errLinkAddressTaken = errors.New("tunnel link addresses are already used by another direct tunnel")

// Two direct tunnels on one server need different address pairs, or their interfaces would clash.
func (s *TunnelService) checkLinkAddressesFree(t *model.ServerTunnel, except int) error {
	var others []model.ServerTunnel
	if err := database.GetDB().Where("id <> ? AND mode = ?", except, model.TunnelModeDirect).Find(&others).Error; err != nil {
		return err
	}
	for _, o := range others {
		if o.Options["local_ip"] == t.Options["local_ip"] || o.Options["peer_ip"] == t.Options["peer_ip"] ||
			o.Options["local_ip"] == t.Options["peer_ip"] || o.Options["peer_ip"] == t.Options["local_ip"] {
			return fmt.Errorf("%w: %v and %v", errLinkAddressTaken, t.Options["local_ip"], t.Options["peer_ip"])
		}
	}
	return nil
}

// CreatePortal makes the listening end of a tunnel; routeInbounds are the local inbound tags that leave through it.
func (s *TunnelService) CreatePortal(spec tunnel.PortalSpec, routeInbounds []string) (*model.ServerTunnel, error) {
	for attempt := 0; ; attempt++ {
		portal, err := tunnel.NewPortal(spec)
		if err != nil {
			return nil, err
		}
		portal.RouteInbounds = routeInbounds
		err = s.create(&portal)
		if errors.Is(err, errRelayPortTaken) && attempt < 5 {
			continue
		}
		if err != nil {
			return nil, err
		}
		return &portal, nil
	}
}

// Import makes the dialing end of a tunnel from the other panel's setup link. A link for a name that is
// already a dialing end updates that end in place, which is how an edit on the entry end reaches this one.
func (s *TunnelService) Import(link, exitOutbound string) (*model.ServerTunnel, error) {
	bridge, err := tunnel.ParseSetupLink(link)
	if err != nil {
		return nil, err
	}
	bridge.ExitOutbound = exitOutbound
	var existing model.ServerTunnel
	switch err := database.GetDB().Where("name = ?", bridge.Name).First(&existing).Error; {
	case err == nil:
		return s.updateDialingEnd(&existing, bridge)
	case !errors.Is(err, gorm.ErrRecordNotFound):
		return nil, err
	}
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
	s.applyChange()
	return nil
}

// Delete removes a tunnel; when this panel made the other end on a node, that end is removed first.
func (s *TunnelService) Delete(id int) error {
	t, err := s.get(id)
	if err != nil {
		return err
	}
	if err := s.refuseWhileInUse(t); err != nil {
		return err
	}
	if t.NodeId > 0 && t.Role == model.TunnelRolePortal {
		if err := s.deleteNodeEnd(t); err != nil {
			return fmt.Errorf("%w; remove only this end if the node is gone for good", err)
		}
	}
	return s.deleteLocal(t)
}

// DeleteHere removes this end only, for a tunnel whose node cannot be asked any more.
func (s *TunnelService) DeleteHere(id int) error {
	t, err := s.get(id)
	if err != nil {
		return err
	}
	return s.deleteLocal(t)
}

// refuseWhileInUse keeps a tunnel that a pool or a node's management still uses, before any end is touched.
func (s *TunnelService) refuseWhileInUse(t *model.ServerTunnel) error {
	var pools []model.TunnelPool
	if err := database.GetDB().Find(&pools).Error; err != nil {
		return err
	}
	for _, p := range pools {
		for _, m := range p.Members {
			if m == t.Name {
				return fmt.Errorf("this tunnel is part of pool %s; delete the pool first", p.Name)
			}
		}
	}
	if t.ManagePort > 0 && t.Role == model.TunnelRolePortal {
		return errManagementTunnel
	}
	var carried []string
	if err := database.GetDB().Model(&model.ServerTunnel{}).Where("parent_id = ?", t.Id).Pluck("name", &carried).Error; err != nil {
		return err
	}
	if len(carried) > 0 {
		return fmt.Errorf("this tunnel carries the management of %s; remove that node first", strings.Join(carried, ", "))
	}
	return nil
}

func (s *TunnelService) deleteLocal(t *model.ServerTunnel) error {
	if err := s.refuseWhileInUse(t); err != nil {
		return err
	}
	if err := database.GetDB().Delete(&model.ServerTunnel{}, t.Id).Error; err != nil {
		return err
	}
	s.applyChange()
	return nil
}

const tunnelProbeTimeout = 8 * time.Second

// Status sends one request through an entry tunnel and reports whether it came back, how fast and from where.
func (s *TunnelService) Status(id int) (tunnel.Status, error) {
	t, err := s.get(id)
	if err != nil {
		return tunnel.Status{}, err
	}
	if t.Mode == model.TunnelModeDirect {
		return s.directStatus(t), nil
	}
	if t.Engine == model.TunnelEngineBackpack {
		return s.backpackStatus(t), nil
	}
	if t.ManagePort > 0 {
		return tunnel.Status{}, errors.New("a management tunnel is tested by its node's status, not by a request through it")
	}
	if t.Role != model.TunnelRolePortal {
		return tunnel.Status{}, errors.New("only the entry end of a tunnel can be tested from this server")
	}
	addr := fmt.Sprintf("127.0.0.1:%d", xrayreverse.ProbePort(*t))
	return tunnel.ProbeVia(addr, tunnel.DefaultProbeURL, tunnelProbeTimeout), nil
}

// applyChange lets both engines pick up a stored change; a missing BackPack engine is reported by Status.
func (s *TunnelService) applyChange() {
	s.xrayService.SetToNeedRestart()
	if err := s.ReconcileEngines(); err != nil {
		logger.Warning("BackPack tunnels:", err)
	}
}

// ReconcileEngines starts and stops BackPack processes to match the stored tunnels.
func (s *TunnelService) ReconcileEngines() error {
	tunnels, err := s.GetAll()
	if err != nil {
		return err
	}
	return tunnelEngines.Reconcile(backpack.WithManagement(tunnels))
}

// StopEngines stops every BackPack process this panel started.
func (s *TunnelService) StopEngines() { tunnelEngines.StopAll() }

func (s *TunnelService) EngineInfo() EngineInfo {
	_, err := os.Stat(tunnelEngines.Binary)
	return EngineInfo{Installed: err == nil, Version: backpack.InstalledVersion(tunnelEngines.Binary), Pinned: backpack.PinnedVersion}
}

// InstallEngine downloads the pinned BackPack release through the panel's own proxy settings and starts its tunnels.
func (s *TunnelService) InstallEngine() error {
	ctx, cancel := context.WithTimeout(context.Background(), engineInstallTimeout)
	defer cancel()
	return s.updateEngineFrom(ctx, directEngineSource, backpack.PinnedVersion)
}

// directInterfaceUp is replaced by tests; the engine creates the interface once the link is established.
var directInterfaceUp = func(name string) bool {
	iface, err := net.InterfaceByName(name)
	return err == nil && iface.Flags&net.FlagUp != 0
}

// directStatus judges a direct tunnel by its link interface, since it has no forwarded Xray traffic to probe.
func (s *TunnelService) directStatus(t *model.ServerTunnel) tunnel.Status {
	if !s.EngineInfo().Installed {
		return tunnel.Status{Error: backpack.ErrNotInstalled.Error()}
	}
	name := backpack.DirectInterface(t.Id)
	if !directInterfaceUp(name) {
		return tunnel.Status{Error: fmt.Sprintf("the link interface %s is not up", name)}
	}
	status := tunnel.Status{Connected: true}
	if t.Role == model.TunnelRolePortal {
		status.ExitIP = t.Address
	}
	return status
}

func (s *TunnelService) backpackStatus(t *model.ServerTunnel) tunnel.Status {
	if !s.EngineInfo().Installed {
		return tunnel.Status{Error: backpack.ErrNotInstalled.Error()}
	}
	snap, err := backpack.ReadSnapshot(tunnelEngines.Dir, t.Name)
	if err != nil {
		return tunnel.Status{Error: err.Error()}
	}
	status := tunnel.Status{Connected: snap.Connected}
	// On the entry end the connected peer is the exit server, which is where traffic leaves.
	if t.Role == model.TunnelRolePortal && snap.Connected {
		if host, _, err := net.SplitHostPort(snap.Peer); err == nil {
			status.ExitIP = host
		}
	}
	return status
}
