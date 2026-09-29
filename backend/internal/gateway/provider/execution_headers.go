package provider

import (
	"net/http"

	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
)

// BindExecutionHeaders 固定原方法值捕获的提供商指针，调用时才投影最新字段。
func BindExecutionHeaders(value *ExecutionProvider) func(http.Header) {
	return func(headers http.Header) {
		provideradapter.ApplyProviderHeaderOverrides(ExecutionProtocolRecord(value), headers)
	}
}

// BindExecutionHeaderValue 保留原请求构造时机及字段读取顺序。
func BindExecutionHeaderValue(value *ExecutionProvider) func(string) (string, bool) {
	return func(name string) (string, bool) {
		return provideradapter.HeaderOverrideValue(ExecutionProtocolRecord(value), name)
	}
}
