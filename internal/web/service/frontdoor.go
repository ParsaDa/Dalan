package service

import (
	"sync"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/config"
	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/frontdoor"
	"github.com/mhsanaei/3x-ui/v3/internal/logger"
)

// frontDoorTTL is how long a table is reused; inbounds change rarely and every request asks for it.
var frontDoorTTL = 5 * time.Second

// FrontDoorService builds the routing table of the single public HTTP port from the panel's own
// settings and inbounds, so an inbound added in the panel is reachable at once.
type FrontDoorService struct {
	settings SettingService

	mu      sync.Mutex
	table   frontdoor.Table
	builtAt time.Time
}

func (s *FrontDoorService) Table() frontdoor.Table {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.builtAt.IsZero() && time.Since(s.builtAt) < frontDoorTTL {
		return s.table
	}
	s.table = s.build()
	s.builtAt = time.Now()
	return s.table
}

func (s *FrontDoorService) build() frontdoor.Table {
	var t frontdoor.Table
	t.PanelPort, _ = s.settings.GetPort()
	if port, configured, err := config.GetPortOverride(); configured && err == nil {
		t.PanelPort = port
	}
	t.PanelPath, _ = s.settings.GetBasePath()
	t.SubPort, _ = s.settings.GetSubPort()
	for _, get := range []func() (string, error){s.settings.GetSubPath, s.settings.GetSubJsonPath, s.settings.GetSubClashPath} {
		if path, err := get(); err == nil && path != "" {
			t.SubPaths = append(t.SubPaths, path)
		}
	}
	var inbounds []model.Inbound
	if err := database.GetDB().Find(&inbounds).Error; err != nil {
		logger.Warning("front door: reading inbounds:", err)
	}
	t.Inbounds = frontdoor.InboundRoutes(inbounds)
	return t
}
