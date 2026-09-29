package app

import (
	gatewayhttp "github.com/TokenFlux/TokenRouter/internal/gateway/httpapi"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
)

// provideOpenAIImages 复用现有请求、输出和活动屏障；模型冷却仍归提供商拥有者。
func provideOpenAIImages(text *gatewayhttp.OpenAITextExecutor, activity *gatewayRequestActivity) *gatewayhttp.OpenAIImagesExecutor {
	cooldown := &provideradapter.ImageToolCooldown{Store: text.Requests.Providers}
	if text.Requests.Readers != nil {
		cooldown.Settings = text.Requests.Readers.Provider.GetOpenAIImagesOAuthUnavailableCooldownSettings
	}
	result := &gatewayhttp.OpenAIImagesExecutor{Requests: text.Requests, Output: text.Output, Cooldown: cooldown, Enter: activity.Enter}
	return result
}
