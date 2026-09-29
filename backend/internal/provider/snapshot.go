package provider

import (
	"slices"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
)

// ProviderSnapshot 是候选判断需要的身份、资格与运行投影，不携带凭据或管理 Extra。
// 模型重写及平台凭据读取仍由各自显式入口提供，不允许从快照反查完整记录。
type ProviderSnapshot struct {
	// ModelPolicy 仅在实际模型匹配点装配，不随协议预检提前读取动态默认值。
	ModelPolicy ModelRoutingSnapshot `json:"-"`
	// 模型协议规则只含模型模式和协议名，不包含凭据。
	ModelProtocolRules   []OpenCodeGoProtocolRule `json:"-"`
	ModelProtocolDefault string                   `json:"-"`
	ID                   int64
	ParentProviderID     *int64
	Platform             string
	Type                 string
	AuthMode             string
	EnabledProtocols     []capability.ProtocolID
	Status               string
	Schedulable          bool
	Concurrency          int
	Priority             int
	ExpiresAt            *time.Time
}

// RoutingSnapshot 返回请求独立副本；协议缺省解析仍由提供商配置规则唯一负责。
func (r *Record) RoutingSnapshot() ProviderSnapshot {
	if r == nil {
		return ProviderSnapshot{}
	}
	snapshot := ProviderSnapshot{
		ID: r.ID, Platform: r.Platform, Type: r.Type, AuthMode: ProtocolAuthMode(r),
		EnabledProtocols: slices.Clone(r.UpstreamProtocols()), Status: r.Status, Schedulable: r.Schedulable,
		Concurrency: r.Concurrency, Priority: r.Priority,
	}
	if r.ParentProviderID != nil {
		id := *r.ParentProviderID
		snapshot.ParentProviderID = &id
	}
	if r.ExpiresAt != nil {
		at := *r.ExpiresAt
		snapshot.ExpiresAt = &at
	}
	if r.IsOpenCodeGo() {
		rules, exists := r.openCodeGoProtocolRules()
		if !exists {
			rules = defaultOpenCodeProtocolRules(r.GetOpenCodeAccountMode())
		}
		snapshot.ModelProtocolRules = slices.Clone(rules)
		snapshot.ModelProtocolDefault = APIProtocolChatCompletions
	}
	return snapshot
}

// 候选按最终模型选择 OpenCode 原生端点，同时服从管理员启用的协议集合。
func (s ProviderSnapshot) ProtocolsForModel(model string) capability.ProviderProtocols {
	p := s.Protocols()
	if s.ModelProtocolDefault == "" || model == "" || len(s.EnabledProtocols) == 1 {
		return p
	}
	model, _ = s.ModelPolicy.Resolve(model)
	chosen := matchOpenCodeGoProtocolRules(model, s.ModelProtocolRules)
	protocol := capability.ProtocolOpenAIChatCompletions
	if chosen == APIProtocolAnthropic {
		protocol = capability.ProtocolAnthropicMessages
	}
	if chosen == APIProtocolResponses {
		protocol = capability.ProtocolOpenAIResponses
	}
	p.Enabled = nil
	if slices.Contains(s.EnabledProtocols, protocol) {
		p.Enabled = []capability.ProtocolID{protocol}
	}
	return p
}

// Protocols 仅把已经解析好的能力传给纯目录，不暴露提供商存储结构。
func (s ProviderSnapshot) Protocols() capability.ProviderProtocols {
	return capability.ProviderProtocols{Platform: s.Platform, Type: s.Type, AuthMode: s.AuthMode, Enabled: slices.Clone(s.EnabledProtocols)}
}
