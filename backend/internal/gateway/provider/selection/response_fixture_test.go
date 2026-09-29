package selection

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/TokenFlux/TokenRouter/internal/egress"
	gatewayadapter "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/gateway/session"
	"github.com/TokenFlux/TokenRouter/internal/scheduler"
	"github.com/TokenFlux/TokenRouter/internal/scheduler/rediscache/codec"
)

// responseSelectionOptions 保留原 WSv2 夹具的显式开关和零值等待配置。
func responseSelectionOptions() Options {
	return Options{
		WS: &egress.OpenAIWSOptions{Enabled: true, OAuthEnabled: true, APIKeyEnabled: true, ResponsesWebsocketsV2: true},

		StickyTTL:   time.Hour,
		ResponseTTL: time.Hour,
	}
}

func responseSelectionParameters() *scheduler.Parameters {
	return scheduler.NewParameters(scheduler.NewSettingsRuntime(scheduler.Diagnostics{}), nil, diagnosticParameterDefaults(&config.Config{}))
}

// selectPreviousResponseForTest 组合原入口的上下文及模型投影，不为私有合同扩大生产 API。
func selectPreviousResponseForTest(s *Compatible, ctx context.Context, group *int64, previous, model string, excluded map[int64]struct{}, compact bool) (*gatewayadapter.SelectionResult, error) {
	ctx = s.withCandidatePolicy(ctx, group, "")
	ctx = s.withOpenAIGroupPrivacyRequirement(ctx, group)
	model = s.resolveGroupRoutingModel(ctx, group, model)
	return s.selectProviderByPreviousResponseIDForCapability(ctx, group, previous, model, excluded, "", compact)
}

// 以下替身仅实现选择合同实际使用的读取；意外访问其他能力直接暴露测试缺口。
type selectionProviderFixture struct {
	Providers
	providers []gatewayadapter.ExecutionProvider
}

func (r selectionProviderFixture) GetByID(ctx context.Context, id int64) (*gatewayadapter.ExecutionProvider, error) {
	for i := range r.providers {
		if r.providers[i].Record.ID == id {
			prepareSelectionFixtureProvider(ctx, &r.providers[i], nil)
			return &r.providers[i], nil
		}
	}
	return nil, errors.New("provider not found")
}

func (r selectionProviderFixture) ListSchedulableByPlatform(_ context.Context, platform string) ([]gatewayadapter.ExecutionProvider, error) {
	var out []gatewayadapter.ExecutionProvider
	for _, value := range r.providers {
		if value.Record.Platform == platform {
			out = append(out, value)
		}
	}
	return out, nil
}

func (r selectionProviderFixture) ListSchedulableByGroupIDAndPlatform(ctx context.Context, groupID int64, platform string) ([]gatewayadapter.ExecutionProvider, error) {
	return r.ListSchedulableByGroupIDAndPlatforms(ctx, groupID, []string{platform})
}

func (r selectionProviderFixture) ListSchedulableUngroupedByPlatform(ctx context.Context, platform string) ([]gatewayadapter.ExecutionProvider, error) {
	return r.ListSchedulableByPlatform(ctx, platform)
}

type responseCacheFixture struct {
	stickyCacheFixture
	session.GatewayCache
}

func (c *responseCacheFixture) GetSessionProviderID(ctx context.Context, group int64, key string) (int64, error) {
	return c.stickyCacheFixture.GetSessionProviderID(ctx, group, key)
}

func (c *responseCacheFixture) SetSessionProviderID(ctx context.Context, group int64, key string, id int64, ttl time.Duration) error {
	return c.stickyCacheFixture.SetSessionProviderID(ctx, group, key, id, ttl)
}

func (c *responseCacheFixture) RefreshSessionTTL(ctx context.Context, group int64, key string, ttl time.Duration) error {
	return c.stickyCacheFixture.RefreshSessionTTL(ctx, group, key, ttl)
}

func (c *responseCacheFixture) DeleteSessionProviderID(ctx context.Context, group int64, key string) error {
	return c.stickyCacheFixture.DeleteSessionProviderID(ctx, group, key)
}

type selectionConcurrencyFixture struct {
	scheduler.ConcurrencyCache
	acquireResults  map[int64]bool
	waitCounts      map[int64]int
	loadBatchErr    error
	loadMap         map[int64]*scheduler.ProviderLoadInfo
	skipDefaultLoad bool
}

func (c selectionConcurrencyFixture) AcquireProviderSlot(_ context.Context, id int64, _ int, _ string) (bool, error) {
	if value, ok := c.acquireResults[id]; ok {
		return value, nil
	}
	return true, nil
}

func (selectionConcurrencyFixture) ReleaseProviderSlot(context.Context, int64, string) error {
	return nil
}

func (c selectionConcurrencyFixture) GetProviderWaitingCount(_ context.Context, id int64) (int, error) {
	return c.waitCounts[id], nil
}

