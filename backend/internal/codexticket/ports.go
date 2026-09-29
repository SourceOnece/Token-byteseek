package codexticket

import (
	"context"
	"encoding/json"
	"github.com/TokenFlux/TokenRouter/internal/egress"
	"github.com/TokenFlux/TokenRouter/internal/gateway/clientmeta"
	"github.com/TokenFlux/TokenRouter/internal/gateway/forward"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/scheduler"
	"github.com/TokenFlux/TokenRouter/internal/settings"
	"github.com/TokenFlux/TokenRouter/internal/upstream"
	upstreamopenai "github.com/TokenFlux/TokenRouter/internal/upstream/openai"
	"net/http"
	"strings"
)

// 票据只依赖原生模块的值与窄端口，不实例化旧 service 或第二套网关。
type Account = provider.Record
type Proxy = egress.Proxy
type SettingRepository = settings.Repository
type ProxyExitInfoProber = egress.ProxyExitInfoProber
type AcquireResult = scheduler.AcquireResult
type ProxyRepository interface {
	GetByID(context.Context, int64) (*egress.Proxy, error)
}
type SecretEncryptor interface {
	Encrypt(string) (string, error)
	Decrypt(string) (string, error)
}

var ErrSettingNotFound = settings.ErrSettingNotFound
var ErrAccountNotFound = provider.ErrProviderNotFound

const PlatformOpenAI = "openai"
const AccountTypeOAuth = "oauth"
const StatusActive = "active"
const StatusError = "error"
const StatusDisabled = "disabled"
const openAICodexTurnStateHeader = "x-codex-turn-state"
const chatgptCodexURL = "https://chatgpt.com/backend-api/codex/responses"

func extractOpenAICodexTurnState(headers http.Header) string {
	return strings.TrimSpace(headers.Get(openAICodexTurnStateHeader))
}

var WithHTTPUpstreamRedirectsDisabled = upstream.WithHTTPUpstreamRedirectsDisabled
var WithHTTPUpstreamProfile = upstream.WithHTTPUpstreamProfile

const HTTPUpstreamProfileOpenAIHarvest upstream.HTTPUpstreamProfile = "openai_harvest"

var ensureCodexIdentityHeaders = upstreamopenai.EnsureCodexIdentityHeaders
var enforceCodexIdentityHeaders = upstreamopenai.EnforceCodexIdentityHeaders
var CompareVersions = clientmeta.CompareVersions

// 邮箱诊断沿用原有读取语义，只读取本次账号的已配置字段。
func firstStringValue(values map[string]any, keys ...string) string {
	for _, key := range keys {
		raw := values[key]
		if raw == nil {
			continue
		}
		if value, ok := raw.(string); ok {
			if value = strings.TrimSpace(value); value != "" {
				return value
			}
		} else if encoded, err := json.Marshal(raw); err == nil {
			value := strings.Trim(strings.TrimSpace(string(encoded)), "\"")
			if value != "" && value != "null" {
				return value
			}
		}
	}
	return ""
}

type GatewayFailureReason = forward.GatewayFailureReason
type UpstreamFailoverError = forward.UpstreamFailoverError

const GatewayFailureStageAccountSelection forward.GatewayFailureStage = "provider_selection"
const GatewayFailureScopeAccount = forward.GatewayFailureScopeProvider
const NextAccountRetry = forward.NextProviderRetry
