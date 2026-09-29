package provider

import (
	"context"
	"strings"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
)

// UnauthorizedObservation 只含经供应商 Adapter 识别的认证失败信号。
type UnauthorizedObservation struct {
	Message                 string
	TokenRevoked            bool
	PermanentlyUnauthorized bool
}

// ApplyUnauthorized 保留凭据母提供商解析、令牌失效与刷新冷却的原次序。
func (s *HealthService) ApplyUnauthorized(ctx context.Context, provider *Record, observation UnauthorizedObservation) bool {
	// Spark 影子共用母提供商凭据。401 的缓存失效、refresh_token 检查、禁用和冷却均作用于母提供商，
	// 避免因影子没有 refresh_token 而误将其永久禁用。母提供商进入冷却后，调度健康检查会排除其影子。
	// 非影子直接使用自身记录；母提供商查找失败或不存在时，回退到当前记录。
	authProvider := provider
	if resolved, rerr := ResolveCredentialRecord(ctx, func(ctx context.Context, id int64) (*Record, error) { return s.providerRepo.GetByID(ctx, id) }, provider); rerr == nil && resolved != nil {
		authProvider = resolved
	}
	// OpenAI: token_invalidated / token_revoked 表示 token 被永久作废（非过期），直接标记 error
	if authProvider.Platform == capability.PlatformOpenAI && observation.TokenRevoked {
		msg := "Token revoked (401): provider authentication permanently revoked"
		if observation.Message != "" {
			msg = "Token revoked (401): " + observation.Message
		}
		s.ApplyAuthenticationFailure(ctx, authProvider, msg)
		return true
	}
	// OpenAI: {"detail":"Unauthorized"} 表示 token 完全无效（非标准 OpenAI 错误格式），直接标记 error
	if authProvider.Platform == capability.PlatformOpenAI && observation.PermanentlyUnauthorized {
		msg := "Unauthorized (401): provider authentication failed permanently"
		if observation.Message != "" {
			msg = "Unauthorized (401): " + observation.Message
		}
		s.ApplyAuthenticationFailure(ctx, authProvider, msg)
		return true
	}
	// OAuth 提供商在 401 错误时临时不可调度（给 token 刷新窗口）；非 OAuth 提供商保持原有 SetError 行为。
	if authProvider.Type == capability.ProviderTypeOAuth {
		// 1. 失效缓存
		if s.options.InvalidateUnauthorizedToken != nil {
			if err := s.options.InvalidateUnauthorizedToken(ctx, authProvider); err != nil {
				s.options.Warn("oauth_401_invalidate_cache_failed", "provider_id", authProvider.ID, "error", err)
			}
		}
		// 缺少 refresh_token 的 OAuth 提供商无法在冷却期内自愈（后台刷新服务也会跳过），
		// 直接走 SetError 永久禁用，避免冷却结束后再被选中产生一发无意义的 502。
		if strings.TrimSpace(authProvider.GetCredential("refresh_token")) == "" {
			msg := "Authentication failed (401): refresh_token missing, cannot recover"
			if observation.Message != "" {
				msg = "OAuth 401 (no refresh_token): " + observation.Message
			}
			s.ApplyAuthenticationFailure(ctx, authProvider, msg)
			return true
		}
		// 2. 临时不可调度，替代 SetError（保持 status=active 让刷新服务能拾取）
		// 注意：此处不再写回 provider.Credentials/expires_at。
		// 原实现使用请求开始时的 provider 快照整列覆盖 credentials JSONB（见
		// persistProviderCredentials → providerRepository.UpdateCredentials → SetCredentials），
		// 在另一个 worker 刚刷新完 refresh_token 的窄窗口内会把新 refresh_token 回滚为旧值，
		// 导致下一周期用旧 refresh_token 调上游拿到 invalid_grant 后，
		// tryRecoverFromRefreshRace 重读 DB 发现 currentRT == usedRT 也救不回来，提供商被错误 disable。
		// 这里仅依赖 InvalidateToken + SetTempUnschedulable 让提供商在冷却期内不被调度，
		// 冷却结束后由 token_provider 的 NeedsRefresh / token_refresh_service 走带分布式锁的正路刷新。
		msg := "Authentication failed (401): invalid or expired credentials"
		if observation.Message != "" {
			msg = "OAuth 401: " + observation.Message
		}
		if authProvider.Platform == capability.PlatformAntigravity {
			extraUpdates := AntigravityForceTokenRefreshExtra("401_invalid")
			if err := s.options.SessionWindows.UpdateExtra(ctx, authProvider.ID, extraUpdates); err != nil {
				s.options.Warn("antigravity_401_force_refresh_mark_failed", "provider_id", authProvider.ID, "error", err)
			} else {
				if authProvider.Extra == nil {
					authProvider.Extra = make(map[string]any, len(extraUpdates))
				}
				for k, v := range extraUpdates {
					authProvider.Extra[k] = v
				}
				s.options.Info("antigravity_401_force_refresh_marked", "provider_id", authProvider.ID)
			}
		}
		cooldownMinutes := s.options.UnauthorizedCooldownMinutes
		if cooldownMinutes <= 0 {
			cooldownMinutes = 10
		}
		until := s.options.Now().Add(time.Duration(cooldownMinutes) * time.Minute)
		s.notifyProviderSchedulingBlocked(authProvider, until, "oauth_401")
		if err := s.providerRepo.SetTempUnschedulable(ctx, authProvider.ID, until, msg); err != nil {
			s.options.Warn("oauth_401_set_temp_unschedulable_failed", "provider_id", authProvider.ID, "error", err)
		}
		return true
	} else {
		// 非 OAuth：保持 SetError 行为
		msg := "Authentication failed (401): invalid or expired credentials"
		if observation.Message != "" {
			msg = "Authentication failed (401): " + observation.Message
		}
		s.ApplyAuthenticationFailure(ctx, authProvider, msg)
		return true
	}
}
