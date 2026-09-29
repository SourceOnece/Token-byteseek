package httpapi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	upstreamcore "github.com/TokenFlux/TokenRouter/internal/upstream"

	"github.com/TokenFlux/TokenRouter/internal/billing"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/TokenFlux/TokenRouter/internal/upstream/openai"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type openAIStream403ProviderRepo struct {
	gatewayprovider.ExecutionProviderStore

	setErrorCalls int
}

func (r *openAIStream403ProviderRepo) SetError(context.Context, int64, string) error {
	r.setErrorCalls++
	return nil
}

type openAIAuthPolicyProviderRepo struct {
	gatewayprovider.ExecutionProviderStore

	tempCalls     int
	setErrorCalls int
}

func (r *openAIAuthPolicyProviderRepo) SetTempUnschedulable(context.Context, int64, time.Time, string) error {
	r.tempCalls++
	return nil
}

func (r *openAIAuthPolicyProviderRepo) SetError(context.Context, int64, string) error {
	r.setErrorCalls++
	return nil
}

type openAIAuthPolicy403Counter struct {
	counts []int64
}

func (s *openAIAuthPolicy403Counter) IncrementOpenAI403Count(context.Context, int64, int) (int64, error) {
	if len(s.counts) == 0 {
		return 1, nil
	}
	count := s.counts[0]
	s.counts = s.counts[1:]
	return count, nil
}

func (*openAIAuthPolicy403Counter) ResetOpenAI403Count(context.Context, int64) error {
	return nil
}

func TestOpenAIHTTPAccessStateBadRequestDoesNotDisableProvider(t *testing.T) {
	repo := &openAIStream403ProviderRepo{}
	svc := newWSFixture(wsFixtureInputs{health: newUpstreamHealthForTest(repo, nil, nil, providercore.HealthOptions{}, nil)})
	provider := &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, ID: 925, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeOAuth, Status: billing.StatusActive, Schedulable: true}}
	body := []byte(`{"error":{"code":"unknown_parameter","message":"Unknown parameter: account disabled"}}`)

	disabled := gatewayprovider.ApplyOpenAIResponseHealth(context.Background(), svc.Output.Health, provider, http.StatusBadRequest, nil, body, false).StopScheduling

	require.False(t, disabled)
	require.Zero(t, repo.setErrorCalls)
	require.False(t, wsFixtureProviderBlocked(svc, provider))
}

func TestOpenAIStreamEchoedAccessStateMessageDoesNotDisableOrFailover(t *testing.T) {
	repo := &openAIStream403ProviderRepo{}
	svc := newWSFixture(wsFixtureInputs{health: newUpstreamHealthForTest(repo, nil, nil, providercore.HealthOptions{}, nil)})
	provider := &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, ID: 926, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeOAuth, Status: billing.StatusActive, Schedulable: true}}
	payload := []byte(`{"type":"response.failed","response":{"error":{"type":"invalid_request_error","code":"unknown_parameter","message":"Unknown parameter: account disabled"}}}`)
	message := openai.ExtractOpenAISSEErrorMessage(payload)

	require.False(t, gatewayprovider.IsOpenAIUpstreamAccessStateError(message, payload))
	require.False(t, openai.OpenAIStreamFailedEventShouldFailover(payload, message))
	status, disabled := svc.Output.TerminalProviderEffects(nil, provider, payload, message, nil)
	require.Equal(t, http.StatusBadGateway, status)
	require.False(t, disabled)
	require.Zero(t, repo.setErrorCalls)
	require.False(t, wsFixtureProviderBlocked(svc, provider))
}

func TestOpenAIHTTPAccessStateTrustsStructuredCode(t *testing.T) {
	repo := &openAIStream403ProviderRepo{}
	svc := newWSFixture(wsFixtureInputs{health: newUpstreamHealthForTest(repo, nil, nil, providercore.HealthOptions{}, nil)})
	provider := &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, ID: 930, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeOAuth, Status: billing.StatusActive, Schedulable: true}}
	body := []byte(`{"error":{"code":"organization_deactivated","message":"request rejected"}}`)

	require.True(t, gatewayprovider.IsOpenAIHTTPUpstreamAccessStateError(http.StatusBadRequest, "", body))
	require.True(t, gatewayprovider.ShouldFailoverOpenAIResponse(http.StatusBadRequest, "", body))
	require.True(t, gatewayprovider.ApplyOpenAIResponseHealth(context.Background(), svc.Output.Health, provider, http.StatusBadRequest, nil, body, false).StopScheduling)
	require.Equal(t, 1, repo.setErrorCalls)
	require.True(t, wsFixtureProviderBlocked(svc, provider))
}

