//go:build wireinject

package app

import (
	provider "github.com/TokenFlux/TokenRouter/internal/egress/provider"

	egress "github.com/TokenFlux/TokenRouter/internal/egress"

	egresshttp "github.com/TokenFlux/TokenRouter/internal/egress/httpapi"

	egresspostgres "github.com/TokenFlux/TokenRouter/internal/egress/postgres"

	egressredis "github.com/TokenFlux/TokenRouter/internal/egress/rediscache"

	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
	"github.com/google/wire"
)

// egress 模块的组合根登记；这里只分组原 provider，不创建资源或复制业务实现。
var egressAssemblyProviders = wire.NewSet(
	wire.Bind(new(provideradapter.OpenAITokenProfileResolver), new(*provider.TLSProfiles)),
	wire.Bind(new(provideradapter.OpenAITokenRouterReader), new(*egress.TLSFingerprintRouterService)),
	provideProxyExpiry,
	provideProxyTransfer,
	provideProxyHTTP,
	provideEgressProxyStore,
	wire.Bind(new(egress.ProxyRepository), new(*egresspostgres.ProxyStore)),
	provideEgressProbe,
	provideEgressAdmin,
	egressredis.NewProxyLatencyCache,
	egresspostgres.NewTLSFingerprintProfileRepository,
	egresspostgres.NewTLSFingerprintRouterRepository,
	egressredis.NewTLSFingerprintProfileCache,
	egressredis.NewTLSFingerprintRouterCache,
	provideEgressProfiles,
	provider.NewTLSProfiles,
	provideEgressRouters,
	provideEgressCollector,
	provideEgressProfileHTTP,
	egresshttp.NewTLSFingerprintRouterHandler,
)
