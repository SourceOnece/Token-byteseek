package text

import (
	"context"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/gateway/failover"
)

// ResponseSelection 保留 HTTP continuation 的逐候选跳过，不把它计为提供商切换。
type ResponseSelection struct {
	Selection
	Available bool
	Skip      *AttemptFailure
	OAuth     failover.OAuth429Provider
}

// ResponseOutcome 将图片部分成功和首输出恢复资格与普通错误明确分开。
type ResponseOutcome struct {
	Outcome
	Images              bool
	NativePartial       bool
	FirstOutputRecovery bool
}

type ResponseOptions struct {
	MaxSwitches       int
	FirstOutputBudget bool
}

// ResponsePorts 只执行一次选取、转发、观测或完成，核心统一拥有重试计数。
type ResponsePorts interface {
	Context() context.Context
	CanAttempt() bool
	Select(map[int64]struct{}) (ResponseSelection, error)
	SelectionFailure(error, int, *AttemptFailure)
	Acquire() bool
	Forward() ResponseOutcome
	PartialImages(error)
	RetryReady(*AttemptFailure) bool
	RetryWait(*AttemptFailure, int, int, time.Duration)
	Exhausted(*AttemptFailure)
	Switched()
	Switching(*AttemptFailure, int, int)
	OtherFailure(error)
	Failed()
	Complete()
	Success()
	Completed(int)
}

// RunResponses 保留同提供商恢复、一次请求的提供商预算和首输出后的禁止重放边界。
func RunResponses(options ResponseOptions, p ResponsePorts) {
	served := false
	if lifecycle, ok := p.(interface {
		Begin()
		Finish(bool)
	}); ok {
		lifecycle.Begin()
		defer func() { lifecycle.Finish(served) }()
	}

	excluded := make(map[int64]struct{})
	sameProvider := make(map[int64]int)
	switches, firstOutputSwitches := 0, 0
	var last *AttemptFailure
	var oauth failover.OAuth429State
	for {
		if !p.CanAttempt() {
			return
		}
		selected, err := p.Select(excluded)
		if err != nil {
			p.SelectionFailure(err, len(excluded), last)
			return
		}
		if !selected.Available {
			return
		}
		if selected.Skip != nil {
			excluded[selected.Provider.ID] = struct{}{}
			last = selected.Skip
			continue
		}
		if !p.Acquire() {
			return
		}
		outcome := p.Forward()
		if outcome.Stop {
			return
		}
		if outcome.Err != nil {
			if fallback, ok := p.(interface{ TryGroupFallback(error) (bool, bool) }); ok && !outcome.NativePartial {
				handled, retry := fallback.TryGroupFallback(outcome.Err)
				if handled {
					if !retry {
						return
					}
					excluded = make(map[int64]struct{})
					sameProvider = make(map[int64]int)
					last = nil
					continue
				}
			}
			if outcome.NativePartial {
				served = true
				p.OtherFailure(outcome.Err)
				p.Complete()
				p.Failed()
				return
			}
			if outcome.Images {
				p.PartialImages(outcome.Err)
			} else {
				if outcome.Failure != nil {
					failure := outcome.Failure
					if !p.RetryReady(failure) {
						return
					}
					if failure.Policy == nil || !failure.Policy.RetryNext {
						p.Exhausted(failure)
						return
					}
					if options.FirstOutputBudget && failover.FirstOutputExhausted(outcome.FirstOutputRecovery, &firstOutputSwitches) {
						p.Exhausted(failure)
						return
					}
					retryLimit := failover.EffectiveSameProviderRetryLimit(failure.Policy, selected.RetryLimit)
					if failure.Policy.RetryableOnSameProvider && failover.SameProviderRetryAllowed(failure.Policy, sameProvider[selected.Provider.ID], retryLimit) {
						if observer, ok := p.(interface{ PrepareRetry(*AttemptFailure, bool) }); ok {
							observer.PrepareRetry(failure, true)
						}
						sameProvider[selected.Provider.ID]++
						delay := failover.SameProviderRetryDelayFor(failure.Policy, sameProvider[selected.Provider.ID])
						p.RetryWait(failure, retryLimit, sameProvider[selected.Provider.ID], delay)
						if !failover.SleepWithContext(p.Context(), delay) {
							return
						}
						continue
					}
					if observer, ok := p.(interface{ PrepareRetry(*AttemptFailure, bool) }); ok {
						observer.PrepareRetry(failure, false)
					}
					p.Switched()
					excluded[selected.Provider.ID] = struct{}{}
					last = failure
					if switches >= options.MaxSwitches {
						p.Exhausted(failure)
						return
					}
					switches++
					if failover.StopOAuth429(selected.OAuth, failure.Policy.StatusCode, switches, &oauth) {
						p.Exhausted(failure)
						return
					}
					p.Switching(failure, switches, options.MaxSwitches)
					continue
				}
				p.OtherFailure(outcome.Err)
				p.Complete()
				p.Failed()
				return
			}
		}
		served = outcome.Attempt.Served
		p.Success()
		p.Complete()
		p.Completed(switches)
		return
	}
}
