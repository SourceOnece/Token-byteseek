// Target 是一次任务尝试的受控执行目标，不保存共享缓存、提供商仓储或网关服务。
package provider

import (
	"context"
	"net/http"

	"github.com/TokenFlux/TokenRouter/internal/creative"

	gemininative "github.com/TokenFlux/TokenRouter/internal/upstream/gemini"
	"github.com/TokenFlux/TokenRouter/internal/upstream/grok"
)

type OpenAIOptions struct {
	OAuth bool
	// BuildOAuth 在任务执行时构造 Codex 认证与隔离会话。
	BuildOAuth   func(context.Context, creative.CreativeRun, []byte, string, string) (*http.Request, error)
	Token        func(context.Context) (string, error)
	URL          func(string) (string, error)
	Prepare      func(*http.Request) *http.Request
	AuthHeaders  func(context.Context, string) (http.Header, error)
	ApplyHeaders func(http.Header)
	Do           func(*http.Request) (*http.Response, error)
	// FetchImage 下载结果图片，不应用生图请求的认证或自定义请求头。
	FetchImage func(context.Context, string) (string, error)
}
type GrokOptions struct {
	OAuth        bool
	Token        func(context.Context) (string, error)
	URL          func(grok.GrokMediaEndpoint) (string, error)
	Prepare      func(*http.Request) *http.Request
	ApplyHeaders func(http.Header)
	Do           func(*http.Request) (*http.Response, error)
}
type Target struct {
	OpenAI *OpenAIOptions
	Grok   *GrokOptions
	Gemini func(string) gemininative.ImageOptions
}

// ExecutePlatform 保留任务实际提供商平台分派，不改变各平台独立协议与错误语义。
func (t *Target) ExecutePlatform(ctx context.Context, platform string, run creative.CreativeRun, payload creative.CreativeRunPayload, model string) ([]creative.CreativeOutput, error) {
	switch platform {
	case creative.PlatformOpenAI:
		return t.ExecuteOpenAI(ctx, run, payload, model)
	case creative.PlatformGrok:
		return t.ExecuteGrok(ctx, run, payload, model)
	case creative.PlatformGemini:
		return t.ExecuteGemini(ctx, run, payload, model)
	default:
		return nil, creative.CreativeNonRetryableError("creative executor unsupported provider platform %s", platform)
	}
}
