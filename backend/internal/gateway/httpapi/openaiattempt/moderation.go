package openaiattempt

import (
	"github.com/TokenFlux/TokenRouter/internal/apikey"
	forwardcore "github.com/TokenFlux/TokenRouter/internal/gateway/forward"
	gatewayhttp "github.com/TokenFlux/TokenRouter/internal/gateway/httpapi"
	"github.com/TokenFlux/TokenRouter/internal/gateway/moderationflow"
	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/moderation"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (h *Support) RecordOpenAICyberWarning(c *gin.Context, reqLog *zap.Logger, apiKey *apikey.APIKey, provider *gatewayprovider.ExecutionProvider, model string, statusCode int, responseBody []byte, warningText string) {
	h.RecordOpenAICyberWarningWithPromptExcerpt(c, reqLog, apiKey, provider, model, statusCode, responseBody, warningText, gatewayhttp.CurrentOpenAICyberWarningPromptExcerpt(c))
}

func (h *Support) RecordOpenAICyberWarningWithPromptExcerpt(c *gin.Context, reqLog *zap.Logger, apiKey *apikey.APIKey, provider *gatewayprovider.ExecutionProvider, model string, statusCode int, responseBody []byte, warningText string, promptExcerpt string) bool {
	return h.RecordOpenAICyberWarningWithSnapshot(c, reqLog, apiKey, provider, model, statusCode, responseBody, warningText, promptExcerpt, gatewayhttp.CurrentOpenAICyberWarningSnapshot(c))
}

func (h *Support) RecordOpenAICyberWarningWithSnapshot(c *gin.Context, reqLog *zap.Logger, apiKey *apikey.APIKey, provider *gatewayprovider.ExecutionProvider, model string, statusCode int, responseBody []byte, warningText string, promptExcerpt string, snapshot moderation.ContentModerationInput) bool {
	if h == nil {
		return false
	}
	return gatewayhttp.RecordOpenAICyberWarningWithSnapshot(gatewayhttp.GatewayModerationEndpoints{}, h.Moderation, c, reqLog, apikey.CopyAPIKey(apiKey), moderationProviderView(provider), model, statusCode, responseBody, warningText, promptExcerpt, snapshot)
}

// RecordOpenAIForwardErrorCyberWarning 记录错误链中携带的上游 cyber 风控警告。
func (h *Support) RecordOpenAIForwardErrorCyberWarning(c *gin.Context, reqLog *zap.Logger, apiKey *apikey.APIKey, provider *gatewayprovider.ExecutionProvider, model string, statusCode int, err error) bool {
	warning, ok := forwardcore.WarningFromError(err)
	if !ok || warning == nil {
		return false
	}
	if warning.StatusCode > 0 {
		statusCode = warning.StatusCode
	}
	h.RecordOpenAICyberWarning(c, reqLog, apiKey, provider, model, statusCode, warning.ResponseBody, warning.Message)
	return true
}

// 仅投影处置所需提供商标识，不泄露执行凭据。
func moderationProviderView(a *gatewayprovider.ExecutionProvider) *moderationflow.Provider {
	if a == nil {
		return nil
	}
	return &moderationflow.Provider{ID: a.Record.ID, Name: a.Record.Name, Platform: a.Record.Platform}
}