func TestOpenAIHTTPAuthMessagesUseExistingStatusPolicies(t *testing.T) {
	t.Run("oauth 401 remains recoverable", func(t *testing.T) {
		repo := &openAIAuthPolicyProviderRepo{}
		rateLimits := newUpstreamHealthForTest(repo, &wsFixtureOptions{}, nil, providercore.HealthOptions{}, nil)

		svc := newWSFixture(wsFixtureInputs{health: rateLimits})
		provider := &gatewayprovider.ExecutionProvider{Record: providercore.Record{
			LoadLocation: time.LoadLocation, ID: 931, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeOAuth, Status: billing.StatusActive, Schedulable: true,
			Credentials: map[string]any{"refresh_token": "refreshable"},
		}}
		body := []byte(`{"error":{"message":"provider is disabled"}}`)

		require.False(t, gatewayprovider.IsOpenAIHTTPUpstreamAccessStateError(http.StatusUnauthorized, "", body))
		require.True(t, gatewayprovider.ApplyOpenAIResponseHealth(context.Background(), svc.Output.Health, provider, http.StatusUnauthorized, nil, body, false).StopScheduling)
		require.Zero(t, repo.setErrorCalls)
		require.Equal(t, 1, repo.tempCalls)
	})

	t.Run("403 uses counter cooldown", func(t *testing.T) {
		repo := &openAIAuthPolicyProviderRepo{}
		counter := &openAIAuthPolicy403Counter{counts: []int64{1}}
		var svc *wsExecutionFixture

		rateLimits := newUpstreamHealthForTest(repo, &wsFixtureOptions{}, nil, providercore.HealthOptions{ForbiddenCounter: counter, Block: func(v *providercore.Record, until time.Time, reason string) {
			wsFixtureBlockProvider(svc, gatewayprovider.NewExecutionProvider(v), until, reason)
		}}, nil)

		svc = newWSFixture(wsFixtureInputs{health: rateLimits})
		rateLimits.Limits.RetryOpenAI = func(v *providercore.Record, h http.Header, body []byte) bool {
			return wsFixtureRetry429(svc, gatewayprovider.NewExecutionProvider(v), h, body)
		}

		provider := &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, ID: 932, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeOAuth, Status: billing.StatusActive, Schedulable: true}}
		body := []byte(`{"error":{"message":"workspace has been suspended"}}`)

		require.False(t, gatewayprovider.IsOpenAIHTTPUpstreamAccessStateError(http.StatusForbidden, "", body))
		require.True(t, gatewayprovider.ApplyOpenAIResponseHealth(context.Background(), svc.Output.Health, provider, http.StatusForbidden, nil, body, false).StopScheduling)
		require.Zero(t, repo.setErrorCalls)
		require.Equal(t, 1, repo.tempCalls)
	})
}

func TestOpenAICapacityFailoverCarriesSafeTerminalResponse(t *testing.T) {
	message := "Our servers are currently overloaded. Please try again later."
	body := []byte(`{"error":{"code":"server_is_overloaded","message":"` + message + `"}}`)
	err := gatewayprovider.NewOpenAIUpstreamFailure(http.StatusBadRequest, nil, body, message, false)

	require.True(t, gatewayprovider.IsOpenAICapacityShed(err))
	require.Equal(t, http.StatusServiceUnavailable, err.ClientStatusCode)
	require.Equal(t, message, err.ClientMessage)
	require.NotContains(t, err.ClientMessage, "server_is_overloaded")
}

func TestOpenAIStreamSemanticStatusesPreservedAcrossTerminalShapes(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		status       int
		wantFailover bool
	}{
		{"unauthorized", `{"type":"error","error":{"type":"authentication_error","code":"invalid_api_key","message":"unauthorized"}}`, http.StatusUnauthorized, true},
		{"forbidden", `{"type":"response.failed","response":{"error":{"type":"permission_error","message":"forbidden"}}}`, http.StatusForbidden, false},
		{"rate_limit", `{"error":{"type":"rate_limit_error","code":"rate_limit_exceeded","message":"slow down"}}`, http.StatusTooManyRequests, true},
		{"overload_529", `{"type":"error","error":{"status_code":529,"code":"overloaded","message":"overloaded"}}`, 529, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := []byte(tt.body)
			message := openai.ExtractOpenAISSEErrorMessage(payload)
			require.Equal(t, tt.status, openai.OpenAIStreamFailureStatus(payload, message))
			require.Equal(t, tt.wantFailover, openai.OpenAIStreamErrorEventShouldFailover(payload, message))
		})
	}
}

func TestOpenAIStreamBareErrorUsesSemanticFailover(t *testing.T) {
	payload := []byte(`{"error":{"type":"rate_limit_error","code":"rate_limit_exceeded","message":"slow down"}}`)
	require.True(t, openai.OpenAIStreamErrorEventShouldFailover(payload, "slow down"))
}

