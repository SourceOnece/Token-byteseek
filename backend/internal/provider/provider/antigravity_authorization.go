// 本文件绑定 Antigravity 的原生交换、地址和诊断，授权状态由提供商核心持有。
package provider

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/upstream/antigravity"
)

func AntigravityAuthorizationOptions(resolveProxy func(context.Context, int64) (string, bool)) provider.AntigravityAuthorizationOptions {
	return provider.AntigravityAuthorizationOptions{
		NewClient: func(proxy string) (provider.AntigravityAuthorizationClient, error) {
			return antigravity.NewClient(proxy)
		},
		ResolveProxy:          resolveProxy,
		GenerateState:         antigravity.GenerateState,
		GenerateCodeVerifier:  antigravity.GenerateCodeVerifier,
		GenerateSessionID:     antigravity.GenerateSessionID,
		GenerateCodeChallenge: antigravity.GenerateCodeChallenge,
		BuildAuthorizationURL: antigravity.BuildAuthorizationURL,
		IsConnectionError:     antigravity.IsConnectionError,
		Printf:                func(format string, args ...any) { _, _ = fmt.Printf(format, args...) },
		Warn:                  slog.Warn,
		Info:                  slog.Info,
	}
}
