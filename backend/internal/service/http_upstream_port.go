package service

import "github.com/TokenFlux/TokenRouter/internal/egress"

// HTTPUpstream 上游 HTTP 请求接口
// 用于向上游 API（Claude、OpenAI、Gemini 等）发送请求
type HTTPUpstream = egress.HTTPUpstream
