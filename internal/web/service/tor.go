package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/config"
	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/logger"
	"github.com/mhsanaei/3x-ui/v3/internal/tor"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/engine"
)

// torLayout, torExits and torProgress are this panel's Tor state; tests point them at a temporary folder.
var (
	torLayout = tor.Layout{
		Root:      filepath.Join(config.GetDBFolderPath(), "tor"),
		BundleDir: filepath.Join(config.GetBinFolderPath(), "tor-bundle"),
	}
	torProgress tor.Progress
	torExits    = newTorExits(torLayout)

	torReleaseBase = tor.ReleaseBaseURL

	torProbe tor.Prober = tunnel.ProbeVia
	torNow              = time.Now
)

// torProbeKeep is how long a probe result is reused, so a page polling the list does not load Tor.
const torProbeKeep = 30 * time.Second

type torProbeResult struct {
	status tunnel.Status
	at     time.Time
}

var (
	torProbeMu    sync.Mutex
	torProbeCache = map[string]torProbeResult{}
)

func newTorExits(layout tor.Layout) *engine.Supervisor {
	return tor.NewSupervisor(layout, storedBridgeLines, &torProgress)
}

func storedBridgeLines() []string {
	text, err := (&SettingService{}).GetTorBridges()
	if err != nil {
		return nil
	}
	lines, err := tor.ValidateBridges(text)
	if err != nil {
		return nil
	}
	return lines
}

var errTorCountry = errors.New("a country is two letters, such as de or nl")

// TorService stores the countries Tor leaves through and the bridges every Tor uses to start.
type TorService struct {
	xrayService XrayService
}

// List returns every exit, oldest first.
func (s *TorService) List() ([]model.TorExit, error) {
	var exits []model.TorExit
	if err := database.GetDB().Order("id asc").Find(&exits).Error; err != nil {
		return nil, err
	}
	return exits, nil
}

func (s *TorService) get(id int) (*model.TorExit, error) {
	var exit model.TorExit
	if err := database.GetDB().First(&exit, id).Error; err != nil {
		return nil, err
	}
	return &exit, nil
}

// Add stores an enabled exit for a country on the lowest free loopback port.
func (s *TorService) Add(country string) (*model.TorExit, error) {
	cc, ok := tor.NormalizeCountry(country)
	if !ok {
		return nil, errTorCountry
	}
	db := database.GetDB()
	var same int64
	if err := db.Model(&model.TorExit{}).Where("country = ?", cc).Count(&same).Error; err != nil {
		return nil, err
	}
	if same > 0 {
		return nil, fmt.Errorf("%s already has an exit", cc)
	}
	if err := s.checkRoom(0); err != nil {
		return nil, err
	}
	port, err := s.freePort()
	if err != nil {
		return nil, err
	}
	exit := &model.TorExit{Country: cc, Port: port, Enable: true}
	if err := db.Create(exit).Error; err != nil {
		return nil, err
	}
	s.applyChange()
	return exit, nil
}

// checkRoom refuses another enabled exit once MaxExits run; except is the exit being changed.
func (s *TorService) checkRoom(except int) error {
	var enabled int64
	if err := database.GetDB().Model(&model.TorExit{}).Where("enable = ? AND id <> ?", true, except).Count(&enabled).Error; err != nil {
		return err
	}
	if enabled >= tor.MaxExits {
		return fmt.Errorf("at most %d Tor exits can be enabled at once; disable or delete one first", tor.MaxExits)
	}
	return nil
}

func (s *TorService) freePort() (int, error) {
	db := database.GetDB()
	taken := map[int]bool{}
	for _, query := range []struct {
		model  any
		column string
		where  string
	}{
		{&model.TorExit{}, "port", ""},
		{&model.Inbound{}, "port", "node_id IS NULL"},
		{&model.ServerTunnel{}, "relay_port", ""},
		{&model.ServerTunnel{}, "port", ""},
	} {
		var ports []int
		q := db.Model(query.model)
		if query.where != "" {
			q = q.Where(query.where)
		}
		if err := q.Pluck(query.column, &ports).Error; err != nil {
			return 0, err
		}
		for _, p := range ports {
			taken[p] = true
		}
	}
	for port := tor.PortBase; port <= tor.PortLimit; port++ {
		if !taken[port] {
			return port, nil
		}
	}
	return 0, errors.New("no free loopback port is left for a Tor exit")
}

