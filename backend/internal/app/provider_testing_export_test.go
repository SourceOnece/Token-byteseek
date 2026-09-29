//go:build integration

package app

// NewProviderTestsForTest 仅供外部集成测试调用真实组合根，不扩大生产 API。
var NewProviderTestsForTest = provideProviderTests

// 以下入口仅在集成测试中组合真实平台探测与应用关闭屏障。
var (
	NewAntigravityRetryForTest = provideAntigravityRetry
	NewAntigravityProbeForTest = provideAntigravityProbe
	NewGatewayActivityForTest  = provideGatewayRequestActivity
)

// 快照回放测试复用生产装配与原 outbox 发布器。
var (
	NewSnapshotForTest       = provideSchedulerSnapshot
	NewProviderEventsForTest = newProviderEvents
)

// 集成测试直接使用原生装配及单向兼容绑定。
var (
	NewProviderHealthRuntimeForTest = provideProviderHealthRuntime
	NewUpstreamHealthForTest        = provideUpstreamHealth
)

// 原生完成装配仅向隔离存储测试开放，不增加生产 API。
var (
	NewCompletionRecordersForTest = ProvideGatewayCompletionRecorders
	NewGatewayBillingRatesForTest = provideGatewayBillingRates
)

// 执行提供商集成合同通过真实装配绑定相同存储，不扩大生产接口。
var NewExecutionProviderStoreForTest = provideExecutionProviderStore

// 提供商存储合同复用生产配置投影与事件绑定。
var NewProviderStoreForTest = provideProviderStore

// 模型诊断集成合同直接使用实际组合根，不构造旧执行服务。
var NewModelAvailabilityForTest = provideGatewayModelAvailability
