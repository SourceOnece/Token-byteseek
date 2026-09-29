// OpenAI 授权创建与手动刷新保留原校验、交换、凭据合并及保存顺序；HTTP 不再执行这些规则。
package provider

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

type OpenAIProviderInputError struct{ Message string }

func (e *OpenAIProviderInputError) Error() string { return e.Message }

type OpenAIProviderOperations interface {
	GetProvider(context.Context, int64) (*Record, error)
	CreateProvider(context.Context, *CreateProviderInput) (*Record, error)
	UpdateProvider(context.Context, int64, *UpdateProviderInput) (*Record, error)
}
type OpenAIProviderImport struct {
	Authorization *OpenAIAuthorization
	Admin         OpenAIProviderOperations
	ProxyURL      func(context.Context, int64) (string, bool, error)
}

func NewOpenAIProviderImport(auth *OpenAIAuthorization, admin OpenAIProviderOperations, proxy func(context.Context, int64) (string, bool, error)) *OpenAIProviderImport {
	return &OpenAIProviderImport{Authorization: auth, Admin: admin, ProxyURL: proxy}
}

type OpenAIOAuthProviderCreateInput struct {
	TicketConfiguration                       json.RawMessage
	SessionID, Code, State, RedirectURI, Name string
	ProxyID, TLSFingerprintRouterID           *int64
	Concurrency, Priority                     int
	GroupIDs                                  []int64
}
type OpenAICodexPATCreateInput struct {
	TicketConfiguration json.RawMessage
	AccessToken         string `json:"-"`
	Name                string
	Notes               *string
	GroupIDs            []int64
	ProxyID             *int64
	Concurrency         *int
	Priority            *int
	RateMultiplier      *float64
	LoadFactor          *int
	ExpiresAt           *int64
	AutoPauseOnExpired  *bool
	CredentialExtras    map[string]any `json:"-"`
	Extra               map[string]any
}

func (OpenAICodexPATCreateInput) String() string { return "OpenAI PAT provider creation input" }
func (s *OpenAIProviderImport) RefreshProvider(ctx context.Context, providerID int64, platform string) (*Record, error) {
	// Get provider
	provider, err := s.Admin.GetProvider(ctx, providerID)
	if err != nil {
		return nil, err
	}

	if provider.Platform != platform {
		return nil, &OpenAIProviderInputError{Message: "Provider platform does not match OAuth endpoint"}
	}

	// Only refresh OAuth-based providers
	if !provider.IsOAuth() {
		return nil, &OpenAIProviderInputError{Message: "Cannot refresh non-OAuth provider credentials"}
	}

	// spark 影子提供商凭据透传母提供商、自身恒空,刷新无意义;在调用上游前早拒,避免先打上游
	// 再被凭据写守卫拦下的无谓副作用。
	if provider.IsCredentialShadow() {
		return nil, &OpenAIProviderInputError{Message: "Cannot refresh spark shadow provider; its credentials are managed by the parent provider"}
	}

	ctx, finish, activityErr := s.Authorization.activity.begin(ctx, ErrProbeStopped)
	if activityErr != nil {
		return nil, activityErr
	}
	defer finish()
	// Use OpenAI OAuth service to refresh token
	tokenInfo, err := s.Authorization.RefreshProviderToken(ctx, provider)
	if err != nil {
		return nil, err
	}

	// Build new credentials from token info
	newCredentials := BuildOpenAIProviderCredentials(tokenInfo)

	// Preserve non-token settings from existing credentials
	for k, v := range provider.Credentials {
		if _, exists := newCredentials[k]; !exists {
			newCredentials[k] = v
		}
	}
	newCredentials = NormalizeOpenAIPersonalAccessTokenCredentials(provider, tokenInfo, newCredentials)

	updatedProvider, err := s.Admin.UpdateProvider(ctx, providerID, &UpdateProviderInput{
		Credentials: newCredentials,
	})
	if err != nil {
		return nil, err
	}

	return updatedProvider, nil
}

