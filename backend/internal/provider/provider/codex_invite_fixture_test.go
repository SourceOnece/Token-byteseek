package provider

import (
	"context"
	"io"
	"net/http"
	"strings"

	providercore "github.com/TokenFlux/TokenRouter/internal/provider"

	"github.com/TokenFlux/TokenRouter/internal/infra/httpclient/tlsfingerprint"

	"github.com/TokenFlux/TokenRouter/internal/egress"
)

type codexInviteResetAdminServiceStub struct {
	provider *providercore.Record
	proxy    *egress.Proxy
}

func (s codexInviteResetAdminServiceStub) GetProvider(ctx context.Context, id int64) (*providercore.Record, error) {
	return s.provider, nil
}

func (s codexInviteResetAdminServiceStub) GetProxy(ctx context.Context, id int64) (*egress.Proxy, error) {
	return s.proxy, nil
}

type codexInviteResetHTTPUpstreamStub struct {
	responses []*http.Response
	requests  []*http.Request
	bodies    []string
	profiles  []*tlsfingerprint.Profile
}

func (s *codexInviteResetHTTPUpstreamStub) Do(req *http.Request, proxyURL string, providerID int64, providerConcurrency int) (*http.Response, error) {
	return s.DoWithTLS(req, proxyURL, providerID, providerConcurrency, nil)
}

func (s *codexInviteResetHTTPUpstreamStub) DoWithTLS(req *http.Request, proxyURL string, providerID int64, providerConcurrency int, profile *tlsfingerprint.Profile) (*http.Response, error) {
	body := ""
	if req.Body != nil {
		payload, _ := io.ReadAll(req.Body)
		body = string(payload)
		req.Body = io.NopCloser(strings.NewReader(body))
	}
	s.requests = append(s.requests, req)
	s.bodies = append(s.bodies, body)
	s.profiles = append(s.profiles, profile)
	if len(s.responses) == 0 {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
	}
	resp := s.responses[0]
	s.responses = s.responses[1:]
	return resp, nil
}

func codexInviteResetJSONResponse(body string) *http.Response {
	return codexInviteResetJSONStatusResponse(http.StatusOK, body)
}

func codexInviteResetJSONStatusResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
