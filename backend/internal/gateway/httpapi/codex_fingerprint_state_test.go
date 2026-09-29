package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testCodexFingerprintSeed = "11111111-1111-4111-8111-111111111111"

func newTestOAuthProvider(id int64, extra map[string]any) *providercore.Record {
	if providercore.CodexFingerprintModeRequiresSeed(providercore.CodexFingerprintModeFromExtra(extra)) {
		if extra == nil {
			extra = make(map[string]any)
		}
		if _, exists := extra[providercore.CodexFingerprintSeedExtraKey]; !exists {
			extra[providercore.CodexFingerprintSeedExtraKey] = testCodexFingerprintSeed
		}
	}
	return &providercore.Record{
		LoadLocation: time.LoadLocation, ID: id,
		Platform: capability.PlatformOpenAI,
		Type:     capability.ProviderTypeOAuth,
		Extra:    extra,
	}
}

// --- deriveStableUUIDv4 ---

// --- GetCodexFingerprintMode ---

// --- resolveConvergedInstallationID ---

// --- resolveConvergedThreadID ---

// --- off 模式：resolveCodexFingerprintIDsFromRequest 返回 nil ---

// --- applyCodexFingerprintHeaders: off 模式 ---

// --- applyCodexFingerprintHeaders: device 模式 ---

// --- applyCodexFingerprintHeaders: session 模式 ---

// --- session 模式：不同客户端得到不同 thread ---

// --- full 模式 ---

// --- H1 修复验证：头和体的 turn_id 一致性 ---

// --- applyCodexFingerprintClientMetadata ---

// --- extractClientSessionID ---

// --- 透传路径：raw 字节版 client_metadata 改写 ---

// --- context 暂存与出站头应用（透传/非透传共用 seam）---

func newFingerprintStageTestContext(t *testing.T) *gin.Context {
	t.Helper()

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	return c
}

func TestStageCodexFingerprintIDs_NilOverwritesPreviousProvider(t *testing.T) {
	c := newFingerprintStageTestContext(t)
	providerA := newTestOAuthProvider(1001, map[string]any{providercore.CodexFingerprintModeExtraKey: "session"})
	idsA := provideradapter.CodexFingerprintIDs(providerA, "sess-x", providercore.CodexFingerprintSession)
	require.NotNil(t, idsA)
	StageCodexFingerprintIDs(c, idsA)
	StageCodexFingerprintIDs( // failover 切到 off 模式提供商：无条件覆写为 nil，上一提供商 IDs 不得残留
		c, nil)

	h := http.Header{}
	h.Set("session_id", "isolated-session")
	providerB := newTestOAuthProvider(1002, map[string]any{"codex_fingerprint_mode": "off"})
	ApplyStagedCodexFingerprintHeaders(c, providerB, h)
	assert.Equal(t, "isolated-session", h.Get("session_id"), "off 提供商不得应用上一提供商的收敛 ID")
	assert.Empty(t, h.Get("x-codex-installation-id"))
}

func TestApplyStagedCodexFingerprintRejectsDifferentOAuthProvider(t *testing.T) {
	c := newFingerprintStageTestContext(t)
	providerA := newTestOAuthProvider(1003, map[string]any{providercore.CodexFingerprintModeExtraKey: "session"})
	idsA := provideradapter.CodexFingerprintIDs(providerA, "sess-a", providercore.CodexFingerprintSession)
	require.NotNil(t, idsA)
	StageCodexFingerprintIDs(c, idsA)

	providerB := newTestOAuthProvider(1004, map[string]any{providercore.CodexFingerprintModeExtraKey: "session"})
	h := make(http.Header)
	h.Set("session-id", "provider-b-session")
	ApplyStagedCodexFingerprintHeaders(c, providerB, h)
	assert.Equal(t, "provider-b-session", h.Get("session-id"))
	assert.Empty(t, h.Get("x-codex-installation-id"))

	body := map[string]any{"client_metadata": map[string]any{"session_id": "provider-b-session"}}
	assert.False(t, ApplyStagedCodexFingerprintClientMetadata(c, providerB, body))
	clientMetadata, ok := body["client_metadata"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "provider-b-session", clientMetadata["session_id"])
}

func TestApplyStagedCodexFingerprintHeaders_SkipsNonOAuthProvider(t *testing.T) {
	c := newFingerprintStageTestContext(t)
	oauthIDs := provideradapter.CodexFingerprintIDs(newTestOAuthProvider(1003, map[string]any{providercore.CodexFingerprintModeExtraKey: "session"}), "sess-y", providercore.CodexFingerprintSession)
	require.NotNil(t, oauthIDs)
	StageCodexFingerprintIDs(c, oauthIDs)

	h := http.Header{}
	apiKeyProvider := &providercore.Record{LoadLocation: time.LoadLocation, ID: 1004, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeAPIKey}
	ApplyStagedCodexFingerprintHeaders(c, apiKeyProvider, h)
	assert.Empty(t, h.Get("x-codex-installation-id"), "stale 收敛 ID 不得应用到非 OAuth 提供商")
}