func TestOpenAIStream403FailoverRequiresStructuredProviderCredentialSignal(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		want    bool
	}{
		{
			name:    "ordinary permission error",
			payload: `{"type":"response.failed","response":{"error":{"type":"permission_error","code":"forbidden","message":"access denied for this request"}}}`,
		},
		{
			name:    "explicit 403 request status",
			payload: `{"type":"error","error":{"type":"permission_error","code":"forbidden","status_code":403,"message":"forbidden content"}}`,
		},
		{
			name:    "structured access state",
			payload: `{"type":"response.failed","response":{"error":{"code":"workspace_suspended","message":"workspace is suspended"}}}`,
			want:    true,
		},
		{
			name:    "explicit credential auth code",
			payload: `{"type":"error","error":{"type":"permission_error","code":"invalid_api_key","status_code":403,"message":"credential rejected"}}`,
			want:    true,
		},
		{
			name:    "explicit authentication type",
			payload: `{"type":"error","error":{"type":"authentication_error","status_code":403,"message":"credential rejected"}}`,
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := []byte(tt.payload)
			message := openai.ExtractOpenAISSEErrorMessage(payload)
			require.Equal(t, http.StatusForbidden, openai.OpenAIStreamFailureStatus(payload, message))
			require.Equal(t, tt.want, openai.OpenAIStreamFailedEventShouldFailover(payload, message))
			require.Equal(t, tt.want, openai.OpenAIStreamErrorEventShouldFailover(payload, message))
		})
	}
}

func TestOpenAIStream403PostOutputProviderSideEffectsIgnoreRequestPermissionErrors(t *testing.T) {
	repo := &openAIStream403ProviderRepo{}
	rateLimits := newUpstreamHealthForTest(repo, nil, nil, providercore.HealthOptions{}, nil)

	svc := newWSFixture(wsFixtureInputs{health: rateLimits})
	provider := &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, ID: 918, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeOAuth}}
	payload := []byte(`{"type":"error","error":{"type":"permission_error","code":"forbidden","status_code":403,"message":"access denied for this request"}}`)

	status, disabled := svc.Output.TerminalProviderEffects(nil, provider, payload, "access denied for this request", nil)

	require.Equal(t, http.StatusForbidden, status)
	require.False(t, disabled)
	require.Zero(t, repo.setErrorCalls)
	require.False(t, wsFixtureProviderBlocked(svc, provider))
}

func TestOpenAIStream403ExplicitCredentialAuthAppliesProviderSideEffects(t *testing.T) {
	repo := &openAIStream403ProviderRepo{}
	rateLimits := newUpstreamHealthForTest(repo, nil, nil, providercore.HealthOptions{}, nil)

	svc := newWSFixture(wsFixtureInputs{health: rateLimits})
	provider := &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, ID: 917, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeOAuth}}
	payload := []byte(`{"type":"error","error":{"type":"permission_error","code":"invalid_api_key","status_code":403,"message":"credential rejected"}}`)

	status, disabled := svc.Output.TerminalProviderEffects(nil, provider, payload, "credential rejected", nil)

	require.Equal(t, http.StatusForbidden, status)
	require.True(t, disabled)
	require.Equal(t, 1, repo.setErrorCalls)
	require.True(t, wsFixtureProviderBlocked(svc, provider))
}

func TestOpenAIWSStandaloneFailedStructured403AppliesProviderSideEffectsOnce(t *testing.T) {
	repo := &openAIStream403ProviderRepo{}
	svc := newWSFixture(wsFixtureInputs{health: newUpstreamHealthForTest(repo, nil, nil, providercore.HealthOptions{}, nil)})
	provider := &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, ID: 923, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeOAuth}}
	failed := []byte(`{"type":"response.failed","response":{"error":{"type":"permission_error","code":"invalid_api_key","status_code":403,"message":"credential rejected"}}}`)

	require.True(t, svc.handleOpenAIWSFailureProviderSideEffects(context.Background(), provider, "gpt-5", nil, failed))
	require.Equal(t, 1, repo.setErrorCalls)
	require.True(t, wsFixtureProviderBlocked(svc, provider))
}

func TestOpenAIWSPairedStructured403SideEffectsCanBeDeduplicated(t *testing.T) {
	repo := &openAIStream403ProviderRepo{}
	svc := newWSFixture(wsFixtureInputs{health: newUpstreamHealthForTest(repo, nil, nil, providercore.HealthOptions{}, nil)})
	provider := &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, ID: 924, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeOAuth}}
	errorEvent := []byte(`{"type":"error","error":{"code":"workspace_suspended","status_code":403,"message":"workspace is suspended"}}`)
	failedEvent := []byte(`{"type":"response.failed","response":{"error":{"code":"workspace_suspended","status_code":403,"message":"workspace is suspended"}}}`)

	applied := svc.handleOpenAIWSFailureProviderSideEffects(context.Background(), provider, "gpt-5", nil, errorEvent)
	if !applied {
		applied = svc.handleOpenAIWSFailureProviderSideEffects(context.Background(), provider, "gpt-5", nil, failedEvent)
	}

	require.True(t, applied)
	require.Equal(t, 1, repo.setErrorCalls)
}

