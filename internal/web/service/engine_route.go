package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/util/link"
	"github.com/mhsanaei/3x-ui/v3/internal/util/netproxy"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service/outbound"
)

// The ways an engine download can reach its release server.
const (
	RoutePanel    = "panel"
	RouteDirect   = "direct"
	RouteOutbound = "outbound"
	RouteLink     = "link"

	engineRouteTag = "engine-route"
)

var errUnknownRoute = errors.New("unknown way to reach the release server")

// EngineRoute says how the panel reaches a release server while it installs an engine: as the panel is set,
// straight from this server, through one of the panel's outbounds (Value is its tag), or through a link (Value).
type EngineRoute struct {
	Via   string `json:"via" form:"via" example:"outbound"`
	Value string `json:"value" form:"value" example:"tunnel-de1"`
}

// EngineRouteChoice is one entry the install dialogs offer.
type EngineRouteChoice struct {
	Via   string `json:"via" example:"outbound"`
	Value string `json:"value" example:"tor-de"`
	// Kind groups the entry: panel, direct, tunnel, tor or outbound.
	Kind string `json:"kind" example:"tor"`
}

// EngineRouteService turns an EngineRoute into the HTTP client an engine download uses.
type EngineRouteService struct{}

var (
	openEngineProxy = func(chosen map[string]any, all []any) (string, func(), error) {
		return (&outbound.OutboundService{}).OpenProxy(chosen, all)
	}
	engineRouteOutbounds = runningOutbounds
)

// runningOutbounds lists the outbounds of the running Xray, which include the injected tunnels and Tor exits;
// without a running core it falls back to the stored template.
func runningOutbounds() []any {
	var raw []byte
	if proc := XrayProcess(); proc != nil && proc.IsRunning() {
		if cfg := proc.GetConfig(); cfg != nil {
			raw = cfg.OutboundConfigs
		}
	}
	if len(raw) == 0 {
		var template struct {
			Outbounds json.RawMessage `json:"outbounds"`
		}
		stored, _ := (&SettingService{}).GetXrayConfigTemplate()
		if json.Unmarshal([]byte(stored), &template) != nil {
			return nil
		}
		raw = template.Outbounds
	}
	var list []any
	_ = json.Unmarshal(raw, &list)
	return list
}

// Client returns the client for a route and a function that releases whatever the route started.
func (s *EngineRouteService) Client(route EngineRoute, timeout time.Duration) (*http.Client, func(), error) {
	switch route.Via {
	case "", RoutePanel:
		return (&SettingService{}).NewProxiedHTTPClient(timeout), func() {}, nil
	case RouteDirect:
		return &http.Client{Timeout: timeout}, func() {}, nil
	case RouteLink, RouteOutbound:
		proxyURL, closeRoute, err := s.resolve(route)
		if err != nil {
			return nil, nil, err
		}
		client, err := netproxy.NewHTTPClient(proxyURL, timeout)
		if err != nil {
			closeRoute()
			return nil, nil, err
		}
		return client, closeRoute, nil
	}
	return nil, nil, fmt.Errorf("%w: %q", errUnknownRoute, route.Via)
}

// Source is an engineSource that downloads through route; call release when the install is over.
func (s *EngineRouteService) Source(route EngineRoute) (src engineSource, release func(), err error) {
	client, release, err := s.Client(route, engineInstallTimeout)
	if err != nil {
		return nil, nil, err
	}
	return engineSourceWith(client), release, nil
}

// resolve gives the proxy URL for a link or outbound route; closeRoute stops any temporary core it started.
func (s *EngineRouteService) resolve(route EngineRoute) (proxyURL string, closeRoute func(), err error) {
	if route.Via == RouteLink {
		return s.resolveLink(strings.TrimSpace(route.Value))
	}
	tag := strings.TrimSpace(route.Value)
	all := engineRouteOutbounds()
	for _, item := range all {
		if chosen, ok := item.(map[string]any); ok && chosen["tag"] == tag {
			return s.openCore(chosen, all)
		}
	}
	return "", nil, fmt.Errorf("the panel has no outbound named %q (is Xray running?)", tag)
}

func (s *EngineRouteService) resolveLink(value string) (string, func(), error) {
	if parsed, err := url.Parse(value); err == nil {
		switch strings.ToLower(parsed.Scheme) {
		case "socks5", "socks5h", "http", "https":
			return value, func() {}, nil
		}
	}
	res, err := link.ParseLink(value)
	if err != nil {
		return "", nil, fmt.Errorf("that is neither a proxy address (socks5://, http://) nor a share link (vless://, vmess://, trojan://, ss://): %w", err)
	}
	chosen := map[string]any(res.Outbound)
	chosen["tag"] = engineRouteTag
	return s.openCore(chosen, nil)
}

func (s *EngineRouteService) openCore(chosen map[string]any, all []any) (string, func(), error) {
	addr, closeCore, err := openEngineProxy(chosen, all)
	if err != nil {
		return "", nil, err
	}
	return "socks5://" + addr, closeCore, nil
}

// Choices lists what the install dialogs offer: the panel's setting, direct, every usable outbound.
func (s *EngineRouteService) Choices() ([]EngineRouteChoice, error) {
	choices := []EngineRouteChoice{{Via: RoutePanel, Kind: "panel"}, {Via: RouteDirect, Kind: "direct"}}
	seen := map[string]bool{}
	for _, item := range engineRouteOutbounds() {
		ob, ok := item.(map[string]any)
		if !ok {
			continue
		}
		tag, _ := ob["tag"].(string)
		protocol, _ := ob["protocol"].(string)
		switch protocol {
		case "freedom", "blackhole", "dns", "loopback":
			continue
		}
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		choices = append(choices, EngineRouteChoice{Via: RouteOutbound, Value: tag, Kind: outboundKind(tag)})
	}
	return choices, nil
}

func outboundKind(tag string) string {
	switch {
	case strings.HasPrefix(tag, "tunnel-"):
		return "tunnel"
	case strings.HasPrefix(tag, "tor-"):
		return "tor"
	}
	return "outbound"
}
