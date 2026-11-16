package controller

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/backpack"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/rakhsh"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"

	"github.com/gin-gonic/gin"
)

type TunnelController struct {
	tunnelService service.TunnelService
	poolService   service.TunnelPoolService
}

func NewTunnelController(g *gin.RouterGroup) *TunnelController {
	a := &TunnelController{}
	a.initRouter(g)
	return a
}

func (a *TunnelController) initRouter(g *gin.RouterGroup) {
	g.GET("/list", a.list)
	g.GET("/link/:id", a.link)
	g.GET("/status/:id", a.status)
	g.GET("/engine", a.engine)
	g.GET("/options", a.options)
	g.GET("/pools/list", a.poolList)

	g.POST("/add", a.add)
	g.POST("/import", a.importLink)
	g.POST("/engine/install", a.installEngine)
	g.POST("/engine/update", a.updateEngine)
	g.POST("/engine/release", a.engineRelease)
	g.POST("/update/:id", a.update)
	g.POST("/setEnable/:id", a.setEnable)
	g.POST("/del/:id", a.del)
	g.POST("/dial", a.dial)
	g.POST("/routetest", a.routeTest)
	g.POST("/pools/add", a.poolAdd)
	g.POST("/pools/del/:id", a.poolDel)
}

const (
	dialTimeout         = 5 * time.Second
	engineUpdateTimeout = 8 * time.Minute
)

// dial runs on the node during a route test: it tries each port from this server's side.
func (a *TunnelController) dial(c *gin.Context) {
	var req struct {
		Host  string `json:"host"`
		Ports []int  `json:"ports"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.routeTest"), err)
		return
	}
	results, err := tunnel.DialPorts(c.Request.Context(), req.Host, req.Ports, dialTimeout)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.routeTest"), err)
		return
	}
	jsonObj(c, results, nil)
}

func (a *TunnelController) routeTest(c *gin.Context) {
	var req struct {
		NodeId  int    `json:"nodeId"`
		Address string `json:"address"`
		Ports   []int  `json:"ports"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.routeTest"), err)
		return
	}
	results, err := a.tunnelService.RouteTest(req.NodeId, req.Address, req.Ports)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.routeTest"), err)
		return
	}
	jsonObj(c, results, nil)
}

func (a *TunnelController) poolList(c *gin.Context) {
	pools, err := a.poolService.List()
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.list"), err)
		return
	}
	jsonObj(c, pools, nil)
}

func (a *TunnelController) poolAdd(c *gin.Context) {
	var req struct {
		Name          string   `json:"name"`
		Members       []string `json:"members"`
		RouteInbounds []string `json:"routeInbounds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.add"), err)
		return
	}
	pool, err := a.poolService.Create(req.Name, req.Members, req.RouteInbounds)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.add"), err)
		return
	}
	jsonMsgObj(c, I18nWeb(c, "pages.tunnels.toasts.add"), pool, nil)
}

func (a *TunnelController) poolDel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.del"), err)
		return
	}
	jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.del"), a.poolService.Delete(id))
}

func (a *TunnelController) list(c *gin.Context) {
	tunnels, err := a.tunnelService.ListWithTraffic()
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.list"), err)
		return
	}
	jsonObj(c, tunnels, nil)
}

func (a *TunnelController) link(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.link"), err)
		return
	}
	link, err := a.tunnelService.SetupLink(id)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.link"), err)
		return
	}
	jsonObj(c, link, nil)
}

func (a *TunnelController) add(c *gin.Context) {
	var req struct {
		Name          string   `json:"name"`
		Address       string   `json:"address"`
		Port          int      `json:"port"`
		Security      string   `json:"security"`
		ServerName    string   `json:"serverName"`
		RouteInbounds []string `json:"routeInbounds"`

		Engine             string         `json:"engine"`
		Transport          string         `json:"transport"`
		FallbackTransports []string       `json:"fallbackTransports"`
		ForwardPorts       []string       `json:"forwardPorts"`
		AcceptUDP          bool           `json:"acceptUdp"`
		Mode               string         `json:"mode"`
		Options            map[string]any `json:"options"`

		NodeId       int    `json:"nodeId"`
		ExitOutbound string `json:"exitOutbound"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.add"), err)
		return
	}
	spec := tunnel.PortalSpec{
		Name: req.Name, Address: req.Address, Port: req.Port, Security: req.Security, ServerName: req.ServerName,
		Engine: req.Engine, Transport: req.Transport, FallbackTransports: req.FallbackTransports,
		ForwardPorts: req.ForwardPorts, AcceptUDP: req.AcceptUDP, Mode: req.Mode, Options: req.Options,
	}
	var (
		portal *model.ServerTunnel
		err    error
	)
	if req.NodeId > 0 {
		portal, err = a.tunnelService.CreateOnNode(req.NodeId, spec, req.RouteInbounds, req.ExitOutbound)
	} else {
		portal, err = a.tunnelService.CreatePortal(spec, req.RouteInbounds)
	}
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.add"), err)
		return
	}
	jsonMsgObj(c, I18nWeb(c, "pages.tunnels.toasts.add"), portal, nil)
}

