package opencode

import (
	"context"
	"github.com/TokenFlux/TokenRouter/internal/upstream/usagecontract"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"strings"
	"testing"
)

// 自定义中继保留路径前缀；Zen 不发送 Go 配额请求。
func TestGoUsagePlanAndRelay(t *testing.T) {
	calls := 0
	input := &usagecontract.Request{BaseURL: "https://relay.invalid/zen/go", OpenCodeGo: true, APIKey: "fixture", Context: func(ctx context.Context) context.Context { return ctx }, ApplyHeaders: func(http.Header) {}, Endpoint: func(base, path string) string {
		require.Equal(t, "/v1/usage", path)
		return base + path
	}}
	input.Do = func(req *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, "https://relay.invalid/zen/go/v1/usage", req.URL.String())
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"usage":{"rolling":{"percent":35},"weekly":{"percent":70}}}`))}, nil
	}
	result, err := (&GoUsageAdapter{}).Query(context.Background(), input)
	require.NoError(t, err)
	require.Len(t, result.Limits, 2)
	input.OpenCodeGo = false
	_, err = (&GoUsageAdapter{}).Query(context.Background(), input)
	require.Error(t, err)
	require.Equal(t, 1, calls)
}
