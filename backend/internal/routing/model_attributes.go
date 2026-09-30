package routing

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/TokenFlux/TokenRouter/internal/modelcatalog"
	"github.com/TokenFlux/TokenRouter/internal/pkg/apperror"
)

var (
	ErrAttributeConfigNotFound = apperror.NotFound("ATTRIBUTE_CONFIG_NOT_FOUND", "attribute configuration not found")
	ErrAttributeConfigConflict = apperror.Conflict("ATTRIBUTE_CONFIG_CONFLICT", "configuration name or group association already exists")
)

// ModelAttributeRule 以最终上游模型匹配展示属性，不参与可请求性计算。
type ModelAttributeRule struct {
	Models     []string                `json:"models"`
	Attributes modelcatalog.Attributes `json:"attributes"`
}

// ModelAttributeConfig 是可以被多个分组共享的属性档案。
type ModelAttributeConfig struct {
	ID          int64                `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Status      string               `json:"status"`
	GroupIDs    []int64              `json:"group_ids"`
	Rules       []ModelAttributeRule `json:"rules"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
}

// ModelAttributeRepository 在同一事务中保存规则与分组关联。
type ModelAttributeRepository interface {
	List(context.Context) ([]ModelAttributeConfig, error)
	Get(context.Context, int64) (*ModelAttributeConfig, error)
	ForGroup(context.Context, int64) (*ModelAttributeConfig, error)
	Save(context.Context, *ModelAttributeConfig) error
	Delete(context.Context, int64) error
}

// ModelAttributeCatalog 是管理员默认查询的只读端口。
type ModelAttributeCatalog struct {
	Snapshot   func() ModelAttributeSnapshot
	Lookup     func(string) modelcatalog.Attributes
	Update     func() error
	Candidates func() func(string) []string
}

type ModelAttributeSnapshot struct {
	Items       []modelcatalog.Entry `json:"items"`
	Version     string               `json:"version"`
	LastUpdated time.Time            `json:"last_updated"`
	LastError   string               `json:"last_error,omitempty"`
}

// EffectiveModelAttributes 保留多路差异，消费者不得据此过滤模型或协议。
type EffectiveModelAttributes = modelcatalog.Presentation

// ModelAttributeService 不保存跨请求配置缓存，所有实例直接读取提交后的数据库状态。
// @project-doc docs/interfaces/model_catalog_and_marketplace.md#model_attributes
type ModelAttributeService struct {
	Repo        ModelAttributeRepository
	Catalog     ModelAttributeCatalog
	Invalidator GroupAuthInvalidator
}

func (s *ModelAttributeService) Save(ctx context.Context, config *ModelAttributeConfig) error {
	config.Name = strings.TrimSpace(config.Name)
	if config.Name == "" || utf8.RuneCountInString(config.Name) > 100 || config.Status != StatusActive && config.Status != StatusDisabled {
		return apperror.BadRequest("INVALID_ATTRIBUTE_CONFIG", "name and active/disabled status are required")
	}
	seenModels := map[string]bool{}
	for i := range config.Rules {
		rule := &config.Rules[i]
		if len(rule.Models) == 0 {
			return apperror.BadRequest("INVALID_ATTRIBUTE_RULE", "each rule requires a model")
		}
		for j, name := range rule.Models {
			name = strings.TrimSpace(name)
			if name == "" || strings.Contains(strings.TrimSuffix(name, "*"), "*") || seenModels[strings.ToLower(name)] {
				return apperror.BadRequest("INVALID_ATTRIBUTE_RULE", "model patterns must be nonempty and unique, with an optional trailing wildcard")
			}
			seenModels[strings.ToLower(name)] = true
			rule.Models[j] = name
		}
		if err := rule.Attributes.Validate(); err != nil {
			return apperror.BadRequest("INVALID_MODEL_ATTRIBUTES", err.Error())
		}
	}
	seenGroups := map[int64]bool{}
	for _, id := range config.GroupIDs {
		if id <= 0 || seenGroups[id] {
			return apperror.BadRequest("INVALID_ATTRIBUTE_GROUPS", "group IDs must be positive and unique")
		}
		seenGroups[id] = true
	}
	var oldGroups []int64
	if config.ID != 0 {
		old, err := s.Repo.Get(ctx, config.ID)
		if err != nil {
			return err
		}
		oldGroups = old.GroupIDs
	}
	if config.Rules == nil {
		config.Rules = []ModelAttributeRule{}
	}
	if config.GroupIDs == nil {
		config.GroupIDs = []int64{}
	}
	if err := s.Repo.Save(ctx, config); err != nil {
		return err
	}
	s.invalidate(ctx, append(oldGroups, config.GroupIDs...))
	return nil
}

func (s *ModelAttributeService) Delete(ctx context.Context, id int64) error {
	old, err := s.Repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.Repo.Delete(ctx, id); err != nil {
		return err
	}
	s.invalidate(ctx, old.GroupIDs)
	return nil
}

func (s *ModelAttributeService) invalidate(ctx context.Context, groups []int64) {
	if s.Invalidator == nil {
		return
	}
	seen := map[int64]bool{}
	for _, id := range groups {
		if !seen[id] {
			s.Invalidator.InvalidateAuthCacheByGroupID(ctx, id)
			seen[id] = true
		}
	}
}

// ResolveModels 每个分组只读一次档案，以可请求结果中的最终模型生成展示投影。
func (s *ModelAttributeService) ResolveModels(ctx context.Context, groupID int64, models []RequestableModel) (map[string]EffectiveModelAttributes, error) {
	config, err := s.Repo.ForGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}
	var candidates func(string) []string
	if s.Catalog.Candidates != nil {
		candidates = s.Catalog.Candidates()
	}
	result := map[string]EffectiveModelAttributes{}
	for _, model := range models {
		names := model.UpstreamModels
		if len(names) == 0 {
			names = []string{model.ID}
		}
		values := make([]modelcatalog.Attributes, 0, len(names))
		for _, name := range names {
			base := s.Catalog.Lookup(name)
			if config != nil && config.Status == StatusActive {
				base = modelcatalog.Merge(base, config.match(name, candidates))
			}
			values = append(values, base)
		}
		attrs, different := modelcatalog.Common(values)
		result[model.ID] = EffectiveModelAttributes{Attributes: attrs, RouteDifferences: different}
	}
	return result, nil
}

func (c *ModelAttributeConfig) match(model string, expand func(string) []string) modelcatalog.Attributes {
	names := []string{strings.ToLower(strings.TrimSpace(model))}
	if expand != nil {
		names = append(names, expand(model)...)
	}
	for _, name := range names {
		for _, rule := range c.Rules {
			for _, pattern := range rule.Models {
				if strings.EqualFold(pattern, name) {
					return rule.Attributes
				}
			}
		}
	}
	for _, rule := range c.Rules {
		for _, pattern := range rule.Models {
			if !strings.HasSuffix(pattern, "*") {
				continue
			}
			for _, name := range names {
				if strings.HasPrefix(strings.ToLower(name), strings.ToLower(strings.TrimSuffix(pattern, "*"))) {
					return rule.Attributes
				}
			}
		}
	}
	return modelcatalog.Attributes{}
}
