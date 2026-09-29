package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 两个别名都保存审计动作但不能复制凭据/题目；下游仍收到完整原始请求。
func TestProviderAliasAuditPrivacy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, prefix := range []string{"accounts", "providers"} {
		for _, tc := range []struct{ method, suffix, body, omitted string }{
			{"GET", "/data", "", ""},
			{"POST", "/codex-quality-test", `{"prompt":"private-canary"}`, "<quality-test prompt omitted>"},
			{"POST", "/codex-quality-schedules", `{"config":{"prompt":"private-canary"}}`, "<quality-test prompt omitted>"},
			{"PUT", "/codex-quality-schedules/:id", `{"config":{"prompt":"private-canary"}}`, "<quality-test prompt omitted>"},
			{"POST", "/import/codex-session", `{"value":"private-canary"}`, "<credential-bearing body omitted>"},
			{"PUT", "/:id/ollama-cloud-usage/session", `{"session":"private-canary"}`, "<credential-bearing body omitted>"},
		} {
			t.Run(prefix+tc.method+tc.suffix, func(t *testing.T) {
				repo := &auditCaptureRepository{}
				svc := service.NewAuditLogService(repo, nil)
				svc.Start()
				t.Cleanup(svc.Stop)
				router := gin.New()
				router.Use(gin.HandlerFunc(NewAuditLogMiddleware(svc)))
				path := "/api/v1/admin/" + prefix + tc.suffix
				router.Handle(tc.method, path, func(c *gin.Context) {
					body, err := io.ReadAll(c.Request.Body)
					require.NoError(t, err)
					require.Equal(t, tc.body, string(body))
					c.Status(http.StatusNoContent)
				})
				request := httptest.NewRequest(tc.method, strings.ReplaceAll(path, ":id", "7"), strings.NewReader(tc.body))
				request.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()
				router.ServeHTTP(w, request)
				svc.Stop()
				require.Equal(t, http.StatusNoContent, w.Code)
				repo.mu.Lock()
				defer repo.mu.Unlock()
				require.Len(t, repo.logs, 1)
				require.Equal(t, path, repo.logs[0].Path)
				require.Equal(t, tc.omitted, repo.logs[0].RequestBody)
				require.NotContains(t, repo.logs[0].RequestBody, "private-canary")
				if tc.suffix == "/data" {
					require.Equal(t, "admin.accounts.export", repo.logs[0].Action)
				}
			})
		}
	}
}
