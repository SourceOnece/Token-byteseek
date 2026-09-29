// Antigravity 刷新资格、凭据合并与 project 保留规则由提供商拥有。
package provider

import (
	"context"
	"strings"
	"time"
)

const AntigravityRefreshWindow = 15 * time.Minute

type AntigravityRefreshRules struct {
	RefreshProviderToken     func(context.Context, *Record) (*AntigravityTokenInfo, error)
	BuildProviderCredentials func(*AntigravityTokenInfo) map[string]any
	Printf, Logf             func(string, ...any)
}

// CacheKey 与原请求、后台刷新共用同一提供商锁键。
func (r *AntigravityRefreshRules) CacheKey(value *Record) string {
	return AntigravityTokenCacheKey(value)
}

// CanRefresh 检查是否可以刷新此提供商
func (r *AntigravityRefreshRules) CanRefresh(provider *Record) bool {
	return provider.Platform == PlatformAntigravity && provider.Type == ProviderTypeOAuth
}

// NeedsRefresh 检查提供商是否需要刷新
// Antigravity 使用固定的15分钟刷新窗口，忽略全局配置
func (r *AntigravityRefreshRules) NeedsRefresh(provider *Record, _ time.Duration) bool {
	if !r.CanRefresh(provider) {
		return false
	}
	if NeedsAntigravityRefreshRequest(provider) {
		return true
	}
	expiresAt := provider.GetCredentialAsTime("expires_at")
	if expiresAt == nil {
		return false
	}
	timeUntilExpiry := time.Until(*expiresAt)
	needsRefresh := timeUntilExpiry < AntigravityRefreshWindow
	if needsRefresh {
		r.Printf("[AntigravityTokenRefresher] Provider %d needs refresh: expires_at=%s, time_until_expiry=%v, window=%v\n",
			provider.ID, expiresAt.Format("2006-01-02 15:04:05"), timeUntilExpiry, AntigravityRefreshWindow)
	}
	return needsRefresh
}

// antigravityForceTokenRefreshExtra 构造强制刷新标记的持久化字段。
func AntigravityForceTokenRefreshExtra(reason string) map[string]any {
	return map[string]any{
		AntigravityForceTokenRefreshExtraKey:       true,
		AntigravityForceTokenRefreshReasonExtraKey: reason,
		AntigravityForceTokenRefreshAtExtraKey:     time.Now().UTC().Format(time.RFC3339),
	}
}

// Refresh 执行 token 刷新
func (r *AntigravityRefreshRules) Refresh(ctx context.Context, provider *Record) (map[string]any, error) {
	tokenInfo, err := r.RefreshProviderToken(ctx, provider)
	if err != nil {
		return nil, err
	}

	newCredentials := r.BuildProviderCredentials(tokenInfo)
	// 合并旧的 credentials，保留新 credentials 中不存在的字段
	newCredentials = MergeCredentials(provider.Credentials, newCredentials)

	// 特殊处理 project_id：如果新值为空但旧值非空，保留旧值
	// 这确保了即使 LoadCodeAssist 失败，project_id 也不会丢失
	if newProjectID, _ := newCredentials["project_id"].(string); newProjectID == "" {
		if oldProjectID := strings.TrimSpace(provider.GetCredential("project_id")); oldProjectID != "" {
			newCredentials["project_id"] = oldProjectID
		}
	}

	// 如果 project_id 获取失败，只记录警告，不返回错误
	// LoadCodeAssist 失败可能是临时网络问题，应该允许重试而不是立即标记为不可重试错误
	// Token 刷新本身是成功的（access_token 和 refresh_token 已更新）
	if tokenInfo.ProjectIDMissing {
		if tokenInfo.ProjectID != "" {
			// 有旧的 project_id，本次获取失败，保留旧值
			r.Logf("[AntigravityTokenRefresher] Provider %d: LoadCodeAssist 临时失败，保留旧 project_id", provider.ID)
		} else {
			// 从未获取过 project_id，本次也失败，但不返回错误以允许下次重试
			r.Logf("[AntigravityTokenRefresher] Provider %d: LoadCodeAssist 失败，project_id 缺失，但 token 已更新，将在下次刷新时重试", provider.ID)
		}
	}

	return newCredentials, nil
}
