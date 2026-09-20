//go:build unit

package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestCodexTicketLegacyAgentIdentityMetadataExemption(t *testing.T) {
	for _, mode := range []string{"basic", "basic_batch", "advanced"} {
		t.Run(mode, func(t *testing.T) {
			s, _, _ := newTicketTestService()
			enableTicketTest(t, s)
			full := ticketAccount()
			full.Credentials["auth_mode"] = OpenAIAuthModeAgentIdentity
			full.GroupIDs = []int64{1}
			meta := ticketSchedulerMetadata(full)
			repo := &ticketAccountStub{accounts: []Account{full}}
			cache := &openAISnapshotCacheStub{snapshotAccounts: []*Account{&meta}, accountsByID: map[int64]*Account{1: &full}}
			g := &OpenAIGatewayService{accountRepo: repo, cfg: &config.Config{}, cache: &schedulerTestGatewayCache{}, concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{})}
			g.cfg.Gateway.Scheduling.LoadBatchEnabled = mode != "basic"
			g.schedulerSnapshot = NewSchedulerSnapshotService(cache, nil, repo, nil, g.cfg)
			g.codexTickets.Store(s)
			s.gateway = g
			ctx := context.Background()
			require.False(t, s.Blocks(ctx, &full, "gpt-6-astra"))
			require.False(t, s.Blocks(ctx, &meta, "gpt-6-astra"), "旧摘要缺auth_mode不能让豁免账号被拦")
			group := int64(1)
			if mode == "advanced" {
				ctx = withAdvancedSchedulerTestGroup(ctx, group)
			}
			result, _, err := g.SelectAccountWithScheduler(ctx, &group, "", "", "gpt-6-astra", nil, OpenAIUpstreamTransportAny, false)
			require.NoError(t, err)
			require.NotNil(t, result)
			if result.ReleaseFunc != nil {
				result.ReleaseFunc()
			}
			require.NotContains(t, meta.Credentials, "auth_mode", "局部补全不改共享摘要")
		})
	}
}

// 慢上游直到轮次预算结束才返回，用来复现后半段模型长期得不到worker的问题。
type ticketFairnessUpstream struct {
	HTTPUpstream
	mu     sync.Mutex
	models map[string]int
}

func (u *ticketFairnessUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	body, _ := io.ReadAll(req.Body)
	u.mu.Lock()
	u.models[gjson.GetBytes(body, "model").String()]++
	u.mu.Unlock()
	<-req.Context().Done()
	return nil, req.Context().Err()
}

