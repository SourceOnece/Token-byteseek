package ops

// RuntimeSettingsRefreshHealth 读取配置刷新任务的运行与成功失败计数。
func (s *OpsService) RuntimeSettingsRefreshHealth() OpsRuntimeSettingsRefreshHealth {
	if s == nil {
		return OpsRuntimeSettingsRefreshHealth{}
	}
	return OpsRuntimeSettingsRefreshHealth{
		Running:      s.runtimeRefreshRunning.Load(),
		SuccessTotal: s.runtimeRefreshSuccess.Load(),
		FailureTotal: s.runtimeRefreshFailure.Load(),
	}
}

// OpsRuntimeSettingsRefreshHealth 汇总测试所需的配置刷新状态。
type OpsRuntimeSettingsRefreshHealth struct {
	Running      bool   `json:"running"`
	SuccessTotal uint64 `json:"success_total"`
	FailureTotal uint64 `json:"failure_total"`
}
