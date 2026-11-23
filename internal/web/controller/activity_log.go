package controller

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
)

// activityRoute names the change a successful POST to a route makes.
type activityRoute struct {
	action string
	// toggle appends ".enable" or ".disable" from the body's "enable" field.
	toggle bool
	// nodeParam: the :id param is a node id, so the entry is filed under that node.
	nodeParam bool
}

const apiPrefix = "/panel/api"

var activityRoutes = map[string]activityRoute{
	"/clients/add":                 {action: "client.add"},
	"/clients/update/:email":       {action: "client.update"},
	"/clients/del/:email":          {action: "client.delete"},
	"/clients/bulkCreate":          {action: "client.bulkAdd"},
	"/clients/bulkDel":             {action: "client.bulkDelete"},
	"/clients/bulkAdjust":          {action: "client.bulkAdjust"},
	"/clients/bulkEnable":          {action: "client.bulkEnable"},
	"/clients/bulkDisable":         {action: "client.bulkDisable"},
	"/clients/resetTraffic/:email": {action: "client.resetTraffic"},
	"/clients/groups/create":       {action: "group.add"},
	"/clients/groups/delete":       {action: "group.delete"},
	"/inbounds/add":                {action: "inbound.add"},
	"/inbounds/update/:id":         {action: "inbound.update"},
	"/inbounds/del/:id":            {action: "inbound.delete"},
	"/inbounds/setEnable/:id":      {action: "inbound", toggle: true},
	"/tunnels/add":                 {action: "tunnel.add"},
	"/tunnels/update/:id":          {action: "tunnel.update"},
	"/tunnels/del/:id":             {action: "tunnel.delete"},
	"/tunnels/setEnable/:id":       {action: "tunnel", toggle: true},
	"/nodes/add":                   {action: "node.add"},
	"/nodes/update/:id":            {action: "node.update", nodeParam: true},
	"/nodes/del/:id":               {action: "node.delete", nodeParam: true},
	"/nodes/setEnable/:id":         {action: "node", toggle: true, nodeParam: true},
	"/hosts/add":                   {action: "host.add"},
	"/hosts/update/:groupId":       {action: "host.update"},
	"/hosts/del/:groupId":          {action: "host.delete"},
	"/tor/add":                     {action: "tor.add"},
	"/tor/del/:id":                 {action: "tor.delete"},
	"/server/restartXrayService":   {action: "xray.restart"},
	"/server/stopXrayService":      {action: "xray.stop"},
	"/server/installXray/:version": {action: "xray.install"},
	"/server/updatePanel":          {action: "panel.update"},
	"/setting/update":              {action: "settings.update"},
	"/setting/updateUser":          {action: "account.update"},
	"/xray/update":                 {action: "xray.config"},
}

// The body is read before the handler, so cap it: activity targets sit near the top.
const activityBodyPeek = 64 << 10

type successPeek struct {
	gin.ResponseWriter
	head []byte
}

func (w *successPeek) Write(b []byte) (int, error) {
	if len(w.head) < 64 {
		w.head = append(w.head, b[:min(len(b), 64-len(w.head))]...)
	}
	return w.ResponseWriter.Write(b)
}

// activityTarget picks what the change was about: a route param first, then the
// usual name fields of the body, then the size of a bulk list.
func activityTarget(c *gin.Context, body map[string]any) string {
	if v := c.Param("email"); v != "" {
		return v
	}
	for _, k := range []string{"email", "remark", "name", "tag", "country", "version"} {
		if s, ok := body[k].(string); ok && s != "" {
			return s
		}
	}
	for _, k := range []string{"emails", "ids", "clients"} {
		if list, ok := body[k].([]any); ok {
			return strconv.Itoa(len(list))
		}
	}
	for _, k := range []string{"id", "groupId", "version"} {
		if v := c.Param(k); v != "" {
			return "#" + v
		}
	}
	return ""
}

// ActivityLog records each successful change made through the API in the activity log.
func ActivityLog(activity *service.ActivityService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodPost {
			c.Next()
			return
		}
		route, ok := activityRoutes[strings.TrimPrefix(c.FullPath(), apiPrefix)]
		if !ok {
			c.Next()
			return
		}
		var body map[string]any
		if c.Request.Body != nil {
			raw, _ := io.ReadAll(io.LimitReader(c.Request.Body, activityBodyPeek))
			rest, _ := io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewReader(append(raw, rest...)))
			_ = json.Unmarshal(raw, &body)
			if body == nil {
				_ = c.Request.ParseForm()
				body = map[string]any{}
				for k := range c.Request.PostForm {
					body[k] = c.Request.PostForm.Get(k)
				}
				c.Request.Body = io.NopCloser(bytes.NewReader(append(raw, rest...)))
			}
		}
		peek := &successPeek{ResponseWriter: c.Writer}
		c.Writer = peek
		c.Next()
		if c.Writer.Status() != http.StatusOK || !bytes.Contains(peek.head, []byte(`"success":true`)) {
			return
		}
		action := route.action
		if route.toggle {
			on, _ := body["enable"].(bool)
			if s, ok := body["enable"].(string); ok {
				on = s == "true"
			}
			action += map[bool]string{true: ".enable", false: ".disable"}[on]
		}
		nodeId := 0
		if route.nodeParam {
			nodeId, _ = strconv.Atoi(c.Param("id"))
		}
		actor := model.ActorYou
		if c.GetBool("api_authed") {
			actor = model.ActorAPI
		}
		activity.Record(actor, action, activityTarget(c, body), nodeId)
	}
}
