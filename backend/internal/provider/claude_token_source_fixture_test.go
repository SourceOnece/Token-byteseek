//go:build unit

package provider

import "log/slog"

// 原缓存与等待断言直接执行提供商 token 用例。
func newClaudeTokenSourceContract(cache AccessTokenCache) *ClaudeTokenSource {
	return &ClaudeTokenSource{Options: ClaudeTokenOptions{
		Cache: cache, Policy: ClaudeProviderRefreshPolicy(), Debug: slog.Debug, Warn: slog.Warn,
	}}
}
