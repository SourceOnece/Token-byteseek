package app

import (
	"time"

	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/TokenFlux/TokenRouter/internal/gateway/httpapi"
	gatewayadapter "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
	"github.com/TokenFlux/TokenRouter/internal/scheduler"
)

// provideUnifiedTextExecutor 固定注入已有执行器，跨平台切号不创建第二套连接或资源池。
func provideUnifiedTextExecutor(openai *httpapi.OpenAIResponsesExecutor, anthropic *httpapi.MessagesExecutor, gemini *httpapi.GeminiExecutor, antigravity *httpapi.AntigravityExecutor, qoder *gatewayadapter.QoderRuntime, refresh *provideradapter.QoderRequestRefresh, queue *scheduler.UserMessageQueueService, cfg *config.Config) *httpapi.UnifiedTextExecutor {
	executor := &httpapi.UnifiedTextExecutor{OpenAI: openai, Anthropic: anthropic, Gemini: gemini, Antigravity: antigravity, Qoder: qoder, QoderRefresh: refresh}
	if cfg != nil {
		executor.MessageQueueMode = cfg.Gateway.UserMessageQueue.GetEffectiveMode()
		executor.MessageQueueWait = cfg.Gateway.UserMessageQueue.WaitTimeout()
		if queue != nil {
			executor.MessageQueue = httpapi.NewUserMsgQueueHelper(queue, httpapi.SSEPingFormatClaude, time.Duration(cfg.Concurrency.PingInterval)*time.Second)
		}
	}
	return executor
}
