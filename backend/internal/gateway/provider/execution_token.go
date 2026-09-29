package provider

import (
	"context"

	"github.com/TokenFlux/TokenRouter/internal/provider"
)

// ExecutionTokenSource 仅供执行提供商的受控凭据读取，原生源不接收旧实体。
type ExecutionTokenSource interface {
	GetAccessToken(context.Context, *provider.Record) (string, error)
}

// ExecutionToken 保留 Gemini/Antigravity project 回填后的旧调用方赋值时机。
func ExecutionToken(ctx context.Context, source ExecutionTokenSource, value *ExecutionProvider) (string, error) {
	record := ExecutionRecord(value)
	token, err := source.GetAccessToken(ctx, record)
	if value != nil && record != nil {
		value.Record.Credentials = record.Credentials
	}
	return token, err
}
