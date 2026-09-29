package accessview

import (
	"github.com/TokenFlux/TokenRouter/internal/scheduler/policy"
)

// CloneGroupAdvancedSchedulerOverrides 委托所属模块的唯一实现。
func CloneGroupAdvancedSchedulerOverrides(overrides GroupAdvancedSchedulerOverrides) GroupAdvancedSchedulerOverrides {
	return policy.CloneGroupAdvancedSchedulerOverrides(overrides)
}