func (s *OpenAIProviderImport) CreateOAuthProvider(ctx context.Context, req OpenAIOAuthProviderCreateInput, platform string) (*Record, error) {
	ctx, finish, activityErr := s.Authorization.activity.begin(ctx, ErrProbeStopped)
	if activityErr != nil {
		return nil, activityErr
	}
	defer finish()
	// Exchange code for tokens
	tokenInfo, err := s.Authorization.ExchangeCode(ctx, &OpenAIExchangeCodeInput{
		SessionID: req.SessionID,

		Code: req.Code,

		State: req.State,

		RedirectURI: req.RedirectURI,

		ProxyID: req.ProxyID,

		TLSFingerprintRouterID: req.TLSFingerprintRouterID,
	})
	if err != nil {
		return nil, err
	}

	// Build credentials from token info
	credentials := BuildOpenAIProviderCredentials(tokenInfo)

	// Use email as default name if not provided
	name := req.Name
	if name == "" && tokenInfo.Email != "" {
		name = tokenInfo.Email
	}
	if name == "" {
		name = "OpenAI OAuth Provider"
	}

	var extra map[string]any
	if req.TLSFingerprintRouterID != nil && *req.TLSFingerprintRouterID > 0 {
		// 保留提供商与 TLS Router 的绑定，确保后续后台 refresh token 也能使用同一套 token 指纹配置。
		extra = map[string]any{
			"tls_fingerprint_router_id": *req.TLSFingerprintRouterID,
		}
	}

	// Create provider
	provider, err := s.Admin.CreateProvider(ctx, &CreateProviderInput{
		TicketConfiguration: req.TicketConfiguration,
		Name:                name,

		Platform: platform,

		Type: "oauth",

		Credentials: credentials,

		Extra: extra,

		ProxyID: req.ProxyID,

		Concurrency: req.Concurrency,

		Priority: req.Priority,

		GroupIDs: req.GroupIDs,
	})
	if err != nil {
		return nil, err
	}

	return provider, nil
}

func (s *OpenAIProviderImport) CreatePATProvider(ctx context.Context, req OpenAICodexPATCreateInput) (*Record, error) {
	DiscardDeprecatedProviderExtra(req.Extra)
	if req.Concurrency != nil && *req.Concurrency < 0 {
		return nil, &OpenAIProviderInputError{Message: "concurrency must be >= 0"}
	}
	if req.Priority != nil && *req.Priority < 0 {
		return nil, &OpenAIProviderInputError{Message: "priority must be >= 0"}
	}
	if req.RateMultiplier != nil && *req.RateMultiplier < 0 {
		return nil, &OpenAIProviderInputError{Message: "rate_multiplier must be >= 0"}
	}
	if req.LoadFactor != nil && *req.LoadFactor > 10000 {
		return nil, &OpenAIProviderInputError{Message: "load_factor must be <= 10000"}
	}

	var proxyURL string
	if req.ProxyID != nil {
		proxyURLValue, found, err := s.ProxyURL(ctx, *req.ProxyID)
		if err != nil {
			return nil, err
		}
		if found {
			proxyURL = proxyURLValue
		}
	}

	ctx, finish, activityErr := s.Authorization.activity.begin(ctx, ErrProbeStopped)
	if activityErr != nil {
		return nil, activityErr
	}
	defer finish()
	tokenInfo, err := s.Authorization.ValidatePersonalAccessToken(ctx, req.AccessToken, proxyURL)
	if err != nil {
		return nil, err
	}

	credentials := MergeCodexImportMap(
		BuildOpenAIProviderCredentials(tokenInfo),
		SanitizeCodexImportCredentialExtras(req.CredentialExtras),
	)
	extra := MergeCodexImportMap(req.Extra, map[string]any{
		"import_source": "codex_personal_access_token",

		"auth_provider": "codex_personal_access_token",

		"imported_at": time.Now().UTC().Format(time.RFC3339),

		"access_token_sha256": CodexTokenFingerprint(req.AccessToken),
	})

	concurrency := 3
	if req.Concurrency != nil {
		concurrency = *req.Concurrency
	}
	priority := 50
	if req.Priority != nil {
		priority = *req.Priority
	}

	provider, err := s.Admin.CreateProvider(ctx, &CreateProviderInput{
		Name:                BuildOpenAICodexPATProviderName(req.Name, tokenInfo),
		TicketConfiguration: req.TicketConfiguration,

		Notes: req.Notes,

		Platform: "openai",

		Type: "oauth",

		Credentials: credentials,

		Extra: extra,

		ProxyID: req.ProxyID,

		Concurrency: concurrency,

		Priority: priority,

		RateMultiplier: req.RateMultiplier,

		LoadFactor: req.LoadFactor,

		GroupIDs: req.GroupIDs,

		ExpiresAt: req.ExpiresAt,

		AutoPauseOnExpired: req.AutoPauseOnExpired,
	})
	if err != nil {
		return nil, err
	}

	return provider, nil
}

func BuildOpenAICodexPATProviderName(name string, tokenInfo *OpenAITokenInfo) string {
	name = strings.TrimSpace(name)
	if name != "" {
		return name
	}
	if tokenInfo != nil {
		for _, candidate := range []string{tokenInfo.Email, tokenInfo.ChatGPTAccountID, tokenInfo.ChatGPTUserID} {
			if candidate = strings.TrimSpace(candidate); candidate != "" {
				return candidate
			}
		}
	}
	return "Codex PAT Provider"
}
