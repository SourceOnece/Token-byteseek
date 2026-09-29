package app

import (
	"log"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/infra/timingwheel"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	providerpostgres "github.com/TokenFlux/TokenRouter/internal/provider/postgres"
)

// provideProviderDeferred 固定同一个提供商存储与时间轮，构造无定时任务副作用。
func provideProviderDeferred(store *providerpostgres.ProviderStore, wheel *timingwheel.Wheel) *provider.DeferredService {
	return provider.NewDeferredService(store, wheel, provider.DeferredOptions{Interval: 10 * time.Second, Now: time.Now, Observe: log.Printf})
}
