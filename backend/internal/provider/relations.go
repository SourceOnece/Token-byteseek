package provider

import (
	"time"

	"github.com/TokenFlux/TokenRouter/internal/routing/accessview"
)

// GroupMembership 只表达提供商与分组的原关联，不引入分组业务或调度优先级。
type GroupMembership struct {
	ProviderID int64
	GroupID    int64
	CreatedAt  time.Time
	Provider   *Record
	Group      *accessview.GroupConfig
}
