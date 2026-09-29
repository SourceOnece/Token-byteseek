package app

import (
	"log"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/provider"
	providerpostgres "github.com/TokenFlux/TokenRouter/internal/provider/postgres"
)

// provideProviderExpiry 为原一分钟周期注入同一提供商存储，启动由维护生命周期登记。
func provideProviderExpiry(store *providerpostgres.ProviderStore) *provider.ExpiryService {
	return provider.NewExpiryService(store, provider.ExpiryOptions{Interval: time.Minute, Now: time.Now, Observe: log.Printf})
}
