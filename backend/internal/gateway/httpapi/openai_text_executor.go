package httpapi

import (
	"time"

	gatewayadapter "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/gateway/session"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
)

// OpenAITextExecutor 组合协议执行、固定请求构造与独立会话状态，不拥有选号或完成队列。
type OpenAITextExecutor struct {
	Compact        *CompactExecutor
	Requests       *OpenAIRequests
	Output         *OpenAIResponseOutput
	Grok           *GrokExecutor
	Credentials    *gatewayadapter.RequestCredentials
	FastPolicy     *gatewayadapter.ExecutionFastPolicy
	Continuation   *session.CompatResponses
	PromptCache    *session.AnthropicPromptCache
	CodexUsage     *provideradapter.CodexUsageObserver
	ForcedTemplate string
	ResponseTTL    func() time.Duration
}
