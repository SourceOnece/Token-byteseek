package routing

import (
	"github.com/TokenFlux/TokenRouter/internal/routing/accessview"
)

// CloneGroup 隔离分组值；提供商等读方复用相同的叶子复制实现。
func CloneGroup(g *Group) *Group {
	return (*Group)(accessview.CloneGroupConfig((*accessview.GroupConfig)(g)))
}
