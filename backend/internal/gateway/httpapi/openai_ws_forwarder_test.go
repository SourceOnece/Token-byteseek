package httpapi

import (
	"context"
	"net/http"
	"testing"
	"time"

	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/TokenFlux/TokenRouter/internal/upstream/openai"
	"github.com/stretchr/testify/require"
)

func TestOpenAIWSTerminalEvent_ResponseFailedRecordsModelTransient(t *testing.T) {
	svc := newWSFixture(wsFixtureInputs{})
	setWSFixtureHealth(svc, newUpstreamHealthForTest(transientCooldownProviderRepo{}, &wsFixtureOptions{}, nil, providercore.HealthOptions{}, nil))

	provider := &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, ID: 5201, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeAPIKey}}
	payload := []byte(`{"type":"response.failed","response":{"error":{"code":"server_error","message":"Internal error"}}}`)

	for range 2 {
		terminalPolicy := svc.handleOpenAIWSTerminalTransientFailure(context.Background(), provider, "gpt-5.5", http.Header{}, payload)
		require.Equal(t, "response.failed", terminalPolicy.TerminalEvent)
		require.Equal(t, http.StatusBadGateway, terminalPolicy.StatusCode)
	}

	require.True(t, wsFixtureModelBlocked(svc, provider, "gpt-5.5"))
}

// TestOpenAIWSTerminalFailureReturnsExplicitPolicyDecision 验证 response.failed
// 不只写提供商状态，还把显式策略结果返回给写客户端事件的调用方。
func TestOpenAIWSTerminalFailureReturnsExplicitPolicyDecision(t *testing.T) {
	svc := newWSFixture(wsFixtureInputs{})
	repo := &openAIWSPolicyRepo{}
	setWSFixtureHealth(svc, newUpstreamHealthForTest(repo, &wsFixtureOptions{}, nil, providercore.HealthOptions{}, nil))

	provider := &gatewayprovider.ExecutionProvider{
		Record: providercore.Record{
			LoadLocation: time.LoadLocation, ID: 5206,
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeAPIKey,
			Credentials: map[string]any{
				"custom_error_codes_enabled": true,
				"custom_error_codes":         []any{float64(http.StatusUnprocessableEntity)},
			},
		},
	}
	payload := []byte(`{"type":"response.failed","response":{"error":{"status_code":422,"message":"configured"}}}`)

	terminalPolicy := svc.handleOpenAIWSTerminalTransientFailure(
		context.Background(), provider, "gpt-5.5", http.Header{}, payload,
	)

	require.Equal(t, "response.failed", terminalPolicy.TerminalEvent)
	require.Equal(t, http.StatusUnprocessableEntity, terminalPolicy.StatusCode)
	require.Equal(t, providercore.ErrorPolicyCustomMatched, terminalPolicy.Decision.Policy)
	require.True(t, terminalPolicy.Decision.ShouldFailover(gatewayprovider.ExecutionErrorPolicy(provider), terminalPolicy.StatusCode, false))
	require.Equal(t, 1, repo.setErrorCalls)
}

// TestOpenAIWSTerminalContentPolicyBypassesProviderPolicy 验证内容安全拒绝
// 即使被语义映射为 502，也不会误命中提供商自定义错误码。
func TestOpenAIWSTerminalContentPolicyBypassesProviderPolicy(t *testing.T) {
	svc := newWSFixture(wsFixtureInputs{})
	repo := &openAIWSPolicyRepo{}
	setWSFixtureHealth(svc, newUpstreamHealthForTest(repo, &wsFixtureOptions{}, nil, providercore.HealthOptions{}, nil))

	provider := &gatewayprovider.ExecutionProvider{
		Record: providercore.Record{
			LoadLocation: time.LoadLocation, ID: 5207,
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeAPIKey,
			Credentials: map[string]any{
				"custom_error_codes_enabled": true,
				"custom_error_codes":         []any{float64(http.StatusBadGateway)},
			},
		},
	}
	payload := []byte(`{"type":"response.failed","response":{"error":{"code":"content_policy","message":"request blocked by policy"}}}`)

	terminalPolicy := svc.handleOpenAIWSTerminalTransientFailure(
		context.Background(), provider, "gpt-5.5", http.Header{}, payload,
	)

	require.Equal(t, http.StatusBadGateway, terminalPolicy.StatusCode)
	require.Equal(t, providercore.ErrorPolicyNone, terminalPolicy.Decision.Policy)
	require.False(t, terminalPolicy.Decision.ShouldFailover(
		gatewayprovider.ExecutionErrorPolicy(provider), terminalPolicy.StatusCode, openai.OpenAIStreamFailedEventShouldFailover(payload, "request blocked by policy"),
	))
	require.Zero(t, repo.setErrorCalls)
}

func TestOpenAIWSErrorEvent_ServerErrorRecordsModelTransient(t *testing.T) {
	svc := newWSFixture(wsFixtureInputs{})
	setWSFixtureHealth(svc, newUpstreamHealthForTest(transientCooldownProviderRepo{}, &wsFixtureOptions{}, nil, providercore.HealthOptions{}, nil))

	provider := &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, ID: 5203, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeAPIKey}}
	payload := []byte(`{"type":"error","error":{"code":"server_error","type":"server_error","message":"Internal error"}}`)

	for range 2 {
		svc.handleOpenAIWSErrorEventTransientFailure(context.Background(), provider, "gpt-5.5", http.Header{}, payload)
	}

	require.True(t, wsFixtureModelBlocked(svc, provider, "gpt-5.5"))
}

