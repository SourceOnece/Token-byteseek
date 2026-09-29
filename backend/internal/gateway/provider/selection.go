package provider

import (
	textflow "github.com/TokenFlux/TokenRouter/internal/gateway/text"
	"github.com/TokenFlux/TokenRouter/internal/scheduler"
	"github.com/TokenFlux/TokenRouter/internal/scheduler/policy"
)

// SelectionResult 是执行边界的已选目标和资源结果；调度核心仍只读取无凭据投影。
// 资源由已有 Lease 接管，反馈参数固化于本次选择，不重新读取保存后的设置。
type SelectionResult struct {
	Provider                  *ExecutionProvider
	Acquired                  bool
	ReleaseFunc               func()
	WaitPlan                  *scheduler.ProviderWaitPlan
	AdvancedScheduler         bool
	AdvancedSchedulerFeedback *policy.FeedbackConfig
}

// capturedTextSelection 在候选返回时立即取得实际计划，结束后不再查看可变候选状态。
func CaptureTextSelection(provider *ExecutionProvider) textflow.Selection {
	plan, provided := ExecutionCandidatePlan(provider)
	return textflow.Selection{Provider: ExecutionSnapshot(provider), RetryLimit: provider.View().GetPoolModeRetryCount(), Plan: plan, PlanProvided: provided}
}
