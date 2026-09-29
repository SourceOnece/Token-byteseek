// 凭据目标适配只投影母提供商查询；影子资格规则由 provider 唯一实现。
package provider

import (
	"context"
	"net/http"

	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"

	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
)

func CredentialProvider(ctx context.Context, repo ExecutionProviderReader, value *ExecutionProvider) (*ExecutionProvider, error) {
	input := ExecutionRecord(value)
	var parent *ExecutionProvider
	resolved, err := providercore.ResolveCredentialRecord(ctx, func(ctx context.Context, id int64) (*providercore.Record, error) {
		var readErr error
		parent, readErr = repo.GetByID(ctx, id)
		return ExecutionRecord(parent), readErr
	}, input)
	if err != nil {
		return nil, err
	}
	if resolved == input {
		return value, nil
	}
	return parent, nil
}

// ExecutionProviderReader 只开放此处所需的一次提供商读取。
type ExecutionProviderReader interface {
	GetByID(context.Context, int64) (*ExecutionProvider, error)
}

// CredentialChatGPTHeaders 保留先解析母提供商、再应用请求头的原顺序。
func CredentialChatGPTHeaders(ctx context.Context, reader ExecutionProviderReader, headers http.Header, value *ExecutionProvider) error {
	resolved, err := CredentialProvider(ctx, reader, value)
	if err != nil {
		return err
	}
	provideradapter.SetChatGPTAccountHeaders(headers, resolved.View())
	return nil
}
