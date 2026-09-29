package httpapi

import (
	"github.com/TokenFlux/TokenRouter/internal/infra/httpclient/tlsfingerprint"
	"net/http"
)

// 守护观察原始上游响应，收据来自本轮构造；旁路请求与关闭开关不改变行为。
func (s *OpenAIRequests) SendWithTLS(req *http.Request, proxy string, id int64, concurrency int, profile *tlsfingerprint.Profile) (*http.Response, error) {
	resp, err := s.Transport.DoWithTLS(req, proxy, id, concurrency, profile)
	if err == nil && resp != nil && s.Tickets != nil {
		s.Tickets.ObserveResponse(req, resp)
	}
	return resp, err
}
