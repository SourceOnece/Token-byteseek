package provider

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/infra/httpclient"
	"github.com/imroc/req/v3"
	"github.com/stretchr/testify/require"
)

// 默认端点属于第三方协议；捕获生产 Options 发出的请求，不访问真实供应商。
func TestOpenAIPrivacyUsesNativeAccountSettingsEndpoint(t *testing.T) {
	client := req.C()
	var requests []*http.Request
	client.Transport.WrapRoundTripFunc(func(http.RoundTripper) req.HttpRoundTripFunc {
		return func(request *http.Request) (*http.Response, error) {
			requests = append(requests, request)
			return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"training_allowed":false}`)), Request: request}, nil
		}
	})
	options := OpenAIAuthorizationOptions(&OpenAIAuthorizationDependencies{PrivacyFactory: func(string) (*req.Client, error) { return client, nil }})
	require.Equal(t, "training_off", options.DisableTraining(context.Background(), "fixture-token", ""))
	require.Len(t, requests, 1)
	require.Equal(t, http.MethodPatch, requests[0].Method)
	require.Equal(t, "https", requests[0].URL.Scheme)
	require.Equal(t, "chatgpt.com", requests[0].URL.Host)
	require.Equal(t, "/backend-api/settings/account_user_setting", requests[0].URL.Path)
	require.Equal(t, "training_allowed", requests[0].URL.Query().Get("feature"))
	require.Equal(t, "false", requests[0].URL.Query().Get("value"))
}

// 使用独立代理地址隔离共享客户端缓存，验证默认 PAT 路径和原始身份字段。
func TestOpenAIPATUsesNativeAccountEndpoint(t *testing.T) {
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("unexpected network request")
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer proxy.Close()
	client, err := httpclient.GetClient(httpclient.Options{ProxyURL: proxy.URL, Timeout: 20 * time.Second, ResponseHeaderTimeout: 15 * time.Second})
	require.NoError(t, err)
	transport := client.Transport
	t.Cleanup(func() { client.Transport = transport })
	var requestedURL string
	client.Transport = req.HttpRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		requestedURL = request.URL.String()
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Request: request, Body: io.NopCloser(strings.NewReader(`{"email":"fixture@example.test","chatgpt_user_id":"user-fixture","chatgpt_account_id":"account-fixture","chatgpt_plan_type":"plus","chatgpt_account_is_fedramp":false}`))}, nil
	})
	options := OpenAIAuthorizationOptions(&OpenAIAuthorizationDependencies{})
	info, err := options.ValidatePAT(context.Background(), "at-fixture", proxy.URL)
	require.NoError(t, err)
	require.Equal(t, "https://auth.openai.com/api/accounts/v1/user-auth-credential/whoami", requestedURL)
	require.Equal(t, "account-fixture", info.ChatGPTAccountID)
}
