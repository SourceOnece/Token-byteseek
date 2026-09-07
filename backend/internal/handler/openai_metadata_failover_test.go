package handler

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/TokenFlux/TokenRouter/internal/pkg/tlsfingerprint"
	"github.com/TokenFlux/TokenRouter/internal/server/middleware"
	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// 合成账号全部返回额度耗尽，验证三类 handler 均真正尝试下一个账号；不产生成功计费。
type metadataQuotaFailoverUpstream struct {
	service.HTTPUpstream
	ids    []int64
	bodies [][]byte
}

func (u *metadataQuotaFailoverUpstream) Do(r *http.Request, _ string, id int64, _ int) (*http.Response, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	u.ids = append(u.ids, id)
	u.bodies = append(u.bodies, body)
	return &http.Response{StatusCode: 429, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"error":{"type":"rate_limit_error","code":"rate_limit_exceeded","message":"quota exhausted"}}`))}, nil
}
func (u *metadataQuotaFailoverUpstream) DoWithTLS(r *http.Request, p string, id int64, n int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(r, p, id, n)
}

// 开→关、关→开、开→开、关→关都不能阻止原来的额度切号或复用上一账号的修复快照。
func TestOpenAIMetadataRepairQuotaFailoverThreeProtocols(t *testing.T) {
	for _, route := range []string{"responses", "chat", "messages"} {
		for _, flags := range [][2]bool{{false, false}, {true, false}, {false, true}, {true, true}} {
			t.Run(fmt.Sprintf("%s/%v", route, flags), func(t *testing.T) {
				gin.SetMode(gin.TestMode)
				accounts := make([]service.Account, 2)
				for i := range accounts {
					accounts[i] = service.Account{ID: int64(9900 + i), Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Priority: i + 1,
						Credentials: map[string]any{"access_token": "synthetic", "chatgpt_account_id": fmt.Sprint("account-", i)},
						Extra:       map[string]any{"codex_metadata_repair_enabled": flags[i], "codex_fingerprint_mode": "full", "codex_fingerprint_seed": "11111111-1111-4111-8111-111111111111"}}
				}
				cfg := &config.Config{RunMode: config.RunModeSimple}
				cfg.Default.RateMultiplier = 1
				cfg.Gateway.MaxAccountSwitches = 1
				repo := &openAIWSFailoverHandlerAccountRepoStub{accounts: accounts}
				up := &metadataQuotaFailoverUpstream{}
				billing := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
				t.Cleanup(billing.Stop)
				svc := service.NewOpenAIGatewayService(repo, nil, nil, nil, nil, nil, nil, cfg, nil, nil, service.NewBillingService(cfg, nil), nil, billing, up, nil, &service.DeferredService{}, nil, nil, nil, nil, nil, nil, nil)
				h := NewOpenAIGatewayHandler(svc, service.NewConcurrencyService(nil), billing, service.NewAPIKeyService(nil, nil, nil, nil, nil, nil, cfg), nil, nil, nil, nil, cfg)
				rec := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(rec)
				body := `{"model":"gpt-6-astra","input":"keep","messages":[{"role":"user","content":"keep"}],"max_tokens":64,"stream":false,"client_metadata":{"session_id":"same","thread_id":"same","turn_id":"turn","x-codex-turn-metadata":"{\"turn_id\":\"turn\",\"root_turn_id\":\"turn\"}"}}`
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/"+route, strings.NewReader(body))
				c.Request.Header.Set("Content-Type", "application/json")
				id := int64(1)
				c.Set(string(middleware.ContextKeyAPIKey), &service.APIKey{ID: 81, GroupID: &id, User: &service.User{ID: 91, Status: service.StatusActive}, Group: &service.Group{ID: id, Platform: service.PlatformOpenAI, Status: service.StatusActive, AllowedClientProtocols: []service.GroupClientProtocol{service.GroupClientProtocolAnthropicMessages, service.GroupClientProtocolOpenAIResponses, service.GroupClientProtocolOpenAIChatCompletions}}})
				c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 91})
				switch route {
				case "responses":
					h.Responses(c)
				case "chat":
					h.ChatCompletions(c)
				case "messages":
					h.Messages(c)
				}
				require.Equal(t, []int64{9900, 9901}, up.ids, "response=%s", rec.Body.String())
				for i, wire := range up.bodies {
					meta := gjson.GetBytes(wire, "client_metadata.x-codex-turn-metadata").String()
					if flags[i] {
						require.Equal(t, gjson.Get(meta, "turn_id").String(), gjson.Get(meta, "root_turn_id").String())
						require.NotEmpty(t, gjson.Get(meta, "root_turn_id").String())
					}
				}
				require.GreaterOrEqual(t, rec.Code, 400)
			})
		}
	}
}
