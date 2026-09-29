//go:build unit

package httpapi

import (
	"testing"

	providererrors "github.com/TokenFlux/TokenRouter/internal/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"

	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/stretchr/testify/require"
)

// TestProviderTestServiceSkipsShadow 验证影子提供商连接测试不再早拒,而是尝试解析母提供商凭据。
func TestProviderTestServiceSkipsShadow(t *testing.T) {
	pid := int64(100)
	shadow := &providererrors.Record{
		ID:               200,
		Platform:         capability.PlatformOpenAI,
		Type:             capability.ProviderTypeOAuth,
		ParentProviderID: &pid,
	}
	repo := &openAIProbeStore{openAIProbeRecords: openAIProbeRecords{providersByID: map[int64]*providererrors.Record{shadow.ID: shadow}}}
	svc := &provideradapter.OpenAIProviderTest{Store: repo}
	c, _ := newTestContext()

	err := executeOpenAIProbeRequest(t, svc, c, 200, "", "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "resolve spark shadow parent")
}
