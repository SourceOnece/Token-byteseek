package app

import (
	"context"

	"github.com/TokenFlux/TokenRouter/internal/app/lifecycle"
	"github.com/TokenFlux/TokenRouter/internal/infra/telemetry/logging"
	"github.com/TokenFlux/TokenRouter/internal/modelcatalog/provider"
)

// modelCatalogRuntimeReady 确保应用构造完成前登记统一模型目录的生命周期。
type modelCatalogRuntimeReady struct{}

// provideModelCatalogRuntime 先加载目录，再启动同步；初始化失败时保持原有降级行为。
// @project-doc docs/architecture/system_architecture.md#startup_and_shutdown
func provideModelCatalogRuntime(catalog *provider.Service, manager *lifecycle.Manager) *modelCatalogRuntimeReady {
	catalogReady := false
	manager.Register(lifecycle.Hook{Name: "ModelCatalogInitialization", StartOrder: 188, Start: func(context.Context) error {
		if catalog == nil {
			return nil
		}
		if err := catalog.Initialize(); err != nil {
			logging.LegacyPrintf("service.modelcatalog", "[Service] Warning: Model catalog initialization failed: %v", err)
			return nil
		}
		catalogReady = true
		return nil
	}})

	manager.Register(lifecycle.Hook{Name: "ModelCatalogService", StartOrder: 980, StopOrder: 20, Start: func(ctx context.Context) error {
		if catalog != nil && catalogReady {
			catalog.Start()
		}
		return nil
	}, Stop: func(ctx context.Context) error {
		if catalog != nil {
			catalog.Stop()
		}
		return nil
	}})
	return &modelCatalogRuntimeReady{}
}
