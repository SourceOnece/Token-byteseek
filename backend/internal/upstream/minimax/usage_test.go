package minimax

import (
	"context"
	"github.com/TokenFlux/TokenRouter/internal/upstream/usagecontract"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"strings"
	"testing"
)

// 查询必须使用管理员配置的中继主机，不将密钥转发到另一域名。
func TestCodingUsageThroughConfiguredRelay(t *testing.T) {
	input := &usagecontract.Request{BaseURL: "https://relay.invalid/v1", APIKey: "fixture-secret", Context: func(ctx context.Context) context.Context { return ctx }, ApplyHeaders: func(http.Header) {}}
	input.Do = func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "relay.invalid", req.URL.Host)
		require.Equal(t, "/v1/api/openplatform/coding_plan/remains", req.URL.Path)
		require.Equal(t, "Bearer fixture-secret", req.Header.Get("Authorization"))
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"base_resp":{"status_code":0},"model_remains":[{"model_name":"general","current_interval_remaining_percent":60,"end_time":1790000000000,"current_weekly_status":1,"current_weekly_remaining_percent":25,"weekly_end_time":1790100000000}]}`))}, nil
	}
	result, err := (&CodingUsageAdapter{}).Query(context.Background(), input)
	require.NoError(t, err)
	require.Len(t, result.Limits, 2)
	tiers := parseMiniMaxUsageTiers([]byte(`{"model_remains":[{"model_name":"video","current_interval_remaining_percent":100}]}`))
	require.Empty(t, tiers)
}
