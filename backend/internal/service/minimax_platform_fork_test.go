//go:build unit

package service

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestMiniMaxAccountProtocolAndQuotaContract(t *testing.T) {
	for _, mode := range []string{AccountModePayG, AccountModeCoding} {
		for _, protocol := range []string{APIProtocolChatCompletions, APIProtocolAnthropic, APIProtocolResponses, APIProtocolAdaptive} {
			account := &Account{Platform: PlatformMiniMax, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "test", "account_mode": mode, "api_protocol": protocol}}
			require.NoError(t, normalizeCNProviderCredentials(account, true))
			require.True(t, account.IsOpenAICompatible())
			require.True(t, account.IsMiniMax())
			require.Equal(t, protocol, account.GetAPIProtocol())
			require.Equal(t, PlatformMiniMax, NormalizeOpenAICompatiblePlatform(account.Platform))
			require.Equal(t, DefaultMiniMaxBaseURL, account.defaultCNProtocolBaseURL(APIProtocolResponses))
			require.Equal(t, DefaultMiniMaxAnthropicBaseURL, account.defaultCNProtocolBaseURL(APIProtocolAnthropic))
			if mode == AccountModeCoding {
				require.Equal(t, UpstreamUsageAdapterMiniMaxCoding, cnUpstreamUsageAdapterName(account))
			} else {
				require.Empty(t, cnUpstreamUsageAdapterName(account))
			}
		}
	}
	require.Error(t, normalizeCNProviderCredentials(&Account{Platform: PlatformMiniMax, Type: AccountTypeOAuth}, true))
	require.Contains(t, AllowedQuotaPlatforms, PlatformMiniMax)
	require.Contains(t, schedulerSnapshotPlatforms(), PlatformMiniMax)
	require.Len(t, domain.SupportedGroupClientProtocols(PlatformMiniMax), 3)
	require.Equal(t, MiniMaxDefaultModelIDs(), defaultRequestModelIDsForPlatform(PlatformMiniMax))
	require.Equal(t, MiniMaxDefaultModelIDs(), defaultModelsListCandidateIDs(PlatformMiniMax))
	require.Len(t, defaultMarketplaceModelDefs(PlatformMiniMax), len(MiniMaxDefaultModelIDs()))
	for _, platform := range []string{PlatformKimi, PlatformMiniMax} {
		require.NoError(t, normalizeCNProviderCredentials(&Account{Platform: platform, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_protocol": APIProtocolResponses}}, true))
	}
}

func TestMiniMaxUsageKeepsConfiguredHostAndRejectsBadData(t *testing.T) {
	const body = `{"base_resp":{"status_code":0},"model_remains":[{"model_name":"video","current_interval_remaining_percent":0},{"model_name":"general","current_interval_remaining_percent":25,"end_time":1789000000000,"current_weekly_status":1,"current_weekly_remaining_percent":60,"weekly_end_time":1789600000}]}`
	for _, base := range []string{"https://api.minimaxi.com/anthropic", "https://api.minimax.io/v1", "https://relay.example/custom"} {
		account := &Account{ID: 901, Platform: PlatformMiniMax, Type: AccountTypeAPIKey, Status: StatusActive, Concurrency: 1, Credentials: map[string]any{"api_key": "test-minimax", "account_mode": AccountModeCoding, "base_url": base}}
		repo := &upstreamUsageAccountRepoStub{account: account}
		upstream := &upstreamUsageHTTPStub{}
		upstream.responses = append(upstream.responses, struct {
			status int
			body   string
			err    error
		}{status: 200, body: body})
		svc := NewUpstreamUsageService(repo, upstream, testUpstreamUsageConfig(), nil)
		result, err := svc.QueryAccount(context.Background(), account.ID)
		require.NoError(t, err)
		require.Equal(t, UpstreamUsageAdapterMiniMaxCoding, result.Adapter)
		require.Len(t, result.Limits, 2)
		require.Equal(t, float64(75), *result.Limits[0].Used)
		require.Equal(t, float64(40), *result.Limits[1].Used)
		require.Equal(t, time.Unix(1789600000, 0).UTC(), *result.Limits[1].ResetAt)
		require.Len(t, upstream.requests, 1)
		request := upstream.requests[0]
		configured, parseErr := url.Parse(base)
		require.NoError(t, parseErr)
		require.Equal(t, configured.Host, request.URL.Host)
		require.Equal(t, http.MethodGet, request.Method)
		require.Equal(t, "/v1/api/openplatform/coding_plan/remains", request.URL.Path)
		require.Equal(t, "Bearer test-minimax", request.Header.Get("Authorization"))
		require.True(t, HTTPUpstreamRedirectsDisabled(request.Context()))
	}
	for _, bad := range []string{`{}`, `{"model_remains":[]}`, `{"model_remains":[{"model_name":"general","current_interval_remaining_percent":-1}]}`, `{"model_remains":[{"model_name":"general","current_interval_remaining_percent":"NaN"}]}`} {
		require.Empty(t, parseMiniMaxUsageTiers([]byte(bad)))
	}
	account := &Account{ID: 902, Platform: PlatformMiniMax, Type: AccountTypeAPIKey, Status: StatusActive, Credentials: map[string]any{"api_key": "test-minimax", "account_mode": AccountModePayG}}
	upstream := &upstreamUsageHTTPStub{}
	svc := NewUpstreamUsageService(&upstreamUsageAccountRepoStub{account: account}, upstream, testUpstreamUsageConfig(), nil)
	_, err := svc.QueryAccount(context.Background(), account.ID)
	require.ErrorIs(t, err, ErrUpstreamUsageUnsupported)
	require.Empty(t, upstream.requests)
	require.False(t, cnUsageOfficialHost(PlatformMiniMax, "api.minimax.io.evil.test"))
}
