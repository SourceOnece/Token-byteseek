package provider

import (
	"context"
	"fmt"
	"strings"

	infraerrors "github.com/TokenFlux/TokenRouter/internal/pkg/apperror"
	"github.com/TokenFlux/TokenRouter/internal/pkg/pagination"
)

// Record management implementations
func (s *Admin) ListProviders(ctx context.Context, page, pageSize int, platform, providerType, status, search string, groupID int64, privacyMode string, sortBy, sortOrder string) ([]Record, int64, error) {
	params := pagination.PaginationParams{Page: page, PageSize: pageSize, SortBy: sortBy, SortOrder: sortOrder}
	providers, result, err := s.providerRepo.ListWithFilters(ctx, params, platform, providerType, status, search, groupID, privacyMode)
	if err != nil {
		return nil, 0, err
	}
	return providers, result.Total, nil
}

// ListProvidersForSchedulerScoreFilter 查询当前管理端筛选范围内用于调度评分的提供商。
func (s *Admin) ListProvidersForSchedulerScoreFilter(ctx context.Context, platform, providerType, status, search string, groupID int64, privacyMode string) ([]Record, error) {
	if s == nil || s.providerRepo == nil {
		return nil, nil
	}
	return s.providerRepo.ListAllWithFilters(ctx, platform, providerType, status, search, groupID, privacyMode)
}

// ListSchedulableProvidersForAdvancedSchedulerScore 查询指定分组内可参与高级评分的提供商。
func (s *Admin) ListSchedulableProvidersForAdvancedSchedulerScore(ctx context.Context, groupID *int64, platform string) ([]Record, error) {
	if s == nil || s.providerRepo == nil || groupID == nil || *groupID <= 0 {
		return nil, nil
	}
	platform = strings.TrimSpace(platform)
	if platform != "" {
		return s.providerRepo.ListSchedulableByGroupIDAndPlatform(ctx, *groupID, platform)
	}
	return s.providerRepo.ListSchedulableByGroupIDAndPlatforms(ctx, *groupID, []string{PlatformAnthropic, PlatformOpenAI, PlatformGemini, PlatformAntigravity, PlatformGrok, PlatformQoder, PlatformKimi, PlatformZhipu, PlatformDeepseek})
}

func (s *Admin) GetProvider(ctx context.Context, id int64) (*Record, error) {
	return s.providerRepo.GetByID(ctx, id)
}

func (s *Admin) GetProvidersByIDs(ctx context.Context, ids []int64) ([]*Record, error) {
	if len(ids) == 0 {
		return []*Record{}, nil
	}

	providers, err := s.providerRepo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to get providers by IDs: %w", err)
	}

	return providers, nil
}

func (s *Admin) DeleteProvider(ctx context.Context, id int64) error {
	// 级联删除 spark 影子提供商（先删影子，再删母提供商）
	shadows, err := s.providerRepo.ListShadowsByParent(ctx, id)
	if err != nil {
		return fmt.Errorf("list spark shadows for cascade delete: %w", err)
	}
	for _, shadow := range shadows {
		if err := s.providerRepo.Delete(ctx, shadow.ID); err != nil {
			return fmt.Errorf("cascade delete spark shadow %d: %w", shadow.ID, err)
		}
	}
	if err := s.providerRepo.Delete(ctx, id); err != nil {
		return err
	}
	return nil
}

func (s *Admin) ClearProviderError(ctx context.Context, id int64) (*Record, error) {
	if err := s.providerRepo.ClearError(ctx, id); err != nil {
		return nil, err
	}
	if err := s.providerRepo.ClearRateLimit(ctx, id); err != nil {
		return nil, err
	}
	if err := s.providerRepo.ClearAntigravityQuotaScopes(ctx, id); err != nil {
		return nil, err
	}
	if err := s.providerRepo.ClearModelRateLimits(ctx, id); err != nil {
		return nil, err
	}
	if err := s.providerRepo.ClearTempUnschedulable(ctx, id); err != nil {
		return nil, err
	}
	if s.options.RuntimeBlocker != nil {
		s.options.RuntimeBlocker.ClearProviderSchedulingBlock(id)
	}
	return s.providerRepo.GetByID(ctx, id)
}

func (s *Admin) SetProviderSchedulable(ctx context.Context, id int64, schedulable bool) (*Record, error) {
	if err := s.providerRepo.SetSchedulable(ctx, id, schedulable); err != nil {
		return nil, err
	}
	updated, err := s.providerRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *Admin) ResetProviderQuota(ctx context.Context, id int64) error {
	provider, err := s.providerRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	// spark 影子提供商不持自有配额(凭据透传母提供商、spark 用量走独立 codex_* 维度由 QueryUsage 维护),
	// 通用 quota 重置对其无意义且语义不一致——明确 400 拒绝(与 OpenAI reset-credit 对影子一致)。
	if provider.IsCredentialShadow() {
		return infraerrors.New(infraerrors.CategoryBadRequest, "SPARK_SHADOW_NO_QUOTA_RESET",
			"cannot reset quota for a spark shadow provider; manage it on the parent provider")
	}
	return s.options.Quotas.ResetQuotaUsedAndClearRateLimitCooldown(ctx, id)
}
