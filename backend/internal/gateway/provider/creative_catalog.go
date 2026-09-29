package provider

import (
	"context"

	"github.com/TokenFlux/TokenRouter/internal/creative"
	creativeprovider "github.com/TokenFlux/TokenRouter/internal/creative/provider"
	"github.com/TokenFlux/TokenRouter/internal/provider"
)

// CreativeCatalogProvider 与任务执行共用提供商规则，保留透传、平台归一化及一跳映射语义。
func CreativeCatalogProvider(value *provider.Record) creative.CatalogProvider {
	if value == nil {
		return nil
	}
	return creativeCatalogProvider{CatalogProvider: creativeprovider.CatalogProvider(value), policy: ModelPolicy{Record: value}}
}

type creativeCatalogProvider struct {
	creative.CatalogProvider
	policy ModelPolicy
}

func (a creativeCatalogProvider) ResolveMappedModel(model string) (string, bool) {
	mapped := a.policy.UpstreamModel(context.Background(), model)
	return mapped, mapped != model
}

func (a creativeCatalogProvider) IsModelSupported(model string) bool {
	return a.policy.Supports(context.Background(), model)
}
