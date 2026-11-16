package controller

import (
	"strconv"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel"
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
	g.GET("/pools/list", a.poolList)

	g.POST("/add", a.add)
	g.POST("/import", a.importLink)
	g.POST("/engine/install", a.installEngine)
	g.POST("/setEnable/:id", a.setEnable)
	g.POST("/del/:id", a.del)
	g.POST("/dial", a.dial)
	g.POST("/routetest", a.routeTest)
	g.POST("/pools/add", a.poolAdd)
	g.POST("/pools/del/:id", a.poolDel)
}

const dialTimeout = 5 * time.Second

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
	results, err := tunnel.DialPorts(req.Host, req.Ports, dialTimeout)
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
	tunnels, err := a.tunnelService.GetAll()
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

		Engine             string   `json:"engine"`
		Transport          string   `json:"transport"`
		FallbackTransports []string `json:"fallbackTransports"`
		ForwardPorts       []string `json:"forwardPorts"`
		AcceptUDP          bool     `json:"acceptUdp"`

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
		ForwardPorts: req.ForwardPorts, AcceptUDP: req.AcceptUDP,
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

func (a *TunnelController) engine(c *gin.Context) {
	jsonObj(c, a.tunnelService.EngineInfo(), nil)
}

func (a *TunnelController) installEngine(c *gin.Context) {
	jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.installEngine"), a.tunnelService.InstallEngine())
}
