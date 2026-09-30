package routing

import (
	"context"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/modelcatalog"
	"github.com/stretchr/testify/require"
)

type attributeRepoFixture struct{ config *ModelAttributeConfig }

func (r *attributeRepoFixture) List(context.Context) ([]ModelAttributeConfig, error) {
	return []ModelAttributeConfig{*r.config}, nil
}

func (r *attributeRepoFixture) Get(context.Context, int64) (*ModelAttributeConfig, error) {
	return r.config, nil
}

func (r *attributeRepoFixture) ForGroup(context.Context, int64) (*ModelAttributeConfig, error) {
	return r.config, nil
}

func (r *attributeRepoFixture) Save(_ context.Context, value *ModelAttributeConfig) error {
	r.config = value
	return nil
}
func (r *attributeRepoFixture) Delete(context.Context, int64) error { r.config = nil; return nil }

func TestModelAttributesUseFinalModelsWithoutChangingRoutes(t *testing.T) {
	yes, no := true, false
	large, small := 100, 1
	config := &ModelAttributeConfig{Status: StatusActive, Rules: []ModelAttributeRule{
		{Models: []string{"upstream-*"}, Attributes: modelcatalog.Attributes{ToolCall: &no}},
		{Models: []string{"upstream-a"}, Attributes: modelcatalog.Attributes{OutputLimit: &small}},
	}}
	service := ModelAttributeService{Repo: &attributeRepoFixture{config}, Catalog: ModelAttributeCatalog{Lookup: func(string) modelcatalog.Attributes {
		return modelcatalog.Attributes{OutputLimit: &large, ToolCall: &yes}
	}}}
	models := []RequestableModel{{ID: "public-alias", PricingModel: "unrelated-price", UpstreamModels: []string{"upstream-a", "upstream-b"}}}
	before := models[0]
	result, err := service.ResolveModels(context.Background(), 1, models)
	require.NoError(t, err)
	require.Equal(t, before, models[0])
	require.Equal(t, 1, *result["public-alias"].OutputLimit)
	require.False(t, *result["public-alias"].ToolCall)
	require.True(t, result["public-alias"].RouteDifferences)
	// 精确规则只覆盖输出上限，不叠加同档案的通配规则。
	result, err = service.ResolveModels(context.Background(), 1, []RequestableModel{{ID: "single", UpstreamModels: []string{"upstream-a"}}})
	require.NoError(t, err)
	require.True(t, *result["single"].ToolCall)
	config.Status = StatusDisabled
	result, err = service.ResolveModels(context.Background(), 1, models)
	require.NoError(t, err)
	require.Equal(t, 100, *result["public-alias"].OutputLimit)
	require.True(t, *result["public-alias"].ToolCall)
}

func TestAttributeConfigValidation(t *testing.T) {
	service := ModelAttributeService{Repo: &attributeRepoFixture{}}
	zero := 0
	for _, config := range []ModelAttributeConfig{
		{Name: "x", Status: "other"},
		{Name: "x", Status: StatusActive, GroupIDs: []int64{1, 1}},
		{Name: "x", Status: StatusActive, Rules: []ModelAttributeRule{{Models: []string{"a*b"}}}},
		{Name: "x", Status: StatusActive, Rules: []ModelAttributeRule{{Models: []string{"a"}, Attributes: modelcatalog.Attributes{OutputLimit: &zero}}}},
	} {
		require.Error(t, service.Save(context.Background(), &config))
	}
	no := false
	valid := ModelAttributeConfig{Name: "metadata", Status: StatusActive, Rules: []ModelAttributeRule{{Models: []string{"*"}, Attributes: modelcatalog.Attributes{ToolCall: &no}}}}
	require.NoError(t, service.Save(context.Background(), &valid))
}
