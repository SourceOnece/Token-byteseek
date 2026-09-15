package service

import (
	"net/http"

	"github.com/TokenFlux/TokenRouter/internal/pkg/tlsfingerprint"
)

// 图片协议测试的函数式传输桩；不访问真实上游或使用生产凭据。
type codexModelsHTTPUpstreamStub struct {
	do func(*http.Request, string, int64, int) (*http.Response, error)
}

func (s *codexModelsHTTPUpstreamStub) Do(req *http.Request, proxy string, id int64, concurrency int) (*http.Response, error) {
	return s.do(req, proxy, id, concurrency)
}

func (s *codexModelsHTTPUpstreamStub) DoWithTLS(req *http.Request, proxy string, id int64, concurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return s.Do(req, proxy, id, concurrency)
}
