package provider

import (
	"github.com/TokenFlux/TokenRouter/internal/egress"
)

// HeaderOverrides 由提供商决定适用性，名称/值安全规则由 egress 唯一执行。
// 结果不持有 credentials 内的 map，也不在读路径更新共享缓存字段。
func (a *Record) HeaderOverrides() map[string]string {
	if !a.IsHeaderOverrideEnabled() {
		return nil
	}
	return egress.ResolveHeaderOverrides(StringMappingFromRaw(a.Credentials["header_overrides"]))
}
