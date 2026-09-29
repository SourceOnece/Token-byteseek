package httpapi

import (
	"context"
	"time"

	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	gatewaytestkit "github.com/TokenFlux/TokenRouter/internal/gateway/testkit"
	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
)

type textFailureStore struct {
	gatewayprovider.ExecutionProviderStore
	tempUnschedCalls, rateLimitedCalls, updateCalls int
	modelRateLimitProviderID                        int64
	modelRateLimitKey                               string
}

func (s *textFailureStore) SetTempUnschedulable(context.Context, int64, time.Time, string) error {
	s.tempUnschedCalls++
	return nil
}

func (s *textFailureStore) SetRateLimited(context.Context, int64, time.Time) error {
	s.rateLimitedCalls++
	return nil
}

func (s *textFailureStore) SetRateLimitedIfLater(c context.Context, id int64, t time.Time) error {
	return s.SetRateLimited(c, id, t)
}

func (s *textFailureStore) UpdateExtra(context.Context, int64, map[string]any) error {
	s.updateCalls++
	return nil
}

func (s *textFailureStore) SetModelRateLimit(_ context.Context, id int64, key string, _ time.Time, _ ...string) error {
	s.modelRateLimitProviderID = id
	s.modelRateLimitKey = key
	return nil
}

// textFailureFixture 组合实际提供商健康、阻断与协议分类器，存储替身只记录写入。
func textFailureFixture(store gatewayprovider.ExecutionProviderStore, observe bool) *OpenAITextExecutor {
	blocks := providercore.NewRuntimeBlockState(time.Now)
	models := providercore.NewModelTransientState(0)
	var health *provideradapter.UpstreamHealth
	if observe {
		health = gatewaytestkit.NewHealthObserver(gatewaytestkit.HealthInput{Store: store, Options: providercore.HealthOptions{Block: blocks.BlockProviderScheduling}})
	}
	grokHealth := &provideradapter.GrokHealth{NormalizeModel: func(value *providercore.Record, model string) string {
		return (gatewayprovider.ModelPolicy{Record: value}).NormalizeOpenAI(model)
	}, Store: store, Health: health, Runtime: blocks, ModelTransient: models}
	return &OpenAITextExecutor{Output: &OpenAIResponseOutput{Health: &provideradapter.OpenAIResponseHealth{Health: health, Runtime: blocks, ModelTransient: models}}, Grok: &GrokExecutor{Health: grokHealth}}
}
