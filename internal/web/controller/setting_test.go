package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/dbtest"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/util/crypto"
	"github.com/mhsanaei/3x-ui/v3/internal/web/locale"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service/discord"
	"github.com/mhsanaei/3x-ui/v3/internal/web/session"
)

func TestValidateRegex(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewSettingController(router.Group("/panel/api"))

	tests := []struct {
		name    string
		body    string
		success bool
	}{
		{name: "Go RE2 inline flag", body: `{"regex":"(?m)^general-purpose$"}`, success: true},
		{name: "invalid expression", body: `{"regex":"["}`, success: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/panel/api/setting/validateRegex", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			if resp.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body=%s", resp.Code, resp.Body.String())
			}
			needle := `"success":true`
			if !tt.success {
				needle = `"success":false`
			}
			if !strings.Contains(resp.Body.String(), needle) {
				t.Fatalf("body = %s, want %s", resp.Body.String(), needle)
			}
		})
	}
}

func TestAPITokenMutationRoutesEnforceExpectedScope(t *testing.T) {
	t.Setenv("XUI_DB_FOLDER", t.TempDir())
	dbtest.InitDB(t, filepath.Join(t.TempDir(), "x-ui.db"))
	row := &model.ApiToken{Name: "route-scope", Token: crypto.HashTokenSHA256("token"), Enabled: true, Scope: model.ApiScopeNodeSync}
	if err := database.GetDB().Create(row).Error; err != nil {
		t.Fatalf("seed token: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewSettingController(router.Group("/panel/api"))
	for _, path := range []string{
		"/panel/api/setting/apiTokens/delete/" + strconv.Itoa(row.Id),
		"/panel/api/setting/apiTokens/setEnabled/" + strconv.Itoa(row.Id),
	} {
		body := `{"expectedScope":"admin","enabled":false}`
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if !strings.Contains(resp.Body.String(), `"success":false`) {
			t.Fatalf("%s accepted wrong expected scope: %s", path, resp.Body.String())
		}
	}
	var stored model.ApiToken
	if err := database.GetDB().First(&stored, row.Id).Error; err != nil {
		t.Fatalf("token was deleted by wrong scope: %v", err)
	}
	if !stored.Enabled {
		t.Fatal("token was disabled by wrong scope")
	}
}

// GHSA-xqqw-jqqv-99h6: a save that keeps 2FA enabled must not be able to
// rebind the authenticator without presenting a current code.
func TestUpdateSettingRequiresCodeToReplaceTwoFactorToken(t *testing.T) {
	t.Setenv("XUI_DB_FOLDER", t.TempDir())
	dbtest.InitDB(t, filepath.Join(t.TempDir(), "x-ui.db"))

	settingService := service.SettingService{}
	if err := settingService.SetTwoFactorToken("ORIGINALSECRET234567"); err != nil {
		t.Fatalf("seed token: %v", err)
	}
	if err := settingService.SetTwoFactorEnable(true); err != nil {
		t.Fatalf("seed enable: %v", err)
	}

	post := func(t *testing.T, mutate func(map[string]any)) string {
		t.Helper()
		base, err := settingService.GetAllSetting()
		if err != nil {
			t.Fatalf("GetAllSetting: %v", err)
		}
		raw, err := json.Marshal(base)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		body := map[string]any{}
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		mutate(body)
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}

		gin.SetMode(gin.TestMode)
		router := gin.New()
		NewSettingController(router.Group("/panel/api"))
		req := httptest.NewRequest(http.MethodPost, "/panel/api/setting/update", strings.NewReader(string(payload)))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		return resp.Body.String()
	}

	t.Run("rebind without code is rejected", func(t *testing.T) {
		got := post(t, func(body map[string]any) {
			body["twoFactorEnable"] = true
			body["twoFactorToken"] = "ATTACKERSECRET567890"
		})
		if !strings.Contains(got, `"success":false`) {
			t.Fatalf("rebind without a 2FA code was accepted: %s", got)
		}
		stored, err := settingService.GetTwoFactorToken()
		if err != nil {
			t.Fatalf("GetTwoFactorToken: %v", err)
		}
		if stored != "ORIGINALSECRET234567" {
			t.Fatalf("stored 2FA secret = %q, want it unchanged", stored)
		}
	})

	t.Run("ordinary save with redacted token still succeeds", func(t *testing.T) {
		got := post(t, func(body map[string]any) {
			body["twoFactorEnable"] = true
			body["twoFactorToken"] = ""
		})
		if !strings.Contains(got, `"success":true`) {
			t.Fatalf("normal settings save was rejected: %s", got)
		}
		stored, err := settingService.GetTwoFactorToken()
		if err != nil {
			t.Fatalf("GetTwoFactorToken: %v", err)
		}
		if stored != "ORIGINALSECRET234567" {
			t.Fatalf("stored 2FA secret = %q, want it preserved", stored)
		}
	})
}

func TestTestDiscordEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 1. Service not initialized
	SetDiscordService(nil)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("I18n", func(_ locale.I18nType, key string, _ ...string) string { return key })
		c.Next()
	})
	NewSettingController(router.Group("/panel/api"))

	req := httptest.NewRequest(http.MethodPost, "/panel/api/setting/testDiscord", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if !strings.Contains(resp.Body.String(), `"success":false`) || !strings.Contains(resp.Body.String(), "pages.settings.discordNotInitialized") {
		t.Fatalf("expected uninitialized error, got %s", resp.Body.String())
	}

	// Setup DB
	t.Setenv("XUI_DB_FOLDER", t.TempDir())
	dbtest.InitDB(t, filepath.Join(t.TempDir(), "x-ui.db"))
	t.Cleanup(func() { SetDiscordService(nil) })

	settingService := service.SettingService{}
	svc := discord.NewDiscordService(settingService)
	SetDiscordService(svc)

	// 2. Discord bot disabled
	_ = settingService.SetDiscordBotEnable(false)
	req = httptest.NewRequest(http.MethodPost, "/panel/api/setting/testDiscord", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if !strings.Contains(resp.Body.String(), `"success":false`) || !strings.Contains(resp.Body.String(), "pages.settings.discordBotNotEnabled") {
		t.Fatalf("expected disabled error, got %s", resp.Body.String())
	}

	// 3. Discord bot enabled but missing config
	_ = settingService.SetDiscordBotEnable(true)
	req = httptest.NewRequest(http.MethodPost, "/panel/api/setting/testDiscord", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if !strings.Contains(resp.Body.String(), `"success":false`) || !strings.Contains(resp.Body.String(), "pages.settings.discordTestFailed") {
		t.Fatalf("expected send failure error, got %s", resp.Body.String())
	}

	// 4. Discord bot enabled with working server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "msg-1"}`))
	}))
	defer server.Close()

	_ = settingService.SetDiscordBotToken("test-bot-token")
	_ = settingService.SetDiscordChannelId("123456789")
	svc.SetBaseURL(server.URL)
	svc.SetHTTPClient(server.Client())

	req = httptest.NewRequest(http.MethodPost, "/panel/api/setting/testDiscord", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if !strings.Contains(resp.Body.String(), `"success":true`) || !strings.Contains(resp.Body.String(), "pages.settings.discordTestSuccess") {
		t.Fatalf("expected success, got %s", resp.Body.String())
	}
}

