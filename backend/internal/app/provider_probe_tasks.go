package app

import (
	"context"
	"log/slog"

	gatewayhttp "github.com/TokenFlux/TokenRouter/internal/gateway/httpapi"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/provider/postgres"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
)

// provideProviderProbeTasks 绑定原 task 协调与条件凭据写入，app 不持有锁或任务规则。
func provideProviderProbeTasks(store *postgres.ProviderStore, connections *gatewayhttp.OpenAIWSConnections, coordinator *provider.OpenAITaskCoordinator) *provideradapter.ProbeTasks {
	return &provideradapter.ProbeTasks{Coordinator: coordinator, Options: provider.OpenAITaskOptions{
		Read: store.GetByID,
		Register: func(ctx context.Context, value *provider.Record) (string, error) {
			return provideradapter.RegisterAgentIdentityTask(ctx, value, "https://auth.openai.com/api/accounts")
		},
		Persist: func(ctx context.Context, value *provider.Record, credentials map[string]any) error {
			_, err := provider.PersistCredentials(ctx, store, value, credentials, slog.Warn)
			return err
		},
		Invalidate: connections.InvalidateProvider,
	}}
}
