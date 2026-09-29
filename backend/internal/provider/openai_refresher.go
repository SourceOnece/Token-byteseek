// 提供商后台与请求刷新复用同一资格、合并和 PAT 清理规则，交换从提供商端口取得。
package provider

import (
	"context"
	"strings"
	"time"
)

type OpenAIRefreshTokenService interface {
	RefreshProviderToken(context.Context, *Record) (*OpenAITokenInfo, error)
}
type OpenAITokenRefresher struct{ Authorization OpenAIRefreshTokenService }

// CacheKey 返回用于分布式锁的缓存键
func (r *OpenAITokenRefresher) CacheKey(provider *Record) string {
	return OpenAITokenCacheKey(provider)
}

// CanRefresh 检查是否能处理此提供商
func (r *OpenAITokenRefresher) CanRefresh(provider *Record) bool {
	if provider.IsCredentialShadow() {
		return false
	}
	return provider.Platform == PlatformOpenAI && provider.Type == ProviderTypeOAuth
}

// NeedsRefresh 检查token是否需要刷新
// expires_at 缺失且处于限流状态时需要刷新，防止限流期间 token 静默过期
func (r *OpenAITokenRefresher) NeedsRefresh(provider *Record, refreshWindow time.Duration) bool {
	if provider.IsOpenAIPersonalAccessToken() {
		return false
	}
	if strings.TrimSpace(provider.GetOpenAIRefreshToken()) == "" {
		return false
	}
	expiresAt := provider.GetCredentialAsTime("expires_at")
	if expiresAt == nil {
		return provider.IsRateLimited()
	}

	return time.Until(*expiresAt) < refreshWindow
}

// Refresh 执行token刷新
// 保留原有credentials中的所有字段，只更新token相关字段
func (r *OpenAITokenRefresher) Refresh(ctx context.Context, provider *Record) (map[string]any, error) {
	tokenInfo, err := r.Authorization.RefreshProviderToken(ctx, provider)
	if err != nil {
		return nil, err
	}

	// 使用服务提供的方法构建新凭证，并保留原有字段
	newCredentials := BuildOpenAIProviderCredentials(tokenInfo)
	newCredentials = MergeCredentials(provider.Credentials, newCredentials)
	newCredentials = NormalizeOpenAIPersonalAccessTokenCredentials(provider, tokenInfo, newCredentials)

	return newCredentials, nil
}