func TestOpenAIWSPayloadTransientStatus_Explicit529IsNotModelTransient(t *testing.T) {
	payload := []byte(`{"type":"response.failed","response":{"error":{"status_code":529,"code":"server_error","message":"overloaded"}}}`)

	require.Zero(t, openAIWSPayloadTransientStatus(payload))
}

func TestOpenAIWSErrorPolicyStatus_PreservesExplicitStatusAndFallbackMapping(t *testing.T) {
	require.Equal(t, 529, openAIWSErrorPolicyStatus([]byte(`{"type":"response.failed","response":{"error":{"status_code":529,"code":"server_error"}}}`)))
	require.Equal(t, http.StatusBadGateway, openAIWSErrorPolicyStatus([]byte(`{"type":"error","error":{"type":"server_error"}}`)))
}

func TestOpenAIWSDial5xxRecordsModelTransient(t *testing.T) {
	svc := newWSFixture(wsFixtureInputs{})
	setWSFixtureHealth(svc, newUpstreamHealthForTest(transientCooldownProviderRepo{}, &wsFixtureOptions{}, nil, providercore.HealthOptions{}, nil))

	provider := &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, ID: 5202, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeAPIKey}}
	dialErr := &openai.WSDialError{
		StatusCode:      http.StatusBadGateway,
		ResponseHeaders: http.Header{"X-Request-Id": []string{"req-ws-502"}},
		ResponseBody:    []byte(`{"error":{"message":"bad gateway"}}`),
	}

	for range 2 {
		svc.handleOpenAIWSDialTransientFailure(context.Background(), provider, "gpt-5.5", dialErr)
	}

	require.Eventually(t, func() bool {
		return wsFixtureModelBlocked(svc, provider, "gpt-5.5")
	}, time.Second, 10*time.Millisecond)
}

// TestOpenAIWSPoolModeErrorUsesConfiguredRetry 验证原生 WebSocket 错误也使用
// 池模式统一决策，不再写入默认模型瞬态冷却。
func TestOpenAIWSPoolModeErrorUsesConfiguredRetry(t *testing.T) {
	svc := newWSFixture(wsFixtureInputs{})
	setWSFixtureHealth(svc, newUpstreamHealthForTest(transientCooldownProviderRepo{}, &wsFixtureOptions{}, nil, providercore.HealthOptions{}, nil))

	provider := &gatewayprovider.ExecutionProvider{
		Record: providercore.Record{
			LoadLocation: time.LoadLocation, ID: 5204,
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeAPIKey,
			Credentials: map[string]any{
				"pool_mode":                    true,
				"pool_mode_retry_status_codes": []any{float64(http.StatusBadGateway)},
			},
		},
	}

	decision := svc.applyOpenAIWSEventErrorPolicy(
		context.Background(), provider, "gpt-5.5", http.StatusBadGateway, http.Header{}, []byte(`{"error":{"message":"bad gateway"}}`),
	)

	require.Equal(t, providercore.ErrorPolicyPoolBypassed, decision.Policy)
	require.True(t, decision.RetryableOnSameProvider(gatewayprovider.ExecutionErrorPolicy(provider), http.StatusBadGateway))
	require.False(t, wsFixtureModelBlocked(svc, provider, "gpt-5.5"))
}

// openAIWSPolicyRepo 记录 WebSocket 显式策略触发的提供商错误写入。
type openAIWSPolicyRepo struct {
	transientCooldownProviderRepo
	setErrorCalls int
}

func (r *openAIWSPolicyRepo) SetError(context.Context, int64, string) error {
	r.setErrorCalls++
	return nil
}

// TestOpenAIWSCustomNonFailoverStatusStopsScheduling 验证 WebSocket 派生出的
// 非默认故障转移状态也执行管理员显式策略，并禁止同提供商重试。
func TestOpenAIWSCustomNonFailoverStatusStopsScheduling(t *testing.T) {
	svc := newWSFixture(wsFixtureInputs{})
	repo := &openAIWSPolicyRepo{}
	setWSFixtureHealth(svc, newUpstreamHealthForTest(repo, &wsFixtureOptions{}, nil, providercore.HealthOptions{}, nil))

	provider := &gatewayprovider.ExecutionProvider{
		Record: providercore.Record{
			LoadLocation: time.LoadLocation, ID: 5205,
			Platform: capability.PlatformOpenAI,
			Type:     capability.ProviderTypeAPIKey,
			Credentials: map[string]any{
				"pool_mode":                  true,
				"custom_error_codes_enabled": true,
				"custom_error_codes":         []any{float64(http.StatusUnprocessableEntity)},
			},
		},
	}

	decision := svc.applyOpenAIWSEventErrorPolicy(
		context.Background(), provider, "gpt-5.5", http.StatusUnprocessableEntity, http.Header{}, []byte(`{"error":{"message":"configured"}}`),
	)

	require.Equal(t, providercore.ErrorPolicyCustomMatched, decision.Policy)
	require.True(t, decision.StopScheduling)
	require.False(t, decision.RetryableOnSameProvider(gatewayprovider.ExecutionErrorPolicy(provider), http.StatusUnprocessableEntity))
	require.Equal(t, 1, repo.setErrorCalls)
}
