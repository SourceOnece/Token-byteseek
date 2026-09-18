package repository

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type ticketIPRoundTripper func(*http.Request) (*http.Response, error)

func (f ticketIPRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func ticketIPResponse(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}

// 默认端点均 HTTPS，前一个超时不取消后一个，且不夹带 OAuth 身份或业务请求。
func TestCodexTicketReferenceIPHTTPSFallback(t *testing.T) {
	p := &proxyProbeService{}
	calls := 0
	c := &http.Client{Transport: ticketIPRoundTripper(func(r *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, "https", r.URL.Scheme)
		require.Empty(t, r.Header.Get("Authorization"))
		require.Empty(t, r.Header.Get("Cookie"))
		require.True(t, r.Close)
		if calls == 1 {
			return nil, context.DeadlineExceeded
		}
		require.NoError(t, r.Context().Err())
		return ticketIPResponse(200, `{"ip":"203.0.113.8"}`), nil
	})}
	result := p.probeTicketReferenceTargets(context.Background(), c)
	require.Equal(t, 2, calls)
	require.Equal(t, "reference", result.Status)
	require.Equal(t, "ipify", result.Source)
	require.Equal(t, "203.0.113.8", result.IP)
}
func TestCodexTicketReferenceIPFailuresAreSafe(t *testing.T) {
	for _, tc := range []struct {
		code int
		body string
		err  error
		want string
	}{
		{403, "secret", nil, "http_error"}, {200, "secret", nil, "invalid_response"}, {200, `{"ip":"not-ip-secret"}`, nil, "invalid_response"}, {0, "", context.DeadlineExceeded, "timeout"}, {0, "", errors.New("socks5://password-secret@proxy"), "network"},
	} {
		t.Run(tc.want, func(t *testing.T) {
			p := &proxyProbeService{configuredProbeURLs: []configuredProbeTarget{{"https://configured.invalid", "ipify"}}}
			c := &http.Client{Transport: ticketIPRoundTripper(func(*http.Request) (*http.Response, error) {
				if tc.err != nil {
					return nil, tc.err
				}
				return ticketIPResponse(tc.code, tc.body), nil
			})}
			result := p.probeTicketReferenceTargets(context.Background(), c)
			require.Equal(t, tc.want, result.Status)
			require.Equal(t, "configured", result.Source)
			require.Empty(t, result.IP)
		})
	}
	p := &proxyProbeService{}
	require.Equal(t, "proxy_config", p.ProbeTicketReferenceIP(context.Background(), "").Status)
}
