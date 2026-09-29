package messageforward

import (
	"context"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/gateway"
	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	anthropicfp "github.com/TokenFlux/TokenRouter/internal/upstream/anthropic"
	"github.com/stretchr/testify/require"
)

// TestGatewayClientDatelineNormalization_Scope 覆盖提供商类型与开关组合：
// 只有开关开启时 Anthropic OAuth/SetupToken 才会通过，API-Key 与非 Anthropic
// 平台始终跳过。
func TestGatewayClientDatelineNormalization_Scope(t *testing.T) {
	repo := &betaSettingsFixture{values: map[string]string{}}
	svc := NewRuntime(Dependencies{Settings: newBetaRuntime(repo.values)}, Options{})
	ctx := context.Background()

	// 默认缺省：parseSettings 与缓存加载器的 fallback 都是 true。
	require.True(t, svc.shouldNormalizeDateline(ctx, &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, Platform: capability.PlatformAnthropic, Type: capability.ProviderTypeOAuth}}))
	require.True(t, svc.shouldNormalizeDateline(ctx, &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, Platform: capability.PlatformAnthropic, Type: capability.ProviderTypeSetupToken}}))
	require.False(t, svc.shouldNormalizeDateline(ctx, &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, Platform: capability.PlatformAnthropic, Type: capability.ProviderTypeAPIKey}}))
	require.False(t, svc.shouldNormalizeDateline(ctx, &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, Platform: capability.PlatformOpenAI, Type: capability.ProviderTypeOAuth}}))

	// 关闭开关：任何提供商都不归一化。
	repo.values[gateway.SettingKeyEnableClientDatelineNormalization] = "false"
	svc.dependencies.Settings.InvalidateForwarding()
	require.False(t, svc.shouldNormalizeDateline(ctx, &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, Platform: capability.PlatformAnthropic, Type: capability.ProviderTypeOAuth}}))
	require.False(t, svc.shouldNormalizeDateline(ctx, &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, Platform: capability.PlatformAnthropic, Type: capability.ProviderTypeSetupToken}}))

	// 重新开启开关：OAuth 再次通过。
	repo.values[gateway.SettingKeyEnableClientDatelineNormalization] = "true"
	svc.dependencies.Settings.InvalidateForwarding()
	require.True(t, svc.shouldNormalizeDateline(ctx, &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, Platform: capability.PlatformAnthropic, Type: capability.ProviderTypeOAuth}}))
}

// TestGatewayClientDatelineNormalization_HelperNoRewrite 覆盖 Forward 使用的辅助路径：
// 开关关闭、API-Key、空提供商或请求体没有指纹 dateline 时返回 ok=false；
// 开关开启且提供商为 Anthropic OAuth/SetupToken 并实际改写时返回 ok=true 和新请求体。
func TestGatewayClientDatelineNormalization_HelperNoRewrite(t *testing.T) {
	repo := &betaSettingsFixture{values: map[string]string{
		gateway.SettingKeyEnableClientDatelineNormalization: "true",
	}}
	svc := NewRuntime(Dependencies{Settings: newBetaRuntime(repo.values)}, Options{})
	ctx := context.Background()

	dirty := []byte(`{"messages":[{"role":"user","content":"<system-reminder>\nToday’s date is 2026/07/01.\n</system-reminder>"}]}`)
	clean := []byte(`{"messages":[{"role":"user","content":"just hello"}]}`)

	// API-Key 提供商：即使请求体包含指纹也不改写。
	next, ok := svc.normalizeDateline(ctx, &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, Platform: capability.PlatformAnthropic, Type: capability.ProviderTypeAPIKey}}, dirty)
	require.False(t, ok)
	require.Nil(t, next)

	// 空提供商：安全跳过。
	next, ok = svc.normalizeDateline(ctx, nil, dirty)
	require.False(t, ok)
	require.Nil(t, next)

	// OAuth 提供商 + 干净请求体：没有变化，ok=false。
	next, ok = svc.normalizeDateline(ctx, &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, Platform: capability.PlatformAnthropic, Type: capability.ProviderTypeOAuth}}, clean)
	require.False(t, ok)
	require.Nil(t, next)

	// OAuth 提供商 + 带指纹请求体：完成改写，ok=true。
	next, ok = svc.normalizeDateline(ctx, &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, Platform: capability.PlatformAnthropic, Type: capability.ProviderTypeOAuth}}, dirty)
	require.True(t, ok)
	require.NotNil(t, next)
	require.Contains(t, string(next), "Today's date is 2026-07-01.")
	require.NotContains(t, string(next), "2026/07/01")
	require.NotContains(t, string(next), "Today’s date is")

	// SetupToken 提供商 + 带指纹请求体：完成改写，ok=true。
	next, ok = svc.normalizeDateline(ctx, &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, Platform: capability.PlatformAnthropic, Type: capability.ProviderTypeSetupToken}}, dirty)
	require.True(t, ok)
	require.Contains(t, string(next), "Today's date is 2026-07-01.")

	// 关闭开关：即使 OAuth 提供商也不改写。
	repo.values[gateway.SettingKeyEnableClientDatelineNormalization] = "false"
	svc.dependencies.Settings.InvalidateForwarding()
	next, ok = svc.normalizeDateline(ctx, &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, Platform: capability.PlatformAnthropic, Type: capability.ProviderTypeOAuth}}, dirty)
	require.False(t, ok)
	require.Nil(t, next)
}

// TestGatewayClientDatelineNormalization_LeavesUserProseUntouched 确认纯转换不会触碰
// <system-reminder> 外的内容；这是开关辅助函数与 anthropicfp 作用域约定之间的集成保护。
func TestGatewayClientDatelineNormalization_LeavesUserProseUntouched(t *testing.T) {
	repo := &betaSettingsFixture{values: map[string]string{
		gateway.SettingKeyEnableClientDatelineNormalization: "true",
	}}
	svc := NewRuntime(Dependencies{Settings: newBetaRuntime(repo.values)}, Options{})
	ctx := context.Background()

	// 用户文本如果只是在 <system-reminder> 外碰巧包含类似指纹的句子，必须逐字节保留。
	body := []byte(`{"messages":[{"role":"user","content":"I wrote: Today’s date is 2026/07/01. What do you think?"}]}`)
	next, ok := svc.normalizeDateline(ctx, &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, Platform: capability.PlatformAnthropic, Type: capability.ProviderTypeOAuth}}, body)
	require.False(t, ok, "must not rewrite user prose outside <system-reminder>")
	require.Nil(t, next)

	// 直接验证纯函数，避免后续扩大扫描范围。
	out, hits, changed := anthropicfp.NormalizeDateline(body)
	require.False(t, changed)
	require.Empty(t, hits)
	require.Equal(t, body, out)
}