// SetEnable turns one exit on or off.
func (s *TorService) SetEnable(id int, enable bool) error {
	exit, err := s.get(id)
	if err != nil {
		return err
	}
	if enable && !exit.Enable {
		if err := s.checkRoom(id); err != nil {
			return err
		}
	}
	if err := database.GetDB().Model(exit).Update("enable", enable).Error; err != nil {
		return err
	}
	s.applyChange()
	return nil
}

// Delete removes an exit unless routing, a balancer or the panel's own egress still names its outbound.
func (s *TorService) Delete(id int) error {
	exit, err := s.get(id)
	if err != nil {
		return err
	}
	if err := s.refuseWhileNamed(tor.OutboundTag(exit.Country)); err != nil {
		return err
	}
	if err := database.GetDB().Delete(&model.TorExit{}, id).Error; err != nil {
		return err
	}
	s.applyChange()
	torProgress.Reset(exit.Country)
	return os.RemoveAll(torLayout.DataDir(exit.Country))
}

// refuseWhileNamed keeps an exit whose outbound a routing rule, a balancer selector or the panel egress uses.
func (s *TorService) refuseWhileNamed(tag string) error {
	settings := &SettingService{}
	template, err := settings.GetXrayConfigTemplate()
	if err != nil {
		return err
	}
	var parsed struct {
		Routing struct {
			Rules []struct {
				OutboundTag string `json:"outboundTag"`
			} `json:"rules"`
			Balancers []struct {
				Selector []string `json:"selector"`
			} `json:"balancers"`
		} `json:"routing"`
	}
	if err := json.Unmarshal([]byte(UnwrapXrayTemplateConfig(template)), &parsed); err != nil {
		return fmt.Errorf("the Xray template cannot be read, so %s cannot be checked: %w", tag, err)
	}
	for _, rule := range parsed.Routing.Rules {
		if rule.OutboundTag == tag {
			return fmt.Errorf("%s is used by a routing rule; remove the rule first", tag)
		}
	}
	for _, balancer := range parsed.Routing.Balancers {
		for _, selector := range balancer.Selector {
			if selector != "" && strings.HasPrefix(tag, selector) {
				return fmt.Errorf("%s is selected by a balancer; remove it from the balancer first", tag)
			}
		}
	}
	if egress, err := settings.GetPanelOutbound(); err == nil && egress == tag {
		return fmt.Errorf("%s is the panel's own egress; choose another one first", tag)
	}
	return nil
}

// Bridges returns the bridge lines every Tor uses, one per line; empty means Tor connects directly.
func (s *TorService) Bridges() (string, error) {
	return (&SettingService{}).GetTorBridges()
}

// SetBridges stores validated bridge lines and restarts the Tors that read them.
func (s *TorService) SetBridges(text string) error {
	lines, err := tor.ValidateBridges(text)
	if err != nil {
		return err
	}
	if err := (&SettingService{}).SetTorBridges(strings.Join(lines, "\n")); err != nil {
		return err
	}
	s.applyChange()
	return nil
}

// applyChange lets Xray and the Tor processes pick up a stored change; a missing bundle is reported by Status.
func (s *TorService) applyChange() {
	torProbeMu.Lock()
	clear(torProbeCache)
	torProbeMu.Unlock()
	s.xrayService.SetToNeedRestart()
	if err := s.ReconcileExits(); err != nil {
		logger.Warning("tor exits:", err)
	}
}

// ReconcileExits starts and stops Tor processes until they match the stored exits.
func (s *TorService) ReconcileExits() error {
	exits, err := s.List()
	if err != nil {
		return err
	}
	for _, exit := range exits {
		if !exit.Enable {
			torProgress.Reset(exit.Country)
		}
	}
	return torExits.Reconcile(tor.ExitRows(exits))
}

// StopExits stops every Tor process this panel started.
func (s *TorService) StopExits() { torExits.StopAll() }

// EngineInfo says whether the Tor bundle is on this server, which release, and which one Dalan installs.
func (s *TorService) EngineInfo() (EngineInfo, error) {
	_, err := os.Stat(torLayout.Binary())
	return EngineInfo{
		Engine: "tor", Installed: err == nil,
		Version: tor.InstalledVersion(torLayout.BundleDir), Pinned: tor.PinnedVersion,
	}, nil
}

// InstallEngine downloads the pinned bundle through the panel's proxy settings and starts the enabled exits.
func (s *TorService) InstallEngine() error {
	ctx, cancel := context.WithTimeout(context.Background(), engineInstallTimeout)
	defer cancel()
	client := (&SettingService{}).NewProxiedHTTPClient(engineInstallTimeout)
	rel, err := tor.Fetch(ctx, client, torReleaseBase, tor.PinnedVersion, runtime.GOARCH)
	if err != nil {
		return err
	}
	if err := tor.Place(rel, tor.PinnedVersion, runtime.GOARCH, torLayout.BundleDir); err != nil {
		return err
	}
	return s.restartExits()
}

