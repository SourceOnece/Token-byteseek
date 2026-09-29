// OpenAI 额度查询、影子资格、重置次数与缓存维护由提供商拥有，供应商访问通过窄端口。
package provider

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	infraerrors "github.com/TokenFlux/TokenRouter/internal/pkg/apperror"
	wire "github.com/TokenFlux/TokenRouter/internal/protocol/openai"
)

var (
	ErrSparkShadowResetNotSupported = infraerrors.New(409, "SPARK_SHADOW_RESET_NOT_SUPPORTED", "spark shadow provider does not support credit reset; reset the parent provider")
	ErrOpenAIQuotaStopped           = infraerrors.New(503, "OPENAI_QUOTA_STOPPED", "openai quota service is stopped")
)

const (
	chatGPTUsagePath                   = "/wham/usage"
	chatGPTRateLimitCreditsPath        = "/wham/rate-limit-reset-credits"
	chatGPTRateLimitResetPath          = "/wham/rate-limit-reset-credits/consume"
	codexInviteResetSupportsRewardless = "true"
	openaiQuotaResetCreditsKey         = "codex_reset_credit_snapshot"
)

type OpenAIQuotaClient interface {
	GetJSON(context.Context, string, map[string]string) (map[string]any, error)
	GetJSONRaw(context.Context, string, map[string]string) ([]byte, error)
	PostJSON(context.Context, string, map[string]any) (map[string]any, error)
}
type PreparedOpenAIQuota struct {
	Provider *Record           `json:"-"`
	Token    string            `json:"-"`
	Client   OpenAIQuotaClient `json:"-"`
}

func (p PreparedOpenAIQuota) String() string { return "prepared OpenAI quota request" }

type OpenAIQuotaOptions struct {
	Configured func() bool
	Read       func(context.Context, int64) (*Record, error)
	Token      func(context.Context, *Record) (string, error)
	Client     func(context.Context, *Record, string) (OpenAIQuotaClient, error)
	SaveExtra  func(context.Context, int64, map[string]any) error
	RedeemID   func() (string, error)
	Warn, Info func(string, ...any)
}
type OpenAIQuotaService struct {
	Options  OpenAIQuotaOptions
	activity operationActivity
}

func NewOpenAIQuotaService(options OpenAIQuotaOptions) *OpenAIQuotaService {
	return &OpenAIQuotaService{Options: options}
}

func remarshalOpenAIQuotaPayload(raw map[string]any, target any) error {
	body, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}

// QueryUsage 查询提供商当前上游限流窗口和可用重置次数。
func (s *OpenAIQuotaService) QueryUsage(ctx context.Context, providerID int64) (*wire.OpenAIQuotaUsage, error) {
	ctx, done, activityErr := s.activity.begin(ctx, ErrOpenAIQuotaStopped)
	if activityErr != nil {
		return nil, activityErr
	}
	defer done()

	providerCtx, err := s.PrepareProvider(ctx, providerID)
	if err != nil {
		return nil, err
	}
	raw, err := providerCtx.Client.GetJSON(ctx, chatGPTUsagePath, map[string]string{
		"supports_rewardless_invites": codexInviteResetSupportsRewardless,
	})
	if err != nil {
		return nil, err
	}

	var usage wire.OpenAIQuotaUsage
	if err := remarshalOpenAIQuotaPayload(raw, &usage); err != nil {
		return nil, err
	}
	usage.FetchedAt = time.Now().Unix()
	details := s.queryResetCreditDetails(ctx, providerCtx)
	if details != nil {
		hasDetailCount := details.AvailableCount != nil
		if usage.RateLimitResetCredits == nil {
			usage.RateLimitResetCredits = &wire.OpenAIRateLimitResetCredits{}
		}
		if details.CreditListPresent {
			usage.RateLimitResetCredits.Credits = details.Credits
		}
		switch {
		case hasDetailCount:
			usage.RateLimitResetCredits.AvailableCount = *details.AvailableCount
		case details.CreditListPresent:
			usage.RateLimitResetCredits.AvailableCount = details.AvailableCreditCount
		}
	}
	return &usage, nil
}

