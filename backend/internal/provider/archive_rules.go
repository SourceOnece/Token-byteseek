package provider

import (
	"errors"
	"fmt"
	"strings"

	"github.com/TokenFlux/TokenRouter/internal/provider/transfer"
)

func ApplyArchiveDefaults(item *transfer.DataProvider, defaults *transfer.OpenAIOAuthImportDefaults) {
	if defaults == nil {
		return
	}
	if !strings.EqualFold(strings.TrimSpace(item.Platform), PlatformOpenAI) {
		return
	}
	if !strings.EqualFold(strings.TrimSpace(item.Type), ProviderTypeOAuth) {
		return
	}

	if !item.NotesSet && defaults.Provider.Notes != nil {
		item.Notes = clonePointer(defaults.Provider.Notes)
	}
	if !item.ConcurrencySet && defaults.Provider.Concurrency != nil {
		item.Concurrency = clonePointer(defaults.Provider.Concurrency)
	}
	if !item.PrioritySet && defaults.Provider.Priority != nil {
		item.Priority = clonePointer(defaults.Provider.Priority)
	}
	if !item.RateMultiplierSet && defaults.Provider.RateMultiplier != nil {
		item.RateMultiplier = clonePointer(defaults.Provider.RateMultiplier)
	}
	if !item.ExpiresAtSet && defaults.Provider.ExpiresAt != nil {
		item.ExpiresAt = clonePointer(defaults.Provider.ExpiresAt)
	}
	if !item.AutoPauseOnExpiredSet && defaults.Provider.AutoPauseOnExpired != nil {
		item.AutoPauseOnExpired = clonePointer(defaults.Provider.AutoPauseOnExpired)
	}

	mergeArchiveDefaults(&item.Credentials, defaults.Credentials)
	mergeArchiveDefaults(&item.Extra, defaults.Extra)
}

func mergeArchiveDefaults(target *map[string]any, defaults map[string]any) {
	if len(defaults) == 0 {
		return
	}
	if *target == nil {
		*target = map[string]any{}
	}
	for key, value := range defaults {
		// 只按顶层键做缺失合并；null、false、0、空数组都算已提供。
		if _, exists := (*target)[key]; !exists {
			(*target)[key] = CloneValues(map[string]any{key: value})[key]
		}
	}
}

func ValidateArchiveProvider(item transfer.DataProvider) error {
	if strings.TrimSpace(item.Name) == "" {
		return errors.New("provider name is required")
	}
	if strings.TrimSpace(item.Platform) == "" {
		return errors.New("provider platform is required")
	}
	if strings.TrimSpace(item.Type) == "" {
		return errors.New("provider type is required")
	}
	if len(item.Credentials) == 0 {
		return errors.New("provider credentials is required")
	}
	switch item.Type {
	case ProviderTypeOAuth, ProviderTypeSetupToken, ProviderTypeAPIKey, ProviderTypeUpstream,
		ProviderTypeBedrock, ProviderTypeServiceAccount, ProviderTypeCosy:
	default:
		return fmt.Errorf("provider type is invalid: %s", item.Type)
	}
	platform := strings.ToLower(strings.TrimSpace(item.Platform))
	if platform == PlatformQoder && item.Type != ProviderTypeCosy {
		return fmt.Errorf("qoder providers require %s provider type", ProviderTypeCosy)
	}
	if platform != PlatformQoder && item.Type == ProviderTypeCosy {
		return fmt.Errorf("%s provider type requires %s platform", ProviderTypeCosy, PlatformQoder)
	}
	if item.RateMultiplier != nil && *item.RateMultiplier < 0 {
		return errors.New("rate_multiplier must be >= 0")
	}
	if item.Concurrency != nil && *item.Concurrency < 0 {
		return errors.New("concurrency must be >= 0")
	}
	if item.Priority != nil && *item.Priority < 0 {
		return errors.New("priority must be >= 0")
	}
	return nil
}

// ArchiveIdentityHints 是供应商解码后的最小投影，不能用于认证或授权。
type ArchiveIdentityHints struct{ Email, PlanType, ChatGPTAccountID, ChatGPTUserID, OrganizationID string }

// ArchiveIDToken 只选择原 OpenAI OAuth 导入的可选身份提示，不扩展其它导入入口。
func ArchiveIDToken(item *transfer.DataProvider) string {
	if item == nil || item.Credentials == nil || strings.ToLower(strings.TrimSpace(item.Platform)) != PlatformOpenAI || strings.ToLower(strings.TrimSpace(item.Type)) != ProviderTypeOAuth {
		return ""
	}
	token, _ := item.Credentials["id_token"].(string)
	if strings.TrimSpace(token) == "" {
		return ""
	}
	return token
}

// FillArchiveIdentity 只填原先缺失的字符串，不覆盖显式提供商信息或增加 token 验证策略。
func FillArchiveIdentity(item *transfer.DataProvider, hints *ArchiveIdentityHints) {
	if item == nil || hints == nil || item.Credentials == nil {
		return
	}
	set := func(key, value string) {
		if value == "" {
			return
		}
		if existing, _ := item.Credentials[key].(string); existing == "" {
			item.Credentials[key] = value
		}
	}
	set("email", hints.Email)
	set("plan_type", hints.PlanType)
	set("chatgpt_account_id", hints.ChatGPTAccountID)
	set("chatgpt_user_id", hints.ChatGPTUserID)
	set("organization_id", hints.OrganizationID)
}
