package provider_test

import (
	"testing"

	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/stretchr/testify/require"
)

func TestProviderSparkShadowHelpers(t *testing.T) {
	pid := int64(100)
	normal := &providercore.Record{ID: 100}
	require.False(t, normal.IsShadow())
	require.False(t, normal.IsCredentialShadow())
	require.Equal(t, providercore.QuotaDimensionGlobal, normal.QuotaDimensionOrDefault())
	shadow := &providercore.Record{ID: 200, ParentProviderID: &pid, QuotaDimension: providercore.QuotaDimensionSpark}
	require.True(t, shadow.IsShadow())
	require.True(t, shadow.IsCredentialShadow())
	require.Equal(t, providercore.QuotaDimensionSpark, shadow.QuotaDimensionOrDefault())
}
