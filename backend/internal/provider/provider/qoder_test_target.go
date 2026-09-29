package provider

import (
	"context"
	"net/http"

	"github.com/TokenFlux/TokenRouter/internal/provider"
)

// Target 把已读取的提供商封装为一次测试句柄，公开信息不暴露凭据。
func (s *QoderProviderTest) Target(value *provider.Record) provider.TestTarget {
	return qoderTestTarget{executor: s, record: value}
}

type qoderTestTarget struct {
	executor *QoderProviderTest
	record   *provider.Record
}

func (t qoderTestTarget) Information() provider.TestTargetInfo {
	return provider.TestTargetInfo{ProviderSnapshot: t.record.RoutingSnapshot(), APIProtocol: (provider.ProtocolTarget{Record: t.record}).GetAPIProtocol()}
}

func (t qoderTestTarget) Execute(ctx context.Context, request provider.PreparedTestRequest, sink provider.TestEventSink) error {
	headers := make(http.Header)
	headers.Set("User-Agent", request.UserAgent)
	headers.Set("originator", request.Originator)
	run := NewTestRun(ctx, headers, sink)
	defer run.Cancel()
	return run.Result(t.executor.Execute(run, t.record, request.Model, request.Prompt))
}
