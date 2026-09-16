//go:build unit

package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenCodeUsageKeepsPrefixAndModeIsolation(t *testing.T) {
	for _, base := range []string{"https://relay.example/prefix/v1", "https://relay.example/prefix"} {
		account := &Account{ID: 911, Platform: PlatformOpenCodeGo, Type: AccountTypeAPIKey, Status: StatusActive, Credentials: map[string]any{"api_key": "test", "account_mode": AccountModeGo, "base_url": base}}
		upstream := &upstreamUsageHTTPStub{}
		upstream.responses = append(upstream.responses, struct {
			status int
			body   string
			err    error
		}{status: 200, body: `{"usage":{"rolling":{"percent":25,"resetsAt":"2030-01-01T00:00:00Z"},"weekly":{"percent":50},"monthly":{"percent":80}}}`})
		svc := NewUpstreamUsageService(&upstreamUsageAccountRepoStub{account: account}, upstream, testUpstreamUsageConfig(), nil)
		result, err := svc.QueryAccount(context.Background(), account.ID)
		require.NoError(t, err)
		require.Len(t, result.Limits, 3)
		require.Equal(t, UpstreamUsageAdapterOpenCodeGo, result.Adapter)
		require.Equal(t, "https://relay.example/prefix/v1/usage", upstream.requests[0].URL.String())
		require.Equal(t, http.MethodGet, upstream.requests[0].Method)
		require.Equal(t, "Bearer test", upstream.requests[0].Header.Get("Authorization"))
		require.Equal(t, float64(80), *result.Limits[2].Used)
	}
	account := &Account{ID: 912, Platform: PlatformOpenCodeGo, Type: AccountTypeAPIKey, Status: StatusActive, Credentials: map[string]any{"api_key": "test", "account_mode": AccountModeZen}}
	upstream := &upstreamUsageHTTPStub{}
	svc := NewUpstreamUsageService(&upstreamUsageAccountRepoStub{account: account}, upstream, testUpstreamUsageConfig(), nil)
	_, err := svc.QueryAccount(context.Background(), account.ID)
	require.ErrorIs(t, err, ErrUpstreamUsageUnsupported)
	require.Empty(t, upstream.requests)
	for _, body := range []string{`{}`, `{"usage":{"rolling":{"percent":"NaN"}}}`, `{"usage":{"rolling":{"percent":-1}}}`} {
		require.Empty(t, parseOpenCodeGoUsageTiers([]byte(body)))
	}
}
