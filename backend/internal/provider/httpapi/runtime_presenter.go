package httpapi

import (
	"context"

	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/provider/httpapi/dto"
)

// RuntimePresenter 只执行管理 DTO 与母提供商展示投影；查询和阈值由提供商用例处理。
type RuntimePresenter struct {
	status  *provider.RuntimeStatusReader
	parents interface {
		GetProvidersByIDs(context.Context, []int64) ([]*provider.Record, error)
	}
	ollama *provider.OllamaCloudUsageService
}

func NewRuntimePresenter(status *provider.RuntimeStatusReader, parents interface {
	GetProvidersByIDs(context.Context, []int64) ([]*provider.Record, error)
}, ollama *provider.OllamaCloudUsageService,
) *RuntimePresenter {
	return &RuntimePresenter{status, parents, ollama}
}

func (p *RuntimePresenter) Present(ctx context.Context, v *provider.Record) ProviderWithConcurrency {
	state := p.status.Read(ctx, v)
	item := p.Project(state)
	p.EnrichShadowParents(ctx, []ProviderWithConcurrency{item})
	return item
}

// EnrichShadowParentInfo 把母提供商的展示信息回填到影子行的 parent_* 字段。
// 纯函数：仅依赖传入的母提供商 map，便于单测；非影子或母提供商缺失时优雅留空。
func EnrichShadowParentInfo(items []ProviderWithConcurrency, parents map[int64]*provider.Record) {
	for i := range items {
		a := items[i].Provider
		if a == nil || a.ParentProviderID == nil {
			continue
		}
		p := parents[*a.ParentProviderID]
		if p == nil {
			continue
		}
		a.ParentEmail = p.GetCredential("email")
		a.ParentPlanType = p.GetCredential("plan_type")
		a.ParentSubscriptionExpiresAt = p.GetCredential("subscription_expires_at")
		a.ParentChatGPTAccountID = p.GetCredential("chatgpt_account_id")
		a.ParentPrivacyMode = p.GetExtraString("privacy_mode")
	}
}

// enrichShadowParents 收集本批影子行的母提供商 ID、一次批量解析（避免 N+1），再回填。
// 解析失败时不报错（parent_* 留空，降级）。
func (p *RuntimePresenter) EnrichShadowParents(ctx context.Context, items []ProviderWithConcurrency) {
	seen := make(map[int64]struct{})
	for i := range items {
		a := items[i].Provider
		if a == nil || a.ParentProviderID == nil {
			continue
		}
		seen[*a.ParentProviderID] = struct{}{}
	}
	if len(seen) == 0 {
		return
	}
	parentIDs := make([]int64, 0, len(seen))
	for pid := range seen {
		parentIDs = append(parentIDs, pid)
	}
	parents, err := p.parents.GetProvidersByIDs(ctx, parentIDs)
	if err != nil {
		return
	}
	pmap := make(map[int64]*provider.Record, len(parents))
	for _, p := range parents {
		pmap[p.ID] = p
	}
	EnrichShadowParentInfo(items, pmap)
}

// Project 只转换已取得的运行观察，不增加查询或改变列表批量语义。
func (p *RuntimePresenter) Project(state provider.RuntimeStatus) ProviderWithConcurrency {
	item := ProviderWithConcurrency{Provider: dto.ProviderFromRecord(state.Record), CurrentConcurrency: state.CurrentConcurrency, CurrentWindowCost: state.CurrentWindowCost, ActiveSessions: state.ActiveSessions, CurrentRPM: state.CurrentRPM, SchedulerScore: state.SchedulerScore, SchedulerScores: state.SchedulerScores}
	if item.Provider == nil {
		return item
	}
	if p.ollama != nil {
		p.ollama.EnrichState(item.OllamaCloudUsage)
	}
	return item
}