// The header greets the admin by name; it must be the signed-in user, never another row.
func TestCurrentUserReturnsSignedInUsername(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		session.SetAPIAuthUser(c, &model.User{Id: 7, Username: "parsa", Password: "hash"})
	})
	NewSettingController(router.Group("/panel/api"))

	req := httptest.NewRequest(http.MethodGet, "/panel/api/setting/user", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	var body struct {
		Success bool `json:"success"`
		Obj     struct {
			Username string `json:"username"`
			Password string `json:"password"`
		} `json:"obj"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %s: %v", resp.Body.String(), err)
	}
	if !body.Success || body.Obj.Username != "parsa" {
		t.Fatalf("got %s, want success with username parsa", resp.Body.String())
	}
	if body.Obj.Password != "" {
		t.Fatalf("password hash leaked: %s", resp.Body.String())
	}
}

// The wizard saves through /setting/profile, so a bad address must come back as a failure the page can show.
func TestUpdateServerProfileEndpoint(t *testing.T) {
	t.Setenv("XUI_DB_FOLDER", t.TempDir())
	dbtest.InitDB(t, filepath.Join(t.TempDir(), "x-ui.db"))
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewSettingController(router.Group("/panel/api"))

	post := func(body string) string {
		req := httptest.NewRequest(http.MethodPost, "/panel/api/setting/profile", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		return resp.Body.String()
	}

	if got := post(`{"serverAddress":"https://x.example/a"}`); !strings.Contains(got, `"success":false`) {
		t.Fatalf("bad address accepted: %s", got)
	}
	if got := post(`{"serverName":"Frankfurt 1","serverCountry":"DE","serverAddress":"203.0.113.7","setupDone":true}`); !strings.Contains(got, `"success":true`) {
		t.Fatalf("profile rejected: %s", got)
	}
	view, err := (&service.SettingService{}).GetAllSettingView()
	if err != nil {
		t.Fatal(err)
	}
	if view.ServerName != "Frankfurt 1" || view.ServerCountry != "DE" || view.ServerAddress != "203.0.113.7" || !view.SetupDone {
		t.Fatalf("saved profile = %q %q %q done=%v", view.ServerName, view.ServerCountry, view.ServerAddress, view.SetupDone)
	}
}
