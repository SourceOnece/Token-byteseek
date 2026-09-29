// 本文件只为提供商创建和编辑契约保存原生记录，不复制管理规则。
package provider

import (
	"context"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/google/uuid"
)

type qoderValidationStore struct {
	provider.AdminStore
	providers map[int64]*provider.Record
}

func (s *qoderValidationStore) Create(_ context.Context, value *provider.Record) error {
	if s.providers == nil {
		s.providers = make(map[int64]*provider.Record)
	}
	if value.ID == 0 {
		value.ID = int64(len(s.providers) + 1)
	}
	s.providers[value.ID] = provider.CloneRecord(value)
	return nil
}

func (s *qoderValidationStore) GetByID(_ context.Context, id int64) (*provider.Record, error) {
	return provider.CloneRecord(s.providers[id]), nil
}

func (s *qoderValidationStore) Update(_ context.Context, value *provider.Record) error {
	s.providers[value.ID] = provider.CloneRecord(value)
	return nil
}

func newQoderValidationAdmin(store *qoderValidationStore, validator *qoderCredentialValidator) *provider.Admin {
	return provider.NewAdmin(store, provider.AdminOptions{
		Creation:    provider.CreationOptions{Now: time.Now, LoadLocation: time.LoadLocation, NewSeed: uuid.NewString},
		Credentials: validator.hooks(),
	})
}
