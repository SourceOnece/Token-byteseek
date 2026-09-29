// 本次转发的失败事实供调用方决定换号，具体平台识别仍留在平台适配器。
package forward

import (
	"fmt"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/gateway/failover"
)

// GatewayFailureStage 标识请求失败的阶段。零值有意按推理阶段处理，
// 从而保持现有 UpstreamFailoverError 调用方的行为。
type GatewayFailureStage string

const (
	GatewayFailureStageInference         GatewayFailureStage = "inference"
	GatewayFailureStageProviderAuth      GatewayFailureStage = "provider_auth"
	GatewayFailureStageProviderSelection GatewayFailureStage = "provider_selection"
)

// GatewayFailureScope 区分单个提供商、共享凭据设施和请求本身，供切换决策使用。
type GatewayFailureScope string

const (
	GatewayFailureScopeProvider GatewayFailureScope = "provider"
	GatewayFailureScopeShared   GatewayFailureScope = "shared"
	GatewayFailureScopeRequest  GatewayFailureScope = "request"
)

// NextProviderAction 为保持向后兼容采用三态值。零值表示旧版重试行为，
// 只有 NextProviderStop 会显式终止提供商切换。
type NextProviderAction uint8

const (
	NextProviderLegacyRetry NextProviderAction = iota
	NextProviderRetry
	NextProviderStop
)

type GatewayFailureReason string

// UpstreamFailoverError 表示可能触发提供商切换的上游或凭据错误。
// 新增元数据保持现有复合字面量源码兼容，并保留旧版切换提供商行为。
type UpstreamFailoverError struct {
	StatusCode                int
	ResponseBody              []byte              // 上游响应体，用于错误透传规则匹配
	ResponseHeaders           map[string][]string // 上游响应头值；HTTP 适配器按原 Header 规则读取，不把传输对象交给核心
	ForceCacheBilling         bool                // Antigravity 粘性会话切换时设为 true
	RetryableOnSameProvider   bool                // 临时性错误（如 Google 间歇性 400、空响应），应在同一提供商上重试 N 次再切换
	SameProviderRetryDelay    time.Duration
	SameProviderRetryDeadline time.Time
	SameProviderRetryMax      int  // 可选的错误级同提供商重试上限，低于 handler 默认预算时优先采用
	RequestScopedTransient    bool // 故障因素与提供商无关（如上游按客户端身份/模型容量降载）：可同提供商重试，但不得据此对提供商做临时封禁
	SafeToFailoverAfterWrite  bool // 仅写出 SSE 注释等非语义字节时，仍可在同一客户端流中切换提供商
	Stage                     GatewayFailureStage
	Scope                     GatewayFailureScope
	Reason                    GatewayFailureReason
	NextProviderAction        NextProviderAction
	ClientStatusCode          int
	ClientMessage             string
}

func (e *UpstreamFailoverError) Error() string {
	if e != nil && e.Stage == GatewayFailureStageProviderAuth {
		return fmt.Sprintf("credential failure: %s (failover)", e.Reason)
	}
	return fmt.Sprintf("upstream error: %d (failover)", e.StatusCode)
}

func (e *UpstreamFailoverError) ShouldRetryNextProvider() bool {
	return e != nil && e.NextProviderAction != NextProviderStop
}

func (e *UpstreamFailoverError) IsCredentialFailure() bool {
	return e != nil && e.Stage == GatewayFailureStageProviderAuth
}

// ShouldReportProviderScheduleFailure 防止把提供方级或请求级凭据失败误归因到当前提供商。
// 旧版错误和推理错误继续保持原有的调度健康上报行为。
func (e *UpstreamFailoverError) ShouldReportProviderScheduleFailure() bool {
	if e == nil {
		return false
	}
	// 本地票据资格变化只重选提供商，不计入上游健康惩罚。
	if e.Stage == GatewayFailureStageProviderSelection {
		return false
	}
	return !e.IsCredentialFailure() || e.Scope == GatewayFailureScopeProvider
}

func (e *UpstreamFailoverError) RetryFailure() *failover.FailureInfo {
	if e == nil {
		return nil
	}
	return &failover.FailureInfo{StatusCode: e.StatusCode, ForceCacheBilling: e.ForceCacheBilling, RetryableOnSameProvider: e.RetryableOnSameProvider, RequestScopedTransient: e.RequestScopedTransient, SameProviderRetryDelay: e.SameProviderRetryDelay, SameProviderRetryDeadline: e.SameProviderRetryDeadline, SameProviderRetryMax: e.SameProviderRetryMax, RetryNext: e.ShouldRetryNextProvider()}
}