// RollbackEngine puts back the bundle the last install replaced.
func (s *TorService) RollbackEngine() error {
	torExits.StopAll()
	if err := tor.Rollback(torLayout.BundleDir); err != nil {
		return err
	}
	return s.restartExits()
}

// restartExits stops every Tor, so the ones Reconcile starts next run the bundle now on disk.
func (s *TorService) restartExits() error {
	torExits.StopAll()
	return s.ReconcileExits()
}

// TorOutbound is an enabled exit as Xray routing sees it: an outbound that rules and balancers can target.
type TorOutbound struct {
	Tag     string `json:"tag" example:"tor-de"`
	Country string `json:"country" example:"de"`
	Enable  bool   `json:"enable" example:"true"`
}

// Outbounds lists the enabled exits, the only ones that exist as outbounds in the generated config.
func (s *TorService) Outbounds() ([]TorOutbound, error) {
	exits, err := s.List()
	if err != nil {
		return nil, err
	}
	out := make([]TorOutbound, 0, len(exits))
	for _, exit := range exits {
		if exit.Enable {
			out = append(out, TorOutbound{Tag: tor.OutboundTag(exit.Country), Country: exit.Country, Enable: true})
		}
	}
	return out, nil
}

// StatusAll judges every exit by country. Probes run side by side and are reused for torProbeKeep.
func (s *TorService) StatusAll() (map[string]tor.Status, error) {
	exits, err := s.List()
	if err != nil {
		return nil, err
	}
	_, statErr := os.Stat(torLayout.Binary())
	statuses := make(map[string]tor.Status, len(exits))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, exit := range exits {
		progress, seen := torProgress.Get(exit.Country)
		observed := tor.Observation{
			Port: exit.Port, Installed: statErr == nil, Running: torExits.Running(tor.OutboundTag(exit.Country)),
			Progress: progress, HasProgress: seen,
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			status := tor.StatusOf(observed, cachedProbe(exit.Country))
			mu.Lock()
			statuses[exit.Country] = status
			mu.Unlock()
		}()
	}
	wg.Wait()
	return statuses, nil
}

// cachedProbe wraps the real prober so one country is probed at most once per torProbeKeep.
func cachedProbe(country string) tor.Prober {
	return func(addr, url string, timeout time.Duration) tunnel.Status {
		torProbeMu.Lock()
		hit, ok := torProbeCache[country]
		torProbeMu.Unlock()
		if ok && torNow().Sub(hit.at) < torProbeKeep {
			return hit.status
		}
		result := torProbe(addr, url, timeout)
		torProbeMu.Lock()
		torProbeCache[country] = torProbeResult{status: result, at: torNow()}
		torProbeMu.Unlock()
		return result
	}
}

// torTrialTimeout bounds one Test of a country that is not running.
const torTrialTimeout = 90 * time.Second

// Test answers whether Tor can leave through a country. A country that already runs is asked directly;
// any other gets a throwaway Tor that is removed afterwards.
func (s *TorService) Test(country string) (tor.TrialResult, error) {
	cc, ok := tor.NormalizeCountry(country)
	if !ok {
		return tor.TrialResult{}, errTorCountry
	}
	if _, err := os.Stat(torLayout.Binary()); err != nil {
		return tor.TrialResult{}, tor.ErrNotInstalled
	}
	exits, err := s.List()
	if err != nil {
		return tor.TrialResult{}, err
	}
	for _, exit := range exits {
		if exit.Country == cc && torExits.Running(tor.OutboundTag(cc)) {
			progress, seen := torProgress.Get(cc)
			status := tor.StatusOf(tor.Observation{Port: exit.Port, Installed: true, Running: true, Progress: progress, HasProgress: seen}, torProbe)
			return trialOf(status), nil
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), torTrialTimeout)
	defer cancel()
	return tor.Trial(ctx, cc, storedBridgeLines(), torLayout, torProbe)
}

func trialOf(status tor.Status) tor.TrialResult {
	switch status.State {
	case tor.StateConnected:
		return tor.TrialResult{OK: true, LatencyMs: status.LatencyMs, ExitIP: status.ExitIP, ExitLoc: status.ExitLoc}
	case tor.StateStarting:
		return tor.TrialResult{Reason: fmt.Sprintf("Tor is still connecting (%d%%)", status.Progress)}
	default:
		return tor.TrialResult{Reason: "Tor is running, but a request through it failed", Error: status.Error}
	}
}