// CacheResetCreditsSnapshot 保存显式查询得到的完整重置次数快照。
// 正数次数必须附带到期明细，否则保留旧缓存，避免前端长期展示无法自然失效的次数。
func (s *OpenAIQuotaService) CacheResetCreditsSnapshot(ctx context.Context, providerID int64, credits *wire.OpenAIRateLimitResetCredits) error {
	ctx, done, activityErr := s.activity.begin(ctx, ErrOpenAIQuotaStopped)
	if activityErr != nil {
		return activityErr
	}
	defer done()

	return s.cacheResetCreditsSnapshot(ctx, providerID, credits, nil)
}

// CachePostResetSnapshot 保存重置后观察到的 credits 与用量窗口。
func (s *OpenAIQuotaService) CachePostResetSnapshot(ctx context.Context, providerID int64, usage *wire.OpenAIQuotaUsage) error {
	ctx, done, activityErr := s.activity.begin(ctx, ErrOpenAIQuotaStopped)
	if activityErr != nil {
		return activityErr
	}
	defer done()

	if usage == nil {
		return s.cacheResetCreditsSnapshot(ctx, providerID, nil, nil)
	}
	return s.cacheResetCreditsSnapshot(
		ctx,
		providerID,
		usage.RateLimitResetCredits,
		BuildOpenAIAutoResetUsageUpdates(usage, time.Now()),
	)
}

// BuildOpenAIAutoResetUsageUpdates 将重置后 5 小时/7 天窗口写入提供商缓存，
// 使下一次列表查询直接展示新配额而无需再次访问上游。
func BuildOpenAIAutoResetUsageUpdates(usage *wire.OpenAIQuotaUsage, now time.Time) map[string]any {
	if usage == nil || usage.RateLimit == nil {
		return nil
	}
	snapshot := &wire.OpenAICodexUsageSnapshot{UpdatedAt: now.UTC().Format(time.RFC3339)}
	applyWindow := func(window *wire.OpenAIRateLimitWindow, primary bool) {
		if window == nil {
			return
		}
		used := window.UsedPercent
		resetAfter := int(window.ResetAfterSeconds)
		windowMinutes := int(window.LimitWindowSeconds / 60)
		if primary {
			snapshot.PrimaryUsedPercent = &used
			snapshot.PrimaryResetAfterSeconds = &resetAfter
			snapshot.PrimaryWindowMinutes = &windowMinutes
		} else {
			snapshot.SecondaryUsedPercent = &used
			snapshot.SecondaryResetAfterSeconds = &resetAfter
			snapshot.SecondaryWindowMinutes = &windowMinutes
		}
	}
	applyWindow(usage.RateLimit.PrimaryWindow, true)
	applyWindow(usage.RateLimit.SecondaryWindow, false)
	return BuildCodexUsageExtraUpdates(snapshot, now)
}

func (s *OpenAIQuotaService) cacheResetCreditsSnapshot(ctx context.Context, providerID int64, credits *wire.OpenAIRateLimitResetCredits, updates map[string]any) error {
	if credits == nil || (credits.AvailableCount > 0 && len(credits.Credits) == 0) {
		return infraerrors.New(
			502,
			"OPENAI_QUOTA_RESET_CREDITS_REFRESH_FAILED",
			"failed to refresh reset-credit expiration details; cached data was preserved",
		)
	}
	if s == nil || s.Options.SaveExtra == nil {
		return infraerrors.InternalServer("OPENAI_QUOTA_NOT_CONFIGURED", "openai quota cache repository is not configured")
	}
	if updates == nil {
		updates = make(map[string]any, 1)
	}
	updates[openaiQuotaResetCreditsKey] = credits
	if err := s.Options.SaveExtra(ctx, providerID, updates); err != nil {
		return infraerrors.New(
			500,
			"OPENAI_QUOTA_CACHE_WRITE_FAILED",
			"failed to cache reset-credit details",
		).WithCause(err)
	}
	return nil
}