func TestCodexTicketHarvestContinuesAfterLastModel(t *testing.T) {
	s, _, _ := setupTicketManualTest(t, 332)
	models := []string{"model-a", "model-b", "model-c", "model-d", "model-e", "model-f"}
	err := configureTicketTestAccount(t, s, CodexTicketAccountPatch{Rules: &CodexTicketRulesPatch{Models: &models}})
	require.NoError(t, err)
	u := &ticketFairnessUpstream{models: map[string]int{}}
	s.gateway.httpUpstream = u
	for i := 0; i < 5; i++ {
		// 测试缓存不自行过期；清理claim模拟下一真实轮次租约已到期。
		s.cache.(*ticketCacheStub).claims = map[string]bool{}
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		s.harvest(ctx)
		cancel()
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	for _, model := range models {
		require.Positive(t, u.models[model], model)
	}
}

func TestCodexTicketRotationUsesBoundedLocalFailover(t *testing.T) {
	s, repo, _ := setupTicketManualTest(t, 332)
	ctx := context.Background()
	batch, err := s.PrepareManualCollection(ctx, manualRequest(s))
	require.NoError(t, err)
	batch.Execute(func(string, any) bool { return true })
	a := repo.accounts[0]
	require.False(t, s.Blocks(ctx, &a, "gpt-6-astra"))
	g := s.gateway
	g.codexTickets.Store(s)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/responses", nil)
	body := []byte(`{"model":"gpt-6-astra","input":"test"}`)
	for _, transport := range []string{"http", "passthrough", "ws"} {
		t.Run(transport, func(t *testing.T) {
			var buildErr error
			switch transport {
			case "http":
				_, buildErr = g.buildUpstreamRequest(ctx, c, &a, body, "rotated-synthetic-token", true, "", true)
			case "passthrough":
				_, buildErr = g.buildUpstreamRequestOpenAIPassthrough(ctx, c, &a, body, "rotated-synthetic-token")
			case "ws":
				_, _, buildErr = g.buildOpenAIWSHeaders(ctx, c, &a, "rotated-synthetic-token", OpenAIWSProtocolDecision{}, true, "", "", "", "gpt-6-astra", "")
			}
			require.ErrorIs(t, buildErr, ErrCodexTicketUnavailable)
			var failover *UpstreamFailoverError
			require.True(t, errors.As(buildErr, &failover))
			require.Equal(t, CodexTicketUnavailableReason, failover.Reason)
			require.True(t, failover.ShouldRetryNextAccount())
			require.False(t, failover.ShouldReportAccountScheduleFailure())
			require.False(t, failover.RetryableOnSameAccount)
			require.False(t, failover.SafeToFailoverAfterWrite)
			require.False(t, failover.IsCredentialFailure())
		})
	}
	// 关闭后出站构建完全沿用原路径，不因旧票据状态继续拦截。
	no := false
	_, err = s.Update(ctx, CodexTicketSettingsUpdate{Enabled: &no})
	require.NoError(t, err)
	_, err = g.buildUpstreamRequest(ctx, c, &a, body, "rotated-synthetic-token", true, "", true)
	require.NoError(t, err)
}

func TestCodexTicketCustomAstraProbeVersionFloor(t *testing.T) {
	for _, model := range []string{"gpt-6-astra-custom", "custom-ASTRA", "gpt-6-other", "gpt-5.6-sol"} {
		t.Run(model, func(t *testing.T) {
			s, _, a := setupTicketProxyTest(t, "fixed", 1)
			models := []string{model}
			err := configureTicketTestAccount(t, s, CodexTicketAccountPatch{Rules: &CodexTicketRulesPatch{Models: &models}})
			require.NoError(t, err)
			u := &ticketSequenceUpstream{responses: []*http.Response{ticketResponseForProxy(200, 292, "")}}
			s.gateway.httpUpstream = u
			s.probe(context.Background(), s.config.Load(), a, model)
			require.Len(t, u.headers, 1)
			minimum := "0.153.4"
			if model == "gpt-5.6-sol" {
				minimum = "0.146.0"
			}
			require.GreaterOrEqual(t, CompareVersions(u.headers[0].Get("version"), minimum), 0)
		})
	}
}

func TestCodexTicketForwardMissingTicketDoesNotWriteOrSend(t *testing.T) {
	for _, mode := range []string{"http", "passthrough", "ws"} {
		t.Run(mode, func(t *testing.T) {
			s, _, _ := newTicketTestService()
			enableTicketTest(t, s)
			cfg := &config.Config{}
			cfg.Gateway.OpenAIWS.Enabled = mode == "ws"
			cfg.Gateway.OpenAIWS.OAuthEnabled = mode == "ws"
			cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = mode == "ws"
			upstream := &httpUpstreamSequenceRecorder{}
			g := &OpenAIGatewayService{cfg: cfg, httpUpstream: upstream, openaiWSResolver: NewOpenAIWSProtocolResolver(cfg)}
			g.codexTickets.Store(s)
			a := ticketAccount()
			a.Extra = map[string]any{"openai_oauth_passthrough": mode == "passthrough", "responses_websockets_v2_enabled": mode == "ws"}
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/responses", nil)
			c.Request.Header.Set("User-Agent", "codex_cli_rs/0.153.4")
			result, err := g.Forward(context.Background(), c, &a, []byte(`{"model":"gpt-6-astra","input":[{"role":"user","content":"test"}],"stream":true}`))
			require.Nil(t, result)
			require.ErrorIs(t, err, ErrCodexTicketUnavailable)
			var failover *UpstreamFailoverError
			require.ErrorAs(t, err, &failover)
			require.False(t, c.Writer.Written(), "必须把换号决定留给handler，不能先写通用502")
			require.Zero(t, upstream.callCount, "缺票时不允许发业务请求")
		})
	}
}
