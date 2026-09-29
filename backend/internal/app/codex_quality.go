package app

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/app/lifecycle"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	providerpostgres "github.com/TokenFlux/TokenRouter/internal/provider/postgres"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
	"github.com/TokenFlux/TokenRouter/internal/quality"
	qualitypostgres "github.com/TokenFlux/TokenRouter/internal/quality/postgres"
	"github.com/TokenFlux/TokenRouter/internal/scheduler"
)

type qualityAnswer struct {
	text string
	err  error
}

func (*qualityAnswer) Begin(context.Context, bool) error { return nil }
func (a *qualityAnswer) Emit(_ context.Context, event provider.TestEvent) error {
	if event.Type == "content" {
		a.text += event.Text
	}
	if event.Type == "error" {
		a.err = errors.New(event.Error)
	}
	return nil
}

// 管理员手动和定时检测共用实例锁，跨实例互斥继续由数据库租约负责。
func provideCodexQuality(data *providerpostgres.ProviderStore, db *sql.DB, cache scheduler.SnapshotCache, executor *provideradapter.OpenAIProviderTest, manager *lifecycle.Manager) *quality.Service {
	store := qualitypostgres.New(data, db, newProviderEvents(data, cache))
	core := quality.New(store, func(ctx context.Context, account *provider.Record, options *quality.CodexQualityRequest) (string, error) {
		answer := &qualityAnswer{}
		run := provideradapter.NewTestRun(ctx, nil, answer)
		defer run.Cancel()
		run.Automatic = true
		run.RequestedProtocol = provider.TextProtocol(options.APIProtocol)
		run.Quality = &provideradapter.QualityProbe{ReasoningEffort: options.ReasoningEffort, Responses: quality.Responses, Chat: quality.Chat}
		if err := executor.Prepare(run, account); err != nil {
			return "", err
		}
		err := executor.Execute(run, account, options.Model, options.Prompt, provider.ProviderTestModeDefault, provider.ProviderTestTypeText)
		return answer.text, errors.Join(run.Result(err), answer.err)
	})
	var stop context.CancelFunc
	done := make(chan struct{})
	manager.Register(lifecycle.Hook{Name: "CodexQualitySchedules", StartOrder: 961, StopOrder: 15,
		Start: func(parent context.Context) error {
			ctx, cancel := context.WithCancel(parent)
			stop = cancel
			go func() {
				defer close(done)
				ticker := time.NewTicker(15 * time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						core.RunDueQualitySchedule(ctx)
					}
				}
			}()
			return nil
		},
		Stop: func(ctx context.Context) error {
			if stop == nil {
				return nil
			}
			stop()
			select {
			case <-done:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	})
	return core
}
