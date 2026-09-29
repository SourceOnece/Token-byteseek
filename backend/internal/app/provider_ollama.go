package app

import (
	"context"
	"database/sql"
	"math/rand/v2"
	"net/http"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/identity"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"

	"github.com/TokenFlux/TokenRouter/internal/infra/httpclient"
	"github.com/TokenFlux/TokenRouter/internal/infra/telemetry/logging"

	"github.com/TokenFlux/TokenRouter/internal/config"
	providerpostgres "github.com/TokenFlux/TokenRouter/internal/provider/postgres"

	postgresinfra "github.com/TokenFlux/TokenRouter/internal/infra/postgres"
	"github.com/google/uuid"
)

// provideOllamaUsage 直接绑定唯一提供商存储、加密器、动态设置与供应商执行句柄。
func provideOllamaUsage(store *providerpostgres.ProviderStore, upstream httpclient.UpstreamTransport, settings *provider.RuntimeSettings, cipher identity.SecretEncryptor, cfg *config.Config, leader provider.CNMonitorLeader, db *sql.DB) *provider.OllamaCloudUsageService {
	var do func(*http.Request, string, int64, int) (*http.Response, error)
	if upstream != nil {
		do = upstream.Do
	}
	var advisory func(context.Context, string) (func(), bool)
	if db != nil {
		advisory = func(ctx context.Context, key string) (func(), bool) {
			return postgresinfra.TryAcquireDBAdvisoryLock(ctx, db, postgresinfra.HashAdvisoryLockID(key))
		}
	}
	options := provider.OllamaUsageOptions{EncryptionKeyConfigured: cfg != nil && cfg.Totp.EncryptionKeyConfigured, Now: time.Now, Jitter: rand.Int64N, InstanceID: uuid.NewString(), Fetch: provideradapter.OllamaUsageFetcher(do), Log: func(format string, args ...any) {
		logging.LegacyPrintf("service.ollama_cloud_usage", format, args...)
	}, Lease: func(ctx context.Context, key, owner string, ttl time.Duration) (func(), bool) {
		return provider.AcquireSingletonLease(ctx, leader, advisory, key, owner, ttl)
	}}
	return provider.NewOllamaCloudUsageService(store, settings, cipher, options)
}
