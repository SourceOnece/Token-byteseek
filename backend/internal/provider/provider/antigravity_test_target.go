package provider

import (
	"context"
	"net/http"
	"strings"

	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/upstream/antigravity"
)

// AntigravityProviderTest 保留静态凭据分流，原生重试和额度规则通过既有探测端口执行。
type AntigravityProviderTest struct {
	Gemini    *GeminiProviderTest
	Anthropic *AnthropicProviderTest
	Probe     func(context.Context, *provider.Record, provider.PreparedTestRequest) (*antigravity.TestConnectionResult, error)
}

func (s *AntigravityProviderTest) Target(value *provider.Record) provider.TestTarget {
	return antigravityTestTarget{executor: s, record: value}
}

type antigravityTestTarget struct {
	executor *AntigravityProviderTest
	record   *provider.Record
}

func (t antigravityTestTarget) Information() provider.TestTargetInfo {
	return provider.TestTargetInfo{ProviderSnapshot: t.record.RoutingSnapshot(), APIProtocol: (provider.ProtocolTarget{Record: t.record}).GetAPIProtocol()}
}

func (t antigravityTestTarget) Execute(ctx context.Context, request provider.PreparedTestRequest, sink provider.TestEventSink) error {
	image, explicit := provider.ProviderTestTypeFromArgs(request.TestType)
	if t.record.Type == provider.ProviderTypeAPIKey {
		if (explicit && image == provider.ProviderTestTypeImage) || strings.HasPrefix(strings.ToLower(request.Model), "gemini-") {
			return t.executor.Gemini.Target(t.record).Execute(ctx, request, sink)
		}
		return t.executor.Anthropic.Target(t.record).Execute(ctx, request, sink)
	}
	run := NewTestRun(ctx, make(http.Header), sink)
	defer run.Cancel()
	if explicit && image == provider.ProviderTestTypeImage {
		return run.Result((TestStreamOutput{}).Error(run, "Image tests are not supported for this Antigravity provider type"))
	}
	if request.Model == "" {
		request.Model = "claude-sonnet-4-5"
	}
	if t.executor.Probe == nil {
		return run.Result((TestStreamOutput{}).Error(run, "Antigravity gateway service not configured"))
	}
	run.Begin(true)
	(TestStreamOutput{}).SendEvent(run, provider.TestEvent{Type: "test_start", Model: request.Model})
	result, err := t.executor.Probe(run.Context, t.record, request)
	if err != nil {
		return run.Result((TestStreamOutput{}).Error(run, err.Error()))
	}
	if result.Text != "" {
		(TestStreamOutput{}).SendEvent(run, provider.TestEvent{Type: "content", Text: result.Text})
	}
	(TestStreamOutput{}).SendEvent(run, provider.TestEvent{Type: "test_complete", Success: true})
	return run.Result(nil)
}
