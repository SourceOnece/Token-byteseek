package app

import (
	"time"

	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/TokenFlux/TokenRouter/internal/infra/telemetry/logging"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
)

// provideScheduledTests 固定唯一计划/结果用例，构造无定时器或后台任务。
func provideScheduledTests(plans provider.ScheduledTestPlanRepository, results provider.ScheduledTestResultRepository) *provider.ScheduledTestService {
	return provider.NewScheduledTestService(plans, results, provider.ScheduledTestOptions{Now: time.Now, NextRun: provideradapter.NextScheduledTestRun})
}

func provideScheduledTestRunner(plans provider.ScheduledTestPlanRepository, scheduled *provider.ScheduledTestService, tests *provider.TestService, recovery *provider.RecoveryService, cfg *config.Config) *provider.ScheduledTestRunnerService {
	location := time.Local
	if parsed, err := time.LoadLocation(cfg.Timezone); err == nil && parsed != nil {
		location = parsed
	}
	return provider.NewScheduledTestRunnerService(plans, scheduled, tests, provider.ScheduledRunnerOptions{Schedule: provideradapter.NewScheduledCron(location), Now: time.Now, NextRun: provideradapter.NextScheduledTestRun, Offset: 10 * time.Second, Observe: func(format string, args ...any) {
		logging.LegacyPrintf("service.scheduled_test_runner", format, args...)
	}, Recover: recovery.RecoverProviderAfterSuccessfulTest})
}
