package httpapi

import (
	"context"

	"github.com/TokenFlux/TokenRouter/internal/provider"
)

// 展示契约只提供列表和单条读取，保持原列表一次批量解析的断言。
type ollamaManagementFixture struct {
	ProviderManagement
	providers []provider.Record
}

func (f *ollamaManagementFixture) GetProvider(_ context.Context, id int64) (*provider.Record, error) {
	for _, value := range f.providers {
		if value.ID == id {
			return &value, nil
		}
	}
	return nil, provider.ErrProviderNotFound
}

func (f *ollamaManagementFixture) ListProviders(context.Context, int, int, string, string, string, string, int64, string, string, string) ([]provider.Record, int64, error) {
	return f.providers, int64(len(f.providers)), nil
}

func newOllamaManagementHandler(source *ollamaManagementFixture, usage *provider.OllamaCloudUsageService) *ManagementHandler {
	runtime := provider.NewRuntimeStatusReader(provider.RuntimeStatusOptions{})
	presenter := NewRuntimePresenter(runtime, source, usage)
	return NewManagementHandler(source, ManagementOptions{
		List:             provider.NewManagementList(source, runtime, nil, usage),
		RuntimePresenter: presenter,
		Presenter:        presenter,
		Ollama:           usage,
	})
}
