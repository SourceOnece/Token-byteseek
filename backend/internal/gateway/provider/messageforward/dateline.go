package messageforward

import (
	"context"

	gatewayadapter "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/upstream/anthropic"
)

// 日期指纹只在 Anthropic OAuth/SetupToken 的开关开启时处理。
func (r *Runtime) shouldNormalizeDateline(ctx context.Context, target *gatewayadapter.ExecutionProvider) bool {
	return target != nil && target.View().IsAnthropicOAuthOrSetupToken() && r.dependencies.Settings != nil && r.dependencies.Settings.IsClientDatelineNormalizationEnabled(ctx)
}

func (r *Runtime) normalizeDateline(ctx context.Context, target *gatewayadapter.ExecutionProvider, body []byte) ([]byte, bool) {
	if !r.shouldNormalizeDateline(ctx, target) {
		return nil, false
	}
	next, _, changed := anthropic.NormalizeDateline(body)
	if !changed {
		return nil, false
	}
	return next, true
}
