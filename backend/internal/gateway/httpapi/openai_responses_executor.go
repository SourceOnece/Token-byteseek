package httpapi

import (
	"context"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/egress"
	"github.com/TokenFlux/TokenRouter/internal/gateway/forward"
	gatewayadapter "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/gin-gonic/gin"
)

// OpenAIHTTPWSAttempt 固化 HTTP 请求转入 WS 执行时的参数，不重做请求转换或选提供商。
type OpenAIHTTPWSAttempt struct {
	ClientPromptCacheKey, Token                      string
	Decision                                         egress.OpenAIWSProtocolDecision
	CodexCLI, Stream                                 bool
	OriginalModel, UpstreamModel, BillingModel       string
	ImageBillingModel, ImageSizeTier, ImageInputSize string
	StartedAt                                        time.Time
	TLS                                              egress.TLSFingerprintRouterMatchResult
	LineageGroupID                                   int64
	LineageSessionHash                               string
	LineageEntryBody                                 []byte
}

// OpenAIResponsesExecutor 组合固定请求、输出与协议执行能力；连接资源由各传输拥有者管理。
type OpenAIResponsesExecutor struct {
	Requests         *OpenAIRequests
	Output           *OpenAIResponseOutput
	Text             *OpenAITextExecutor
	Grok             *GrokExecutor
	Lineage          *OpenAIEncryptedLineage
	ImageBridge      *gatewayadapter.ResponseImagePolicy
	ResolveTransport func(*gatewayadapter.ExecutionProvider) egress.OpenAIWSProtocolDecision
	WebSocket        func(context.Context, *gin.Context, *gatewayadapter.ExecutionProvider, map[string]any, OpenAIHTTPWSAttempt) (*forward.OpenAIResult, error)
}
