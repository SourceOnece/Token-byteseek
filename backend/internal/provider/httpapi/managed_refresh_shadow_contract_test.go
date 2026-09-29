package httpapi

import (
	"context"
	"net/http"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"

	"github.com/TokenFlux/TokenRouter/internal/server/httpx"
	"github.com/stretchr/testify/require"
)

// TestRefreshSingleProvider_RejectsShadow 验证手动刷新对 spark 影子在调用上游前早拒
// (影子凭据由母提供商管理、自身恒空,刷新无意义)。该守卫同时覆盖单提供商与批量刷新两入口。
func TestRefreshSingleProvider_RejectsShadow(t *testing.T) {
	h := provider.NewManagedRefreshService(provider.ManagedRefreshOptions{}) // 影子在使用任何依赖前即返回,无需注入
	parentID := int64(5)
	shadow := &provider.Record{
		ID:               9,
		Platform:         capability.PlatformOpenAI,
		Type:             capability.ProviderTypeOAuth, // IsOAuth()=true,确保不是先撞 NOT_OAUTH
		ParentProviderID: &parentID,
		QuotaDimension:   provider.QuotaDimensionSpark,
	}

	_, _, err := h.Refresh(context.Background(), shadow)
	require.Error(t, err, "影子刷新应被早拒")
	require.Equal(t, http.StatusBadRequest, httpx.ErrorCode(err))
}