func (s *OpenAIQuotaService) queryResetCreditDetails(ctx context.Context, providerCtx *PreparedOpenAIQuota) *wire.OpenAIRateLimitResetCreditDetails {
	raw, err := providerCtx.Client.GetJSONRaw(ctx, chatGPTRateLimitCreditsPath, nil)
	if err != nil {
		s.Options.Warn("openai_quota_reset_credit_details_failed", "provider_id", providerCtx.Provider.ID, "error", err)
		return nil
	}
	details, err := wire.ParseOpenAIRateLimitResetCreditDetails(raw)
	if err != nil {
		s.Options.Warn("openai_quota_reset_credit_details_parse_failed", "provider_id", providerCtx.Provider.ID, "error", err)
		// 列表解析失败时只接受独立有效的可用次数，列表本身仍保持 fail-closed。
		if details.AvailableCount == nil {
			return nil
		}
	}
	if details.AvailableCount == nil && !details.CreditListPresent {
		return nil
	}
	return &details
}

// ResetCredit 消耗一次限流窗口重置次数。
func (s *OpenAIQuotaService) ResetCredit(ctx context.Context, providerID int64) (*wire.OpenAIQuotaResetResult, error) {
	ctx, done, activityErr := s.activity.begin(ctx, ErrOpenAIQuotaStopped)
	if activityErr != nil {
		return nil, activityErr
	}
	defer done()

	provider, err := s.LoadProvider(ctx, providerID)
	if err != nil {
		return nil, err
	}
	// 影子提供商共享母提供商凭据和额度，重置次数必须显式在母提供商上操作，避免误把影子操作扩散到全局额度。
	if provider.IsCredentialShadow() {
		return nil, ErrSparkShadowResetNotSupported
	}
	providerCtx, err := s.PrepareProvider(ctx, providerID)
	if err != nil {
		return nil, err
	}
	redeemRequestID, err := s.Options.RedeemID()
	if err != nil {
		return nil, infraerrors.Newf(500, "OPENAI_QUOTA_REDEEM_ID_FAILED", "failed to generate redeem id: %v", err)
	}

	raw, err := providerCtx.Client.PostJSON(ctx, chatGPTRateLimitResetPath, map[string]any{
		"redeem_request_id": redeemRequestID,
	})
	if err != nil {
		return nil, err
	}

	var result wire.OpenAIQuotaResetResult
	if err := remarshalOpenAIQuotaPayload(raw, &result); err != nil {
		return nil, err
	}
	s.Options.Info("openai_quota_reset_success",
		"provider_id", providerID,
		"code", result.Code,
		"windows_reset", result.WindowsReset,
	)
	return &result, nil
}

func (s *OpenAIQuotaService) PrepareProvider(ctx context.Context, providerID int64) (*PreparedOpenAIQuota, error) {
	if s == nil || s.Options.Configured == nil || !s.Options.Configured() {
		return nil, infraerrors.InternalServer("OPENAI_QUOTA_NOT_CONFIGURED", "openai quota service is not configured")
	}
	provider, err := s.LoadProvider(ctx, providerID)
	if err != nil {
		return nil, err
	}
	if !provider.IsOpenAIOAuth() {
		return nil, infraerrors.BadRequest("OPENAI_QUOTA_UNSUPPORTED_PROVIDER", "only OpenAI OAuth providers support quota reset")
	}

	if provider.IsCredentialShadow() {
		parent, resolveErr := s.LoadProvider(ctx, *provider.ParentProviderID)
		if resolveErr != nil {
			return nil, infraerrors.Newf(502, "OPENAI_QUOTA_SHADOW_RESOLVE_FAILED", "failed to resolve shadow provider: %v", resolveErr)
		}
		if parent.IsCredentialShadow() {
			return nil, infraerrors.Newf(502, "OPENAI_QUOTA_SHADOW_RESOLVE_FAILED", "spark shadow parent %d is itself a shadow", parent.ID)
		}
		if !parent.IsOpenAIOAuth() {
			return nil, infraerrors.Newf(502, "OPENAI_QUOTA_SHADOW_RESOLVE_FAILED", "spark shadow parent %d is not OpenAI OAuth", parent.ID)
		}
		provider = parent
	}

	if strings.TrimSpace(provider.GetChatGPTAccountID()) == "" && strings.TrimSpace(provider.GetCredential("organization_id")) == "" {
		return nil, infraerrors.BadRequest("OPENAI_QUOTA_MISSING_PROVIDER_ID", "chatgpt_account_id is missing; please re-authorize this provider")
	}

	token := ""
	if !provider.IsOpenAIAgentIdentity() && s.Options.Token != nil {
		token, err = s.Options.Token(ctx, provider)
		if err != nil {
			return nil, err
		}
	}
	if !provider.IsOpenAIAgentIdentity() && strings.TrimSpace(token) == "" {
		token = provider.GetOpenAIAccessToken()
	}
	if !provider.IsOpenAIAgentIdentity() && strings.TrimSpace(token) == "" {
		return nil, infraerrors.BadRequest("OPENAI_QUOTA_MISSING_TOKEN", "missing OpenAI OAuth access token")
	}

	client, err := s.Options.Client(ctx, provider, token)
	if err != nil {
		return nil, err
	}
	return &PreparedOpenAIQuota{Provider: provider, Token: token, Client: client}, nil
}

