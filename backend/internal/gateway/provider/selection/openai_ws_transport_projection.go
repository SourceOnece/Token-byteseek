package selection

import (
	"github.com/TokenFlux/TokenRouter/internal/egress"
	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
)

// resolveOpenAIWSTransport 按原时机投影当前提供商和启动配置，传输规则只有原生实现。
func (s *Compatible) ResolveTransport(value *gatewayprovider.ExecutionProvider) egress.OpenAIWSProtocolDecision {
	view := gatewayprovider.ExecutionProtocolRecord(value)
	if view !=
		nil {
		view.Concurrency = value.Record.Concurrency
	}
	var options *egress.OpenAIWSOptions

	mode := ""
	if s != nil {
		options = s.options.WS
		mode = s.options.WSIngressMode
	}
	decision := gatewayprovider.ResolveOpenAIWSTransport(view, options, mode)
	if value != nil && s != nil && s.tickets != nil && s.tickets.VerifiedFlow(value.Record.ID) && (decision.Transport == egress.OpenAIUpstreamTransportResponsesWebsocketV2 || decision.Reason == "ws_v2_mode_http_bridge") {
		return egress.OpenAIWSHTTPDecision("ws_v2_mode_http_bridge")
	}
	return decision
}
