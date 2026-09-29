package selection

import (
	"context"

	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/scheduler"
)

// 原白盒合同直接观察平台核心的无凭据结果，不构造旧评分类型。
type openAIProviderLoadPlan struct{ candidates []scheduler.CandidateScore }

func (s *compatiblePicker) buildOpenAIProviderLoadPlan(ctx context.Context, req scheduler.PlatformSelectionInput, values []*gatewayprovider.ExecutionProvider, loads map[int64]*scheduler.ProviderLoadInfo) openAIProviderLoadPlan {
	core, scope := s.platformSelector()
	plan := core.BuildPlan(ctx, req, scope.pointers(values), loads)
	var candidates []scheduler.CandidateScore
	for _, value := range plan.CandidatesSnapshot() {
		candidates = append(candidates, scheduler.CandidateScore{Provider: &scheduler.ScoreProvider{ID: value.Provider.ID, Priority: value.Provider.Priority}, Score: value.Score})
	}
	return openAIProviderLoadPlan{candidates: candidates}
}
