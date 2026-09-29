package provider

import (
	"context"
	"net/http"

	"github.com/TokenFlux/TokenRouter/internal/provider"
)

// Target 固定本次提供商快照，适配器只根据已经准备好的测试路由执行。
func (s *CNProviderTest) Target(value *provider.Record) provider.TestTarget {
	return cnTestTarget{executor: s, record: value}
}

type cnTestTarget struct {
	executor *CNProviderTest
	record   *provider.Record
}

func (t cnTestTarget) Information() provider.TestTargetInfo {
	return provider.TestTargetInfo{ProviderSnapshot: t.record.RoutingSnapshot(), APIProtocol: (provider.ProtocolTarget{Record: t.record}).GetAPIProtocol()}
}

func (t cnTestTarget) Execute(ctx context.Context, request provider.PreparedTestRequest, sink provider.TestEventSink) error {
	run := NewTestRun(ctx, make(http.Header), sink)
	run.Automatic = request.Automatic
	run.Headers.Set("User-Agent", request.UserAgent)
	defer run.Cancel()
	var err error
	switch request.Route {
	case provider.TestRouteCNAdaptive:
		err = t.executor.ExecuteAdaptive(run, t.record, request.Model, request.Prompt)
	case provider.TestRouteCNResponses:
		err = t.executor.Responses.Execute(run, t.record, request.Model, request.Prompt, request.Mode, request.TestType)
	default:
		err = t.executor.Execute(run, t.record, request.Model, request.Prompt)
	}
	return run.Result(err)
}
