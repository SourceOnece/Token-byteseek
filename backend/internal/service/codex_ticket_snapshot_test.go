//go:build unit

package service

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 沿快照门控账号/最终模型，不能把缺票变成账号总停调，compact 不可误用客户端模型。
func TestCodexTicketSnapshotModelGateAndCompact(t *testing.T) {
	s, cache, _ := newTicketTestService()
	g := &OpenAIGatewayService{cfg: &config.Config{Gateway: config.GatewayConfig{OpenAICompactModel: "gpt-5.5"}}}
	g.codexTickets.Store(s)
	a := ticketAccount()
	ctx := context.Background()
	require.False(t, g.isOpenAIAccountRequestBlocked(ctx, &a, "gpt-6-astra", false))
	require.Zero(t, cache.gets)
	enableTicketTest(t, s)
	require.True(t, g.isOpenAIAccountRequestBlocked(ctx, &a, "gpt-6-astra", false))
	require.False(t, g.isOpenAIAccountRequestBlocked(ctx, &a, "gpt-6-astra", true))
	a.Credentials["model_mapping"] = map[string]any{"alias": "gpt-6-astra"}
	require.True(t, g.isOpenAIAccountRequestBlocked(ctx, &a, "alias", false))
	seedTicket(t, s, &a, "gpt-6-astra", "fake-token")
	require.False(t, g.isOpenAIAccountRequestBlocked(ctx, &a, "alias", false))
	require.True(t, g.isOpenAIAccountRequestBlocked(ctx, &a, "gpt-5.6-sol", false))
	other := a
	other.ID++
	require.True(t, g.isOpenAIAccountRequestBlocked(ctx, &other, "gpt-6-astra", false))
	other.Type = AccountTypeAPIKey
	require.False(t, g.isOpenAIAccountRequestBlocked(ctx, &other, "gpt-6-astra", false))
	require.True(t, a.Schedulable)
	require.Empty(t, a.Extra)
	require.Empty(t, cache.claims)
}

// 三种出站入口必须对缺票报错；关闭后保留调用前行为，无现场采集。
func TestCodexTicketSnapshotBuildersRejectMissingTicket(t *testing.T) {
	s, cache, _ := newTicketTestService()
	enableTicketTest(t, s)
	g := &OpenAIGatewayService{}
	g.codexTickets.Store(s)
	a := ticketAccount()
	c, _ := gin.CreateTestContext(nil)
	c.Request, _ = http.NewRequest(http.MethodPost, "/v1/responses", nil)
	body := []byte(`{"model":"gpt-6-astra","input":"test"}`)
	_, err := g.buildUpstreamRequest(context.Background(), c, &a, body, "fake-token", true, "", true)
	require.ErrorIs(t, err, ErrCodexTicketUnavailable)
	_, err = g.buildUpstreamRequestOpenAIPassthrough(context.Background(), c, &a, body, "fake-token")
	require.ErrorIs(t, err, ErrCodexTicketUnavailable)
	_, _, err = g.buildOpenAIWSHeaders(context.Background(), c, &a, "fake-token", OpenAIWSProtocolDecision{Transport: OpenAIUpstreamTransportResponsesWebsocketV2}, true, "", "", "", "gpt-6-astra", "")
	require.ErrorIs(t, err, ErrCodexTicketUnavailable)
	require.Empty(t, cache.claims)
	no := false
	_, err = s.Update(context.Background(), CodexTicketSettingsUpdate{Enabled: &no})
	require.NoError(t, err)
	_, err = g.buildUpstreamRequest(context.Background(), c, &a, body, "fake-token", true, "", true)
	require.NoError(t, err)
}

// 有效且不临期时不采集；临期拿到 312 不销毁旧票，到期后门控，再次获票恢复。
func TestCodexTicketSnapshotRefreshUsesTTLOnly(t *testing.T) {
	s, cache, a := setupTicketProxyTest(t, "fixed", 1)
	ticket := seedTicket(t, s, a, "gpt-6-astra", "fake-token")
	up := &ticketSequenceUpstream{responses: []*http.Response{ticketResponseForProxy(200, 312, "")}}
	s.gateway.httpUpstream = up
	ctx := context.Background()
	s.probe(ctx, s.config.Load(), a, "gpt-6-astra")
	require.Empty(t, up.proxies)
	key := codexTicketKey(s.config.Load(), a, "gpt-6-astra", "fake-token")
	setExpiry := func(exp time.Time) {
		raw, _ := json.Marshal(codexTicketValue{State: ticket, ExpiresAt: exp})
		cipher, err := s.cipher.Encrypt(string(raw))
		require.NoError(t, err)
		require.NoError(t, cache.Set(ctx, key, cipher, time.Hour))
	}
	setExpiry(time.Now().Add(9 * time.Minute))
	s.probe(ctx, s.config.Load(), a, "gpt-6-astra")
	require.Len(t, up.proxies, 1)
	require.False(t, s.Blocks(ctx, a, "gpt-6-astra"))
	setExpiry(time.Now().Add(-time.Second))
	require.True(t, s.Blocks(ctx, a, "gpt-6-astra"))
	cache.claims = map[string]bool{}
	up.responses = []*http.Response{ticketResponseForProxy(200, 292, "")}
	s.probe(ctx, s.config.Load(), a, "gpt-6-astra")
	require.False(t, s.Blocks(ctx, a, "gpt-6-astra"))
}
