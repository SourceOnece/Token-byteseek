package selection

import "github.com/TokenFlux/TokenRouter/internal/scheduler"

// loads 为调度测试投影候选负载，评分与获取槽位仍使用实际选择器。
func (g *projectionScope) loads(values []providerWithLoad) []scheduler.FlowLoad {
	if values == nil {
		return nil
	}
	out := make([]scheduler.FlowLoad, len(values))
	for i, a := range values {
		out[i] = scheduler.FlowLoad{Provider: g.provider(a.provider), LoadInfo: a.loadInfo}
	}
	return out
}