func (c selectionConcurrencyFixture) GetProvidersLoadBatch(ctx context.Context, providers []scheduler.ProviderWithConcurrency) (map[int64]*scheduler.ProviderLoadInfo, error) {
	if c.loadBatchErr != nil {
		return nil, c.loadBatchErr
	}
	out := make(map[int64]*scheduler.ProviderLoadInfo, len(providers))
	if c.skipDefaultLoad && c.loadMap != nil {
		for _, acc := range providers {
			if load, ok := c.loadMap[acc.ID]; ok {
				out[acc.ID] = load
			}
		}
		return out, nil
	}
	for _, acc := range providers {
		if c.loadMap != nil {
			if load, ok := c.loadMap[acc.ID]; ok {
				out[acc.ID] = load
				continue
			}
		}
		out[acc.ID] = &scheduler.ProviderLoadInfo{ProviderID: acc.ID, LoadRate: 0}
	}
	return out, nil
}

// selectionSnapshotFixture 经真实快照读取器解码，数据库重检仍读取另一份新状态。
type selectionSnapshotFixture struct {
	scheduler.SnapshotCache
	providersByID map[int64]*gatewayadapter.ExecutionProvider
}

func (s *selectionSnapshotFixture) GetProvider(_ context.Context, id int64) (scheduler.SnapshotProvider, error) {
	value := s.providersByID[id]
	if value == nil {
		return nil, nil
	}
	copy := value.Record
	return codec.WrapRecord(&copy), nil
}

type groupAwareStubOpenAIProviderRepo struct {
	selectionProviderFixture
}

func (r groupAwareStubOpenAIProviderRepo) ListSchedulableByGroupIDAndPlatform(ctx context.Context, groupID int64, platform string) ([]gatewayadapter.ExecutionProvider, error) {
	var result []gatewayadapter.ExecutionProvider
	for _, acc := range r.providers {
		if acc.Record.Platform == platform && openAIStickyProviderMatchesGroup(&acc, &groupID) {
			result = append(result, acc)
		}
	}
	return result, nil
}

func (r groupAwareStubOpenAIProviderRepo) ListSchedulableUngroupedByPlatform(ctx context.Context, platform string) ([]gatewayadapter.ExecutionProvider, error) {
	var result []gatewayadapter.ExecutionProvider
	for _, acc := range r.providers {
		if acc.Record.Platform == platform && openAIStickyProviderMatchesGroup(&acc, nil) {
			result = append(result, acc)
		}
	}
	return result, nil
}

// codex 配额读取合同保留写入哨兵，任何原不应发生的持久化仍使断言失败。
type openAICodexExtraListRepo struct {
	selectionProviderFixture
	rateLimitCh chan time.Time
}

func (r *openAICodexExtraListRepo) SetRateLimited(_ context.Context, _ int64, at time.Time) error {
	if r.rateLimitCh != nil {
		r.rateLimitCh <- at
	}
	return nil
}

// hydrationProviderSource 保留原回源错误，不自行模拟补全成功或失败。
type hydrationProviderSource struct {
	scheduler.SnapshotProviderSource
	source Providers
}

func (s hydrationProviderSource) GetByID(ctx context.Context, id int64) (scheduler.SnapshotProvider, error) {
	value, err := s.source.GetByID(ctx, id)
	return codec.WrapRecord(gatewayadapter.ExecutionRecord(value)), err
}

func (r selectionProviderFixture) ListSchedulableByGroupIDAndPlatforms(ctx context.Context, groupID int64, platforms []string) ([]gatewayadapter.ExecutionProvider, error) {
	var result []gatewayadapter.ExecutionProvider
	for i := range r.providers {
		prepareSelectionFixtureProvider(ctx, &r.providers[i], &groupID)
		if slices.Contains(platforms, r.providers[i].Record.Platform) && openAIStickyProviderMatchesGroup(&r.providers[i], &groupID) {
			result = append(result, r.providers[i])
		}
	}
	return result, nil
}

func (r groupAwareStubOpenAIProviderRepo) ListSchedulableByGroupIDAndPlatforms(ctx context.Context, groupID int64, platforms []string) ([]gatewayadapter.ExecutionProvider, error) {
	var result []gatewayadapter.ExecutionProvider
	for i := range r.providers {
		if slices.Contains(platforms, r.providers[i].Record.Platform) && openAIStickyProviderMatchesGroup(&r.providers[i], &groupID) {
			prepareSelectionFixtureProvider(ctx, &r.providers[i], &groupID)
			result = append(result, r.providers[i])
		}
	}
	return result, nil
}

func (r groupAwareStubOpenAIProviderRepo) GetByID(ctx context.Context, id int64) (*gatewayadapter.ExecutionProvider, error) {
	for i := range r.providers {
		if r.providers[i].Record.ID == id {
			copy := r.providers[i]
			groups := copy.Record.GroupIDs
			prepareSelectionFixtureProvider(ctx, &copy, nil)
			copy.Record.GroupIDs = groups
			return &copy, nil
		}
	}
	return nil, errors.New("provider not found")
}
