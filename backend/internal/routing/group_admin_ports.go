package routing

import (
	"context"

	"github.com/TokenFlux/TokenRouter/internal/scheduler/policy"
)

// GroupProvider 仅提供管理分组需要的提供商资格，不携带凭据。
type GroupProvider struct {
	ID       int64
	Platform string
	Type     string
}

type GroupProviders interface {
	GetByIDs(context.Context, []int64) ([]GroupProvider, error)
	ListSchedulableByGroupID(context.Context, int64) ([]CatalogueProvider, error)
}

type GroupKeyReader interface {
	ListKeysByGroupID(context.Context, int64) ([]string, error)
}
type GroupAdminInvalidator interface {
	GroupAuthInvalidator
	InvalidateAuthCacheByKey(context.Context, string)
}
type GroupPricingInvalidator interface{ InvalidateCache() }

// GroupAdminOptions 注入读取时机和原有闭合事务；规则不依赖装配和具体存储。
type GroupAdminOptions struct {
	ModelResolver RequestableResolver
	DefaultModels func(string) []string
	GlobalWeights func(context.Context) (policy.ScoreWeights, error)
	Mutate        func(context.Context, func(context.Context) error) error
}

// GroupAdmin 拥有分组管理和复制规则，缓存与读写均使用 app 提供的唯一实例。
type GroupAdmin struct {
	groupRepo                     GroupRepository
	groupDuplicateRepo            GroupDuplicateRepository
	groupSortOrderRepo            GroupSortOrderRepository
	providerRepo                  GroupProviders
	apiKeyRepo                    GroupKeyReader
	authCacheInvalidator          GroupAdminInvalidator
	pricingConfigCacheInvalidator GroupPricingInvalidator
	options                       GroupAdminOptions
}

func NewGroupAdmin(repo GroupRepository, duplicate GroupDuplicateRepository, sortOrder GroupSortOrderRepository, providers GroupProviders, keys GroupKeyReader, invalidator GroupAdminInvalidator, pricingConfigs GroupPricingInvalidator, options GroupAdminOptions) *GroupAdmin {
	return &GroupAdmin{groupRepo: repo, groupDuplicateRepo: duplicate, groupSortOrderRepo: sortOrder, providerRepo: providers, apiKeyRepo: keys, authCacheInvalidator: invalidator, pricingConfigCacheInvalidator: pricingConfigs, options: options}
}

// ValidateAdvancedOverrides 按旧次序先验证局部字段，确有权重覆盖才读取动态全局值。
func (s *GroupAdmin) ValidateAdvancedOverrides(ctx context.Context, overrides GroupAdvancedSchedulerOverrides) error {
	if err := policy.ValidateGroupOverrides(overrides); err != nil {
		return err
	}
	if !policy.HasWeightOverrides(overrides) {
		return nil
	}
	weights, err := s.options.GlobalWeights(ctx)
	if err != nil {
		return err
	}
	return policy.ValidateEffectiveWeights(policy.ApplyGroupWeightOverrides(weights, overrides))
}
