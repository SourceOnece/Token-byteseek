package httpapi

import (
	"context"
	"time"

	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/accessview"
	"github.com/TokenFlux/TokenRouter/internal/scheduler/policy"

	"github.com/TokenFlux/TokenRouter/internal/provider"
)

func (s *managementListFixture) ListProviders(ctx context.Context, page, pageSize int, platform, providerType, status, search string, groupID int64, privacyMode string, sortBy, sortOrder string) ([]provider.Record, int64, error) {
	s.lastListProviders.platform = platform
	s.lastListProviders.providerType = providerType
	s.lastListProviders.status = status
	s.lastListProviders.search = search
	s.lastListProviders.groupID = groupID
	s.lastListProviders.privacyMode = privacyMode
	s.lastListProviders.sortBy = sortBy
	s.lastListProviders.sortOrder = sortOrder
	s.lastListProviders.calls++
	providers := s.providers
	total := len(providers)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = total
	}
	start := (page - 1) * pageSize
	if start >= total {
		return []provider.Record{}, int64(total), nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return providers[start:end], int64(total), nil
}

func (s *managementListFixture) ListProvidersForSchedulerScoreFilter(_ context.Context, platform, providerType, status, search string, groupID int64, privacyMode string) ([]provider.Record, error) {
	s.schedulerScoreFilterCalls++
	if s.providerSchedulerScoreFilterProviders != nil {
		return s.providerSchedulerScoreFilterProviders, nil
	}
	return s.providers, nil
}

func (s *managementListFixture) ListSchedulableProvidersForAdvancedSchedulerScore(_ context.Context, groupID *int64, platform string) ([]provider.Record, error) {
	s.openAISchedulerScorePoolCalls++
	if groupID != nil {
		s.openAISchedulerScorePoolGroupIDs = append(s.openAISchedulerScorePoolGroupIDs, *groupID)
	}
	providers := s.openAISchedulerScorePoolProviders
	if providers == nil {
		providers = s.providers
	}
	out := make([]provider.Record, 0, len(providers))
	for _, provider := range providers {
		if (platform != "" && provider.Platform != platform) || !provider.IsSchedulable() {
			continue
		}
		if groupID == nil {
			if len(provider.ProviderGroups) == 0 && len(provider.GroupIDs) == 0 {
				out = append(out, provider)
			}
			continue
		}
		for _, providerGroup := range provider.ProviderGroups {
			if providerGroup.GroupID == *groupID {
				out = append(out, provider)
				break
			}
		}
	}
	return out, nil
}

// managementListFixture 只拥有列表与评分候选读模型，保留原分页和调用计数。
type managementListFixture struct {
	ProviderManagement
	providers                                                               []provider.Record
	providerSchedulerScoreFilterProviders                                   []provider.Record
	openAISchedulerScorePoolProviders                                       []provider.Record
	schedulerScoreFilterCalls, openAISchedulerScorePoolCalls, getGroupCalls int
	openAISchedulerScorePoolGroupIDs                                        []int64
	lastListProviders                                                       struct {
		platform, providerType, status, search, privacyMode, sortBy, sortOrder string
		groupID                                                                int64
		calls                                                                  int
	}
}

func newManagementListFixture() *managementListFixture {
	now := time.Now().UTC()
	return &managementListFixture{providers: []provider.Record{{ID: 3, Name: "provider", Platform: provider.PlatformAnthropic, Type: provider.ProviderTypeOAuth, Status: provider.StatusActive, CreatedAt: now, UpdatedAt: now}}}
}

// newManagementListFixtureHandler 使用真实列表、评分和展示实现，静态缺省值保持原独立构造。
func newManagementListFixtureHandler(source *managementListFixture) *ManagementHandler {
	runtime := provider.NewRuntimeStatusReader(provider.RuntimeStatusOptions{})
	scores := provider.NewSchedulerScoreView(source, provideradapter.SchedulerScoreOptions(nil, nil, func(_ context.Context, group *accessview.GroupConfig) policy.EffectiveSettings {
		var overrides policy.GroupAdvancedSchedulerOverrides
		if group != nil && group.SchedulerType == "advanced" {
			overrides = group.AdvancedSchedulerOverrides
		}
		return policy.ResolveEffective(7, policy.ScoreWeights{Priority: 1, Load: 1, Queue: 0.7, ErrorRate: 0.8, TTFT: 0.5, Previous: 5, SessionSticky: 3}, policy.RuntimeSettings{}, overrides)
	}))
	presenter := NewRuntimePresenter(runtime, source, nil)
	return NewManagementHandler(source, ManagementOptions{List: provider.NewManagementList(source, runtime, scores, nil), RuntimePresenter: presenter})
}
