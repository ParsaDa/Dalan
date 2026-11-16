package service

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strings"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/logger"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/backpack"
)

// TunnelEdit holds the settings of a stored tunnel that can change. The entry end owns the shared ones (address,
// port, transports, forwarded ports, options) and the exit end owns its exit outbound and its own options.
type TunnelEdit struct {
	Address            string         `json:"address" example:"203.0.113.10"`
	Port               int            `json:"port" example:"8443"`
	ServerName         string         `json:"serverName" example:"www.example.com"`
	Transport          string         `json:"transport" example:"wssmux"`
	FallbackTransports []string       `json:"fallbackTransports"`
	ForwardPorts       []string       `json:"forwardPorts"`
	AcceptUDP          bool           `json:"acceptUdp" example:"false"`
	Options            map[string]any `json:"options"`
	RouteInbounds      []string       `json:"routeInbounds"`
	ExitOutbound       string         `json:"exitOutbound" example:"direct"`
}

// Update changes a tunnel in place. An entry end made for a node sends the node its end first, so a refusal
// there leaves both ends as they were.
func (s *TunnelService) Update(id int, edit TunnelEdit) (*model.ServerTunnel, error) {
	t, err := s.get(id)
	if err != nil {
		return nil, err
	}
	if t.ManagePort > 0 {
		return nil, errManagementTunnel
	}
	next := *t
	next.Options = maps.Clone(t.Options)
	if t.Role == model.TunnelRoleBridge {
		err = applyExitEdit(t, &next, edit)
	} else {
		err = s.applyEntryEdit(t, &next, edit)
	}
	if err != nil {
		return nil, err
	}
	if t.Role == model.TunnelRolePortal && t.NodeId > 0 {
		if err := s.pushToNode(&next); err != nil {
			return nil, fmt.Errorf("the node did not take the change to tunnel %q: %w", t.Name, err)
		}
	}
	if err := database.GetDB().Save(&next).Error; err != nil {
		if t.Role == model.TunnelRolePortal && t.NodeId > 0 {
			if pushErr := s.pushToNode(t); pushErr != nil {
				logger.Warning("tunnels: could not put back the node's end after a failed save:", pushErr)
			}
		}
		return nil, err
	}
	s.applyChange()
	return &next, nil
}

func applyExitEdit(old, next *model.ServerTunnel, edit TunnelEdit) error {
	next.ExitOutbound = strings.TrimSpace(edit.ExitOutbound)
	if next.Engine != model.TunnelEngineBackpack {
		return nil
	}
	next.Options = maps.Clone(edit.Options)
	if next.Mode == model.TunnelModeDirect {
		return keepLinkAddresses(next, old.Options)
	}
	// Rendering is the engine's own validation of the options.
	_, err := backpack.RenderConfig(*next)
	return err
}

// keepLinkAddresses leaves the pair of tunnel addresses as stored: both ends agree on them at setup.
func keepLinkAddresses(next *model.ServerTunnel, stored map[string]any) error {
	if next.Options == nil {
		next.Options = map[string]any{}
	}
	for _, key := range []string{"local_ip", "peer_ip"} {
		if _, ok := next.Options[key]; !ok {
			next.Options[key] = stored[key]
		}
	}
	_, err := backpack.RenderConfig(*next)
	return err
}

func (s *TunnelService) applyEntryEdit(old, next *model.ServerTunnel, edit TunnelEdit) error {
	next.Address, next.Port = strings.TrimSpace(edit.Address), edit.Port
	if next.Address == "" {
		return errors.New("tunnel address is required")
	}
	if next.Port < 1 || next.Port > 65535 {
		return fmt.Errorf("tunnel port %d is out of range", next.Port)
	}
	next.RouteInbounds = edit.RouteInbounds
	if next.Engine == model.TunnelEngineBackpack {
		next.ForwardPorts, next.AcceptUDP = edit.ForwardPorts, edit.AcceptUDP
		next.Options = maps.Clone(edit.Options)
		if next.Mode != model.TunnelModeDirect {
			next.Transport, next.FallbackTransports = edit.Transport, edit.FallbackTransports
		} else if err := keepLinkAddresses(next, old.Options); err != nil {
			return err
		}
	} else if next.Security == model.TunnelSecurityReality {
		if strings.TrimSpace(edit.ServerName) == "" {
			return errors.New("a REALITY tunnel needs a server name")
		}
		next.ServerName = strings.TrimSpace(edit.ServerName)
	}
	if next.Engine == model.TunnelEngineBackpack {
		if _, err := backpack.RenderConfig(*next); err != nil {
			return err
		}
	}
	if err := s.refuseChangeWhileCarrying(old, next); err != nil {
		return err
	}
	if next.Port != old.Port && listensOnPort(*next) {
		return s.checkPortFree(next.Port, old.Id)
	}
	return nil
}

// A node's management rides on the tunnel's address, port and identity, so those stay while it does.
func (s *TunnelService) refuseChangeWhileCarrying(old, next *model.ServerTunnel) error {
	if old.Address == next.Address && old.Port == next.Port && old.Transport == next.Transport &&
		old.ServerName == next.ServerName {
		return nil
	}
	var carried []string
	if err := database.GetDB().Model(&model.ServerTunnel{}).Where("parent_id = ?", old.Id).Pluck("name", &carried).Error; err != nil {
		return err
	}
	if len(carried) > 0 {
		return fmt.Errorf("this tunnel carries the management of %s; its address, port and transport cannot change", strings.Join(carried, ", "))
	}
	return nil
}

// pushToNode sends the node the current setup link; the node updates its end of the same name in place.
func (s *TunnelService) pushToNode(portal *model.ServerTunnel) error {
	remote, err := s.remoteFor(portal.NodeId)
	if err != nil {
		return err
	}
	link, err := tunnel.SetupLink(*portal)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), nodeCallTimeout)
	defer cancel()
	_, err = remote.ImportTunnel(ctx, link, "")
	return err
}

// updateDialingEnd applies a newer setup link to the stored dialing end of the same name.
func (s *TunnelService) updateDialingEnd(existing *model.ServerTunnel, fresh model.ServerTunnel) (*model.ServerTunnel, error) {
	if existing.Role != model.TunnelRoleBridge || existing.ManagePort > 0 {
		return nil, fmt.Errorf("a tunnel named %q already exists", existing.Name)
	}
	if fresh.RelayPort != existing.RelayPort && fresh.RelayPort > 0 {
		if err := s.checkRelayPortFree(fresh.RelayPort); err != nil {
			return nil, err
		}
	}
	if fresh.Mode == model.TunnelModeDirect {
		if err := s.checkLinkAddressesFree(&fresh, existing.Id); err != nil {
			return nil, err
		}
	}
	if fresh.ExitOutbound == "" {
		fresh.ExitOutbound = existing.ExitOutbound
	}
	fresh.Id, fresh.Enable = existing.Id, existing.Enable
	if err := database.GetDB().Save(&fresh).Error; err != nil {
		return nil, err
	}
	s.applyChange()
	return &fresh, nil
}
