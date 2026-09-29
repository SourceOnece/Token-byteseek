package app

import (
	"context"
	"encoding/json"
	"github.com/TokenFlux/TokenRouter/internal/codexticket"
	"log/slog"
	"time"

	egressadapter "github.com/TokenFlux/TokenRouter/internal/egress/provider"
	"github.com/TokenFlux/TokenRouter/internal/infra/httpclient"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"

	"github.com/TokenFlux/TokenRouter/internal/app/lifecycle"
	providerpostgres "github.com/TokenFlux/TokenRouter/internal/provider/postgres"

	billingpostgres "github.com/TokenFlux/TokenRouter/internal/billing/postgres"

	egresspostgres "github.com/TokenFlux/TokenRouter/internal/egress/postgres"
	"github.com/TokenFlux/TokenRouter/internal/routing"

	routingpostgres "github.com/TokenFlux/TokenRouter/internal/routing/postgres"
	"github.com/google/uuid"
)

// provideProviderAdmin 绑定唯一管理用例与原平台执行端口，构造不运行后台任务。
func provideProviderAdmin(store *providerpostgres.ProviderStore, usage *billingpostgres.ProviderUsageStore, blocker provider.RuntimeUnblocker, privacy *provider.PrivacyService, groups *routingpostgres.GroupStore, proxies *egresspostgres.ProxyStore, tasks *lifecycle.Tasks, upstream httpclient.UpstreamTransport, tls *egressadapter.TLSProfiles, tickets *codexticket.CodexTicketService) *provider.Admin {
	options := provider.AdminOptions{ShadowModels: provideradapter.DefaultSparkShadowModels, Duplicates: store, Quotas: usage, RuntimeBlocker: blocker, Privacy: privacy, Groups: providerGroupReferences{groups}, Proxies: proxies, Creation: provider.CreationOptions{Now: time.Now, LoadLocation: time.LoadLocation, NewSeed: uuid.NewString}, Credentials: provideradapter.CreateCredentialHooks(upstream, tls), Background: tasks.Go, Error: slog.Error}
	options.CreateConfigured = func(ctx context.Context, value *provider.Record, groups []int64, raw json.RawMessage) (bool, error) {
		if value.Platform != provider.PlatformOpenAI || value.Type != provider.ProviderTypeOAuth || value.IsCredentialShadow() || value.IsOpenAIAgentIdentity() {
			return false, nil
		}
		var patch *codexticket.CodexTicketAccountPatch
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &patch); err != nil {
				return false, err
			}
		}
		return tickets.CreateAccountWithDefaults(ctx, value, groups, patch)
	}
	return provider.NewAdmin(store, options)
}

type providerGroupReferences struct{ store *routingpostgres.GroupStore }

func (g providerGroupReferences) GetGroup(ctx context.Context, id int64) (*provider.GroupReference, error) {
	value, err := g.store.GetByID(ctx, id)
	if value == nil {
		return nil, err
	}
	return &provider.GroupReference{ID: value.ID, Name: value.Name, RequireOAuthOnly: value.RequireOAuthOnly}, err
}

func (g providerGroupReferences) ActiveGroups(ctx context.Context, platform string) ([]provider.GroupReference, error) {
	rows, err := g.store.ListActive(ctx)
	if rows == nil {
		return nil, err
	}
	out := make([]provider.GroupReference, len(rows))
	for i, v := range rows {
		out[i] = provider.GroupReference{ID: v.ID, Name: v.Name, RequireOAuthOnly: v.RequireOAuthOnly}
	}
	return out, err
}

func (g providerGroupReferences) ValidateGroups(ctx context.Context, ids []int64) error {
	return routing.ValidateGroupIDs(ctx, g.store, ids)
}
