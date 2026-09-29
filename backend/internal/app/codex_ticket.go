package app

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	dbent "github.com/TokenFlux/TokenRouter/ent"
	"github.com/TokenFlux/TokenRouter/internal/app/lifecycle"
	"github.com/TokenFlux/TokenRouter/internal/codexticket"
	ticketpostgres "github.com/TokenFlux/TokenRouter/internal/codexticket/postgres"
	ticketredis "github.com/TokenFlux/TokenRouter/internal/codexticket/rediscache"
	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/TokenFlux/TokenRouter/internal/egress"
	egressprovider "github.com/TokenFlux/TokenRouter/internal/egress/provider"
	"github.com/TokenFlux/TokenRouter/internal/identity"
	"github.com/TokenFlux/TokenRouter/internal/infra/httpclient"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	providerpostgres "github.com/TokenFlux/TokenRouter/internal/provider/postgres"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
	"github.com/TokenFlux/TokenRouter/internal/quality"
	qualitypostgres "github.com/TokenFlux/TokenRouter/internal/quality/postgres"
	"github.com/TokenFlux/TokenRouter/internal/scheduler"
	"github.com/TokenFlux/TokenRouter/internal/scheduler/rediscache/codec"
	"github.com/TokenFlux/TokenRouter/internal/settings"
	"github.com/redis/go-redis/v9"
)

// 单一组合根装配票据扩展，复用生产凭据、槽位、连接池及快照，不启动旧服务。
type ticketRuntime struct {
	store       *ticketpostgres.Store
	snapshots   *scheduler.SnapshotService
	credentials *provider.OpenAIExecutionCredentials
	concurrency *scheduler.ConcurrencyService
	transport   httpclient.UpstreamTransport
	profiles    *egressprovider.TLSProfiles
	slotTTL     time.Duration
}

func (r *ticketRuntime) TicketAccountStore() codexticket.CodexTicketAccountStore { return r.store }
func (r *ticketRuntime) TicketSnapshotAccount(ctx context.Context, id int64) (*provider.Record, error) {
	value, err := r.snapshots.GetProvider(ctx, id)
	if err != nil {
		return nil, err
	}
	return codec.RecordValue(value)
}
func (r *ticketRuntime) TicketModelSupported(value *provider.Record, model string) bool {
	// 票据键已使用最终出站模型，不能再执行一次账号映射，否则可能误报缺票。
	return value.FinalModelWhitelisted(model, provideradapter.ModelDefaults(), provideradapter.ModelRules(value))
}
func (r *ticketRuntime) GetAccessToken(ctx context.Context, value *provider.Record) (string, string, error) {
	return r.credentials.Resolve(ctx, value)
}
func (r *ticketRuntime) TicketAccountHeaders(ctx context.Context, headers http.Header, value *provider.Record) error {
	resolved, err := provider.ResolveCredentialRecord(ctx, r.store.GetByID, value)
	if err != nil {
		return err
	}
	provideradapter.SetChatGPTAccountHeaders(headers, resolved)
	return nil
}
func (r *ticketRuntime) TicketSlotTTL() time.Duration { return r.slotTTL }
func (r *ticketRuntime) AcquireTicketSlot(ctx context.Context, value *provider.Record) (*scheduler.AcquireResult, error) {
	return r.concurrency.AcquireProviderSlot(ctx, value.ID, value.Concurrency)
}
func (r *ticketRuntime) TicketTransportAvailable() bool { return r.transport != nil }
func (r *ticketRuntime) SendTicketHarvest(req *http.Request, proxy string, value *provider.Record) (*http.Response, error) {
	return r.transport.Do(req, proxy, value.ID, value.Concurrency)
}
func (r *ticketRuntime) SendTicketVerification(req *http.Request, proxy string, value *provider.Record) (*http.Response, error) {
	profile := r.profiles.ResolveRequestTLS(egress.TLSSelection{Enabled: value.IsTLSFingerprintEnabled(), DirectProfileID: value.GetTLSFingerprintProfileID()})
	return r.transport.DoWithTLS(req, proxy, value.ID, value.Concurrency, profile)
}

func provideCodexTickets(client *dbent.Client, db *sql.DB, providers *providerpostgres.ProviderStore, settingStore *settings.Store, cache scheduler.SnapshotCache, snapshots *scheduler.SnapshotService, credentials *provider.OpenAIExecutionCredentials, concurrency *scheduler.ConcurrencyService, transport httpclient.UpstreamTransport, profiles *egressprovider.TLSProfiles, proxies egress.ProxyRepository, ip egress.ProxyExitInfoProber, redis *redis.Client, cipher identity.SecretEncryptor, cfg *config.Config, manager *lifecycle.Manager) *codexticket.CodexTicketService {
	store := ticketpostgres.New(client, db, providers, settingStore, newProviderEvents(providers, cache))
	runtime := &ticketRuntime{store: store, snapshots: snapshots, credentials: credentials, concurrency: concurrency, transport: transport, profiles: profiles, slotTTL: time.Duration(cfg.Gateway.ConcurrencySlotTTLMinutes) * time.Minute}
	core := codexticket.NewCodexTicketService(runtime, store, ticketredis.NewCodexTicketCache(redis), cipher, ip, proxies)
	providers.SetListFilter(func(ctx context.Context, query *dbent.ProviderQuery) error {
		qualitypostgres.ApplyCodexQualityFilter(query, quality.CodexQualityFilter(ctx))
		filter := codexticket.CodexTicketFilter(ctx)
		static, length, valid := codexticket.ParseCodexTicketFilter(filter)
		if !valid {
			return errors.New("无效的票据筛选")
		}
		if length == 0 {
			ticketpostgres.ApplyCodexTicketFilter(query, filter)
			return nil
		}
		read, cancel := context.WithTimeout(ctx, 8*time.Second)
		defer cancel()
		candidates := query.Clone()
		ticketpostgres.ApplyCodexTicketFilter(candidates, static)
		entities, err := candidates.All(read)
		if err != nil {
			return err
		}
		values, err := providers.RecordsFromEntities(read, entities)
		if err != nil {
			return err
		}
		ids, err := core.FilterActualTicketLength(read, values, length)
		if err != nil {
			return err
		}
		ticketpostgres.ApplyCodexTicketFilter(query, filter, ids)
		return nil
	})
	manager.Register(lifecycle.Hook{Name: "CodexTickets", StartOrder: 960, StopOrder: 15,
		Start: func(context.Context) error { core.Start(); return nil },
		Stop:  func(context.Context) error { core.Stop(); return nil },
	})
	return core
}
