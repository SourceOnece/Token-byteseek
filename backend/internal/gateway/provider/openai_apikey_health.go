package provider

import (
	"context"
	"errors"
	"net/http"
	"strings"

	forwardcore "github.com/TokenFlux/TokenRouter/internal/gateway/forward"
	"github.com/TokenFlux/TokenRouter/internal/upstream/openai"
)

// ClassifyOpenAIAPIKeyHealthFailure 保留请求取消、平台故障与提供商故障的原归因边界。
func ClassifyOpenAIAPIKeyHealthFailure(err error) (int, []byte, bool) {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return 0, nil, false
	}

	var failoverErr *forwardcore.UpstreamFailoverError
	if errors.As(err, &failoverErr) {
		// 已有独立恢复、同提供商重试或非提供商归因的错误，不进入健康计数。
		if failoverErr.IsCredentialFailure() ||
			failoverErr.RequestScopedTransient ||
			failoverErr.RetryableOnSameProvider ||
			failoverErr.Scope == forwardcore.GatewayFailureScopeRequest ||
			failoverErr.Scope == forwardcore.GatewayFailureScopeShared {
			return failoverErr.StatusCode, failoverErr.ResponseBody, false
		}
		if failoverErr.StatusCode == http.StatusTooManyRequests || failoverErr.StatusCode >= http.StatusInternalServerError {
			return failoverErr.StatusCode, failoverErr.ResponseBody, true
		}
		return failoverErr.StatusCode, failoverErr.ResponseBody, false
	}

	var imageErr *openai.OpenAIImagesUpstreamError
	if errors.As(err, &imageErr) {
		if imageErr.StatusCode == http.StatusTooManyRequests || imageErr.StatusCode >= http.StatusInternalServerError {
			return imageErr.StatusCode, []byte(strings.TrimSpace(imageErr.Message)), true
		}
	}
	return 0, nil, false
}
