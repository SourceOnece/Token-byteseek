package httpapi

import provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"

// OpenAIImagesExecutor 组合图片单次执行与响应处理，不持有提供商选择或结算循环。
type OpenAIImagesExecutor struct {
	Requests *OpenAIRequests
	Output   *OpenAIResponseOutput
	Cooldown *provideradapter.ImageToolCooldown
	Enter    func() (func(), error)
}
