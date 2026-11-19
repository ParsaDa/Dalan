package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
)

type TorController struct {
	torService service.TorService
}

func NewTorController(g *gin.RouterGroup) *TorController {
	a := &TorController{}
	a.initRouter(g)
	return a
}

func (a *TorController) initRouter(g *gin.RouterGroup) {
	g.GET("/list", a.list)
	g.GET("/engine", a.engine)
	g.GET("/bridges", a.bridges)

	g.POST("/add", a.add)
	g.POST("/del/:id", a.del)
	g.POST("/setEnable/:id", a.setEnable)
	g.POST("/test", a.test)
	g.POST("/engine/install", a.installEngine)
	g.POST("/engine/rollback", a.rollbackEngine)
	g.POST("/bridges", a.setBridges)
}

type torCountryBody struct {
	Country string `json:"country" form:"country"`
}

func (a *TorController) list(c *gin.Context) {
	exits, err := a.torService.ListWithStatus()
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tor.toasts.list"), err)
		return
	}
	jsonObj(c, exits, nil)
}

func (a *TorController) add(c *gin.Context) {
	var body torCountryBody
	if err := c.ShouldBind(&body); err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tor.toasts.add"), err)
		return
	}
	exit, err := a.torService.Add(body.Country)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tor.toasts.add"), err)
		return
	}
	jsonMsgObj(c, I18nWeb(c, "pages.tor.toasts.add"), exit, nil)
}

func (a *TorController) del(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tor.toasts.del"), err)
		return
	}
	jsonMsg(c, I18nWeb(c, "pages.tor.toasts.del"), a.torService.Delete(id))
}

func (a *TorController) setEnable(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tor.toasts.update"), err)
		return
	}
	var body struct {
		Enable bool `json:"enable" form:"enable"`
	}
	if err := c.ShouldBind(&body); err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tor.toasts.update"), err)
		return
	}
	jsonMsg(c, I18nWeb(c, "pages.tor.toasts.update"), a.torService.SetEnable(id, body.Enable))
}

func (a *TorController) test(c *gin.Context) {
	var body torCountryBody
	if err := c.ShouldBind(&body); err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tor.toasts.test"), err)
		return
	}
	result, err := a.torService.Test(c.Request.Context(), body.Country)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tor.toasts.test"), err)
		return
	}
	jsonObj(c, result, nil)
}

func (a *TorController) engine(c *gin.Context) {
	info, err := a.torService.EngineInfo()
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tor.toasts.engine"), err)
		return
	}
	jsonObj(c, info, nil)
}

func (a *TorController) installEngine(c *gin.Context) {
	var req struct {
		Route service.EngineRoute `json:"route"`
	}
	// The body is optional: an empty one installs as the panel is set.
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			jsonMsg(c, I18nWeb(c, "pages.tor.toasts.installEngine"), err)
			return
		}
	}
	jsonMsg(c, I18nWeb(c, "pages.tor.toasts.installEngine"), a.torService.InstallEngine(req.Route))
}

func (a *TorController) rollbackEngine(c *gin.Context) {
	jsonMsg(c, I18nWeb(c, "pages.tor.toasts.rollbackEngine"), a.torService.RollbackEngine())
}

func (a *TorController) bridges(c *gin.Context) {
	text, err := a.torService.Bridges()
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tor.toasts.bridges"), err)
		return
	}
	jsonObj(c, gin.H{"bridges": text}, nil)
}

func (a *TorController) setBridges(c *gin.Context) {
	var body struct {
		Bridges string `json:"bridges" form:"bridges"`
	}
	if err := c.ShouldBind(&body); err != nil {
		jsonMsg(c, I18nWeb(c, "pages.tor.toasts.bridges"), err)
		return
	}
	jsonMsg(c, I18nWeb(c, "pages.tor.toasts.bridges"), a.torService.SetBridges(body.Bridges))
}
