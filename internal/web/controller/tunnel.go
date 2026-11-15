package controller

import (
	"strconv"

	"github.com/mhsanaei/3x-ui/v3/internal/tunnel"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"

	"github.com/gin-gonic/gin"
)

type TunnelController struct {
	tunnelService service.TunnelService
}

func NewTunnelController(g *gin.RouterGroup) *TunnelController {
	a := &TunnelController{}
	a.initRouter(g)
	return a
}

func (a *TunnelController) initRouter(g *gin.RouterGroup) {
	g.GET("/list", a.list)
	g.GET("/link/:id", a.link)

	g.POST("/add", a.add)
	g.POST("/import", a.importLink)
	g.POST("/setEnable/:id", a.setEnable)
	g.POST("/del/:id", a.del)
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
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.add"), err)
		return
	}
	portal, err := a.tunnelService.CreatePortal(tunnel.PortalSpec{
		Name: req.Name, Address: req.Address, Port: req.Port, Security: req.Security, ServerName: req.ServerName,
	}, req.RouteInbounds)
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
	jsonMsg(c, I18nWeb(c, "pages.tunnels.toasts.del"), a.tunnelService.Delete(id))
}
