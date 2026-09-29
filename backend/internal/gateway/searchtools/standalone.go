package searchtools

import (
	"context"

	"github.com/TokenFlux/TokenRouter/internal/scheduler"
	"github.com/TokenFlux/TokenRouter/internal/search/contract"
)

// Selection 只携带选中标识和取得资源，不向核心暴露提供商凭据或旧实体。
type Selection struct {
	ProviderID int64
	Acquired   bool
	Release    func()
	WaitPlan   *scheduler.ProviderWaitPlan
}
type StandalonePorts interface {
	Select(context.Context, string, map[int64]struct{}) (Selection, bool, error)
	Acquire(context.Context, Selection) (func(), bool, error)
	Execute(context.Context, int64, StandaloneRequest, string, int) (*contract.SearchResponse, string, error)
	CanSwitch(error) bool
}
type StandaloneResult struct {
	ProviderID int64
	Response   *contract.SearchResponse
	Provider   string
}
type StandaloneFailure struct {
	Stage string
	Cause error
}

func (e *StandaloneFailure) Error() string {
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return "web search failed"
}
func (e *StandaloneFailure) Unwrap() error { return e.Cause }

// noAvailableProvidersError 保留原 HTTP 使用的首字母大写消息，表示选择阶段没有提供商。
type noAvailableProvidersError struct{}

func (noAvailableProvidersError) Error() string { return "No available providers" }

// RunStandalone 保留最多四次提供商选择、首次失败映射及后续 failover 的原有区别。
// 请求 Lease 由 HTTP 在写完响应后释放；失败切换只提前释放当前 attempt。
func RunStandalone(ctx context.Context, request StandaloneRequest, model string, maxResults int, ports StandalonePorts, lease *scheduler.Lease) (StandaloneResult, error) {
	failed := make(map[int64]struct{})
	var result StandaloneResult
	hasProvider := false
	var lastErr error
	for attempt := 0; attempt < 4; attempt++ {
		selection, present, err := ports.Select(ctx, model, failed)
		if err != nil {
			if attempt == 0 {
				return result, &StandaloneFailure{Stage: "selection", Cause: err}
			}
			break
		}
		if !present {
			if attempt == 0 {
				return result, &StandaloneFailure{Stage: "selection", Cause: noAvailableProvidersError{}}
			}
			break
		}
		release, ok, err := ports.Acquire(ctx, selection)
		if !ok {
			if attempt == 0 && err != nil {
				return result, &StandaloneFailure{Stage: "concurrency", Cause: err}
			}
			failed[selection.ProviderID] = struct{}{}
			continue
		}
		attemptLease := scheduler.NewAttemptLease(lease, nil, release)
		result.ProviderID = selection.ProviderID
		hasProvider = true
		result.Response, result.Provider, lastErr = ports.Execute(ctx, selection.ProviderID, request, model, maxResults)
		if lastErr == nil {
			break
		}
		if !ports.CanSwitch(lastErr) {
			break
		}
		failed[selection.ProviderID] = struct{}{}
		attemptLease.Release()
		result.ProviderID = 0
		hasProvider = false
	}
	if lastErr != nil || result.Response == nil {
		return result, &StandaloneFailure{Stage: "execute", Cause: lastErr}
	}
	if !hasProvider {
		return result, &StandaloneFailure{Stage: "selection", Cause: noAvailableProvidersError{}}
	}
	return result, nil
}