// options lists every setting of an engine the panel can write (the engine query is backpack or rakhsh,
// backpack by default), for both tunnel modes.
func (a *TunnelController) options(c *gin.Context) {
	switch c.DefaultQuery("engine", model.TunnelEngineBackpack) {
	case model.TunnelEngineBackpack:
		jsonObj(c, backpack.Catalog, nil)
	case model.TunnelEngineRakhsh:
		jsonObj(c, rakhsh.Catalog, nil)
	default:
		jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.list"), errUnknownEngine)
	}
}

func (a *TunnelController) update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.update"), err)
		return
	}
	var edit service.TunnelEdit
	if err := c.ShouldBindJSON(&edit); err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.update"), err)
		return
	}
	updated, err := a.tunnelService.Update(id, edit)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.update"), err)
		return
	}
	jsonMsgObj(c, I18nWeb(c, "pages.tunnels.toasts.update"), updated, nil)
}

func (a *TunnelController) importLink(c *gin.Context) {
	var req struct {
		Link         string `json:"link"`
		ExitOutbound string `json:"exitOutbound"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.import"), err)
		return
	}
	bridge, err := a.tunnelService.Import(req.Link, req.ExitOutbound)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.import"), err)
		return
	}
	jsonMsgObj(c, I18nWeb(c, "pages.tunnels.toasts.import"), bridge, nil)
}

func (a *TunnelController) setEnable(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.update"), err)
		return
	}
	body := struct {
		Enable bool `json:"enable" form:"enable"`
	}{}
	if err := c.ShouldBind(&body); err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.update"), err)
		return
	}
	jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.update"), a.tunnelService.SetEnable(id, body.Enable))
}

func (a *TunnelController) del(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.del"), err)
		return
	}
	if c.Query("here") == "true" {
		jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.del"), a.tunnelService.DeleteHere(id))
		return
	}
	jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.del"), a.tunnelService.Delete(id))
}

func (a *TunnelController) status(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.status"), err)
		return
	}
	status, err := a.tunnelService.Status(id)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.status"), err)
		return
	}
	jsonObj(c, status, nil)
}

var errUnknownEngine = errors.New("the engine query must be backpack or rakhsh")

func (a *TunnelController) engine(c *gin.Context) {
	info, err := a.tunnelService.EngineInfo(c.DefaultQuery("engine", model.TunnelEngineBackpack))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.updateEngine"), err)
		return
	}
	jsonObj(c, info, nil)
}

func (a *TunnelController) updateEngine(c *gin.Context) {
	var req struct {
		NodeId  int    `json:"nodeId"`
		Engine  string `json:"engine"`
		Version string `json:"version"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.updateEngine"), err)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), engineUpdateTimeout)
	defer cancel()
	if err := a.tunnelService.UpdateEngine(ctx, req.NodeId, req.Engine, req.Version); err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.updateEngine"), err)
		return
	}
	info, err := a.tunnelService.EngineInfo(req.Engine)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.updateEngine"), err)
		return
	}
	jsonObj(c, info, nil)
}

func (a *TunnelController) engineRelease(c *gin.Context) {
	var req struct {
		Engine  string `json:"engine"`
		Version string `json:"version"`
		Offset  int    `json:"offset"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.updateEngine"), err)
		return
	}
	chunk, err := a.tunnelService.EngineReleaseChunk(req.Engine, req.Version, req.Offset)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.updateEngine"), err)
		return
	}
	jsonObj(c, chunk, nil)
}

func (a *TunnelController) installEngine(c *gin.Context) {
	var req struct {
		Engine string `json:"engine"`
	}
	// The body is optional: an empty one installs BackPack, as before there was a second engine.
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.installEngine"), err)
			return
		}
	}
	jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.installEngine"), a.tunnelService.InstallEngine(req.Engine))
}
