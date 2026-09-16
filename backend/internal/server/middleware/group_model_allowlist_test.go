package middleware

import (
	"bytes"
	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// 默认关闭时不读取/改写正文；开启后所有协议都在模型映射前拒绝未命中项。
func TestGroupModelAllowlistAdmission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, path := range []string{"/v1/responses", "/v1/messages", "/v1/chat/completions", "/images/edits"} {
		for _, tc := range []struct {
			name, body string
			enabled    bool
			status     int
		}{
			{"disabled", `{"model":"blocked"}`, false, 200},
			{"allowed", `{"model":"public"}`, true, 200},
			{"denied", `{"model":"blocked"}`, true, 404},
			{"duplicate", `{"model":"public","model":"blocked"}`, true, 404},
			{"case variant", `{"Model":"blocked"}`, true, 404},
		} {
			t.Run(path+tc.name, func(t *testing.T) {
				key := &service.APIKey{Group: &service.Group{ModelsListConfig: service.GroupModelsListConfig{Enabled: true, Models: []string{"display-only"}}, ModelAllowlist: service.GroupModelAllowlist{Enabled: tc.enabled, Models: []string{"public"}}}}
				r := gin.New()
				r.POST(path, func(c *gin.Context) {
					checkGroupModelAllowlist(c, key)
					if c.IsAborted() {
						return
					}
					body, err := io.ReadAll(c.Request.Body)
					require.NoError(t, err)
					require.Equal(t, tc.body, string(body))
					c.Status(http.StatusOK)
				})
				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(tc.body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Upgrade", "websocket")
				r.ServeHTTP(w, req)
				require.Equal(t, tc.status, w.Code, w.Body.String())
			})
		}
	}
}

func TestGroupModelAllowlistGeminiPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	key := &service.APIKey{Group: &service.Group{ModelAllowlist: service.GroupModelAllowlist{Enabled: true, Models: []string{"gemini-allowed"}}}}
	r := gin.New()
	r.POST("/v1beta/models/*modelAction", func(c *gin.Context) {
		checkGroupModelAllowlist(c, key)
		if !c.IsAborted() {
			c.Status(200)
		}
	})
	for _, tc := range []struct {
		model  string
		status int
	}{{"gemini-allowed", 200}, {"gemini-blocked", 404}} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("POST", "/v1beta/models/"+tc.model+":generateContent", bytes.NewBufferString(`{"contents":[]}`)))
		require.Equal(t, tc.status, w.Code)
	}
}
