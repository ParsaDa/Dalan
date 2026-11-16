package controller

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/logger"
	"github.com/mhsanaei/3x-ui/v3/internal/tunnel/enrol"
	"github.com/mhsanaei/3x-ui/v3/internal/web/middleware"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service/enrolment"

	"github.com/gin-gonic/gin"
)

const enrolmentBodyLimit = 16 << 10

type EnrolmentController struct {
	enrolments enrolment.Service
}

// NewEnrolmentController registers the operator routes on api and the node-facing route on the panel's base group.
func NewEnrolmentController(api, base *gin.RouterGroup) *EnrolmentController {
	a := &EnrolmentController{}
	api.GET("/list", a.list)
	api.POST("/add", a.add)
	api.POST("/del/:id", a.del)
	// Public on purpose: a node holds no session, and the sealed body is its credential.
	base.POST("/"+enrol.RedeemPath, middleware.MaxBodyBytes(enrolmentBodyLimit), a.redeem)
	return a
}

func (a *EnrolmentController) list(c *gin.Context) {
	rows, err := a.enrolments.List()
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.enrolments.toasts.list"), err)
		return
	}
	if rows == nil {
		rows = []model.NodeEnrolment{}
	}
	jsonObj(c, rows, nil)
}

func (a *EnrolmentController) add(c *gin.Context) {
	var spec enrolment.Spec
	if err := c.ShouldBindJSON(&spec); err != nil {
		jsonMsg(c, I18nWeb(c, "pages.enrolments.toasts.add"), err)
		return
	}
	created, err := a.enrolments.Create(spec)
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.enrolments.toasts.add"), err)
		return
	}
	jsonMsgObj(c, I18nWeb(c, "pages.enrolments.toasts.add"), created, nil)
}

func (a *EnrolmentController) del(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, I18nWeb(c, "pages.enrolments.toasts.del"), err)
		return
	}
	jsonMsg(c, I18nWeb(c, "pages.enrolments.toasts.del"), a.enrolments.Delete(id))
}

func (a *EnrolmentController) redeem(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusRequestEntityTooLarge, "enrolment request is too large")
		return
	}
	reply, err := a.enrolments.Redeem(body, c.ClientIP())
	switch {
	case errors.Is(err, enrolment.ErrGone):
		c.String(http.StatusGone, err.Error())
	case err != nil:
		logger.Warning("enrolment failed:", err)
		c.String(http.StatusInternalServerError, "the main panel could not complete the enrolment; see its log")
	default:
		c.Data(http.StatusOK, "application/json", reply)
	}
}