func TestOpenAIStreamAccessStateAppliesProviderHealthBeforeFailover(t *testing.T) {
	svc := newWSFixture(wsFixtureInputs{})
	provider := &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, ID: 919, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeSetupToken}}
	payload := []byte(`{"type":"response.failed","response":{"error":{"code":"workspace_suspended","message":"workspace is suspended"}}}`)

	status, disabled := svc.Output.TerminalProviderEffects(nil, provider, payload, "workspace is suspended", nil)

	require.Equal(t, http.StatusForbidden, status)
	require.True(t, disabled)
	require.True(t, wsFixtureProviderBlocked(svc, provider))
}

func TestOpenAIStreamPairedFailureAppliesProviderSideEffectsOnce(t *testing.T) {
	const upstream = "data: {\"type\":\"response.output_text.delta\",\"delta\":\"partial\"}\n\n" +
		"data: {\"type\":\"error\",\"error\":{\"status_code\":403,\"code\":\"workspace_suspended\",\"message\":\"workspace is suspended\"}}\n\n" +
		"data: {\"type\":\"response.failed\",\"response\":{\"id\":\"resp_failed\",\"error\":{\"status_code\":403,\"code\":\"workspace_suspended\",\"message\":\"workspace is suspended\"}}}\n\n"

	t.Run("native", func(t *testing.T) {
		repo := &openAIStream403ProviderRepo{}
		svc := newWSFixture(wsFixtureInputs{options: &wsFixtureOptions{}, corrector: openai.NewCodexToolCorrector(), health: newUpstreamHealthForTest(repo, nil, nil, providercore.HealthOptions{}, nil)})
		provider := &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, ID: 921, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeOAuth}}
		recorder := newOpenAIResponseFlushRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       io.NopCloser(strings.NewReader(upstream)),
		}

		result, err := svc.Output.ReadStreamObservation(context.Background(), resp, c, provider, time.Now(), "gpt-5", "gpt-5", "")

		require.Error(t, err)
		require.NotNil(t, result)
		require.Equal(t, 1, repo.setErrorCalls)
	})

	t.Run("passthrough", func(t *testing.T) {
		repo := &openAIStream403ProviderRepo{}
		svc := newWSFixture(wsFixtureInputs{options: &wsFixtureOptions{}, health: newUpstreamHealthForTest(repo, nil, nil, providercore.HealthOptions{}, nil)})
		provider := &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, ID: 922, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeOAuth}}
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
		writer := &passthroughFlushTestWriter{
			ResponseWriter:  c.Writer,
			recorder:        recorder,
			failAfterWrites: -1,
		}
		c.Writer = writer
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       io.NopCloser(strings.NewReader(upstream)),
		}

		result, err := openai.ReadPassthroughStreaming(context.Background(), resp, upstreamcore.NewOutputContext(ResponseSink{Writer: c.Writer}), svc.Output.PassthroughOptions(context.Background(), c, provider), time.Now(), "gpt-5", "gpt-5")

		require.Error(t, err)
		require.NotNil(t, result)
		require.Equal(t, 1, repo.setErrorCalls)
	})
}

func TestOpenAIStreamOAuthLike429GetsDeadlineWithoutImmediateRuntimeBlock(t *testing.T) {
	for _, providerType := range []string{capability.ProviderTypeOAuth, capability.ProviderTypeSetupToken} {
		t.Run(providerType, func(t *testing.T) {
			svc := newWSFixture(wsFixtureInputs{})
			provider := &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, ID: 920, Platform: capability.PlatformOpenAI, Type: providerType}}
			payload := []byte(`{"type":"error","error":{"type":"rate_limit_error","code":"rate_limit_exceeded","message":"slow down"}}`)
			status, disabled := svc.Output.TerminalProviderEffects(nil, provider, payload, "slow down", nil)
			err := (gatewayprovider.OpenAIFailoverPolicy{Health: svc.Output.Health}).NewProviderFailure(provider, status, nil, payload, "slow down", disabled, false)

			require.Equal(t, http.StatusTooManyRequests, status)
			require.False(t, disabled)
			require.True(t, err.RetryableOnSameProvider)
			require.False(t, err.SameProviderRetryDeadline.IsZero())
			require.False(t, wsFixtureProviderBlocked(svc, provider))
		})
	}
}