func (s *OpenAIQuotaService) LoadProvider(ctx context.Context, providerID int64) (*Record, error) {
	if s == nil || s.Options.Read == nil {
		return nil, infraerrors.InternalServer("OPENAI_QUOTA_NOT_CONFIGURED", "openai quota service is not configured")
	}
	provider, err := s.Options.Read(ctx, providerID)
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, infraerrors.NotFound("PROVIDER_NOT_FOUND", "provider not found")
	}
	return provider, nil
}

// BuildCodexSparkWindowExtraUpdates 从 /wham/usage 的 additional_rate_limits 中提取 Codex Spark 窗口。
// 返回的 key 复用普通 codex_* 命名，影子提供商自己的 Extra 可直接被调度层和前端读取。
func BuildCodexSparkWindowExtraUpdates(usage *wire.OpenAIQuotaUsage, now time.Time) map[string]any {
	if usage == nil {
		return nil
	}
	var spark *wire.OpenAIRateLimit
	for i := range usage.AdditionalRateLimits {
		a := usage.AdditionalRateLimits[i]
		if a.MeteredFeature == "codex_bengalfox" {
			spark = a.RateLimit
			break
		}
	}
	if spark == nil {
		return nil
	}

	// 复用普通 Codex 探测的窗口归一化逻辑，保证 primary/secondary 到 5h/7d 的映射一致。
	snap := &wire.OpenAICodexUsageSnapshot{}
	if w := spark.PrimaryWindow; w != nil {
		p := w.UsedPercent
		snap.PrimaryUsedPercent = &p
		ra := int(w.ResetAfterSeconds)
		snap.PrimaryResetAfterSeconds = &ra
		wm := int(w.LimitWindowSeconds / 60)
		snap.PrimaryWindowMinutes = &wm
	}
	if w := spark.SecondaryWindow; w != nil {
		p := w.UsedPercent
		snap.SecondaryUsedPercent = &p
		ra := int(w.ResetAfterSeconds)
		snap.SecondaryResetAfterSeconds = &ra
		wm := int(w.LimitWindowSeconds / 60)
		snap.SecondaryWindowMinutes = &wm
	}

	normalized := snap.Normalize()
	if normalized == nil {
		return nil
	}

	updates := make(map[string]any)
	if normalized.Used5hPercent != nil {
		updates["codex_5h_used_percent"] = *normalized.Used5hPercent
	}
	if normalized.Reset5hSeconds != nil {
		updates["codex_5h_reset_after_seconds"] = *normalized.Reset5hSeconds
	}
	if normalized.Window5hMinutes != nil {
		updates["codex_5h_window_minutes"] = *normalized.Window5hMinutes
	}
	if normalized.Used7dPercent != nil {
		updates["codex_7d_used_percent"] = *normalized.Used7dPercent
	}
	if normalized.Reset7dSeconds != nil {
		updates["codex_7d_reset_after_seconds"] = *normalized.Reset7dSeconds
	}
	if normalized.Window7dMinutes != nil {
		updates["codex_7d_window_minutes"] = *normalized.Window7dMinutes
	}
	if r := CodexResetAtRFC3339(now, normalized.Reset5hSeconds); r != nil {
		updates["codex_5h_reset_at"] = *r
	}
	if r := CodexResetAtRFC3339(now, normalized.Reset7dSeconds); r != nil {
		updates["codex_7d_reset_at"] = *r
	}
	if len(updates) == 0 {
		return nil
	}
	updates["codex_usage_updated_at"] = now.Format(time.RFC3339)
	return updates
}

func (s *OpenAIQuotaService) StopContext(ctx context.Context) error {
	return s.activity.stop(ctx, "OpenAIQuotaService")
}
