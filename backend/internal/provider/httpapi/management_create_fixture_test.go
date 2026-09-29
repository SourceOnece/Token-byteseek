package httpapi

import (
	"context"
	"sync"

	"github.com/TokenFlux/TokenRouter/internal/provider"
)

// 创建契约仅记录提交给用例的输入，其它能力不提供默认业务实现。
type managementCreateFixture struct {
	ProviderManagement
	mu                sync.Mutex
	createdProviders  []*provider.CreateProviderInput
	createProviderErr error
}

func (s *managementCreateFixture) CreateProvider(_ context.Context, input *provider.CreateProviderInput) (*provider.Record, error) {
	s.mu.Lock()
	s.createdProviders = append(s.createdProviders, input)
	s.mu.Unlock()
	if s.createProviderErr != nil {
		return nil, s.createProviderErr
	}
	return &provider.Record{ID: 300, Name: input.Name, Status: provider.StatusActive}, nil
}

func (s *managementCreateFixture) ForceOpenAIPrivacy(context.Context, *provider.Record) string {
	return ""
}

func (s *managementCreateFixture) ForceAntigravityPrivacy(context.Context, *provider.Record) string {
	return ""
}

// 测试使用同一原生 HTTP、批处理和展示实现，不构造旧管理员聚合。
func newManagementCreateFixtureHandler(source *managementCreateFixture) *ManagementHandler {
	presenter := NewRuntimePresenter(provider.NewRuntimeStatusReader(provider.RuntimeStatusOptions{}), source, nil)
	batch := provider.NewManagementBatch(source, nil, provider.ManagementCreationOptions{Privacy: source})
	return NewManagementHandler(source, ManagementOptions{Presenter: presenter, Privacy: source, Batch: batch})
}
