package provider

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/stretchr/testify/require"
)

// 真实供应商解析配合本地固定响应，缓存 key 相同但凭据身份改变。
type agUsageIdentityTransport struct{ calls atomic.Int32 }

func (t *agUsageIdentityTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	t.calls.Add(1)
	body := `{"models":{}}`
	if strings.Contains(r.URL.Path, "loadCodeAssist") {
		body = `{"currentTier":{"id":"FREE"}}`
	}
	return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
}

func TestAntigravityUsageCacheDoesNotCrossCredentialIdentity(t *testing.T) {
	transport := &agUsageIdentityTransport{}
	old := http.DefaultTransport
	http.DefaultTransport = transport
	t.Cleanup(func() { http.DefaultTransport = old })
	quota := &provider.AntigravityQuota{Options: AntigravityQuotaOptions(8*1024*1024, nil)}
	svc := provider.NewOAuthUsageService(nil, provider.NewOAuthUsageCache(), nil, provider.OAuthUsageOptions{Antigravity: provider.AntigravityUsageOptions{
		CanFetch: quota.CanFetch,
		Fetch: func(ctx context.Context, value *provider.Record) (*provider.UsageInfo, error) {
			result, err := quota.FetchQuota(ctx, value, quota.GetProxyURL(ctx, value))
			if result == nil {
				return nil, err
			}
			return result.UsageInfo, err
		},
		Degrade: AntigravityDegradedUsage, Enrich: EnrichUsageWithProviderError,
	}})
	a := &provider.Record{ID: 885, Platform: capability.PlatformAntigravity, Type: capability.ProviderTypeOAuth, Status: provider.StatusActive, Credentials: map[string]any{"access_token": "first", "project_id": "fixture"}}
	_, err := svc.GetAntigravityUsage(context.Background(), a)
	require.NoError(t, err)
	require.Equal(t, int32(2), transport.calls.Load())
	a.Credentials = map[string]any{"access_token": "second", "project_id": "fixture"}
	_, err = svc.GetAntigravityUsage(context.Background(), a)
	require.NoError(t, err)
	require.Equal(t, int32(4), transport.calls.Load())
}
