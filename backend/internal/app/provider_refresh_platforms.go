package app

import (
	"fmt"
	"log"

	egressprovider "github.com/TokenFlux/TokenRouter/internal/egress/provider"
	"github.com/TokenFlux/TokenRouter/internal/infra/httpclient"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
)

// providerRefreshRegistrations 区分平台注册集合，避免装配重复创建执行器。
type providerRefreshRegistrations []provider.RefreshRegistration

func provideRefreshPlatforms(claude *provider.ClaudeAuthorization, openai *provider.OpenAIAuthorization, gemini *provider.GeminiAuthorization, antigravity *provider.AntigravityAuthorization, qoder *provideradapter.QoderAuthorization, grok *provider.GrokAuthorization, transport httpclient.UpstreamTransport, profiles *egressprovider.TLSProfiles) providerRefreshRegistrations {
	claudeRefresh := &provider.ClaudeTokenRefresher{Authorization: claude}
	openaiRefresh := &provider.OpenAITokenRefresher{Authorization: openai}
	geminiRefresh := &provider.GeminiTokenRefresher{Authorization: gemini, Key: provideradapter.GeminiTokenCacheKey}
	agRefresh := &provider.AntigravityRefreshRules{
		RefreshProviderToken:     antigravity.RefreshProviderToken,
		BuildProviderCredentials: antigravity.BuildProviderCredentials,
		Printf:                   func(format string, args ...any) { _, _ = fmt.Printf(format, args...) },
		Logf:                     log.Printf,
	}
	qoderRefresh := provideradapter.NewQoderTokenRefresher(provideradapter.QoderRefreshOptions{
		Transport: transport, Profiles: profiles, BuildCredentials: qoder.Core.BuildProviderCredentials,
	})
	grokRefresh := provider.NewGrokTokenRefresher(grok)
	return providerRefreshRegistrations{
		{Platform: provider.PlatformAnthropic, Refresher: claudeRefresh, Executor: claudeRefresh},
		{Platform: provider.PlatformOpenAI, Refresher: openaiRefresh, Executor: openaiRefresh},
		{Platform: provider.PlatformGemini, Refresher: geminiRefresh, Executor: geminiRefresh},
		{Platform: provider.PlatformAntigravity, Refresher: agRefresh, Executor: agRefresh},
		{Platform: provider.PlatformQoder, Refresher: qoderRefresh, Executor: qoderRefresh},
		{Platform: provider.PlatformGrok, Refresher: grokRefresh, Executor: grokRefresh},
	}
}
