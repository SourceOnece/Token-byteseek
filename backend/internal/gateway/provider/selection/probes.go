package selection

import (
	"context"
	"fmt"

	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/provider"

	"github.com/TokenFlux/TokenRouter/internal/routing"
)

// Probe 连接已绑定的原生选择器与提供商测试，保留平台分发及 Gemini 兜底。
type Probe struct {
	ProviderTest    ProviderTester
	gatewaySvc      *Generic
	openAIGateway   *Compatible
	geminiCompatSvc *Gemini
}

// ProviderTester 只执行已选提供商的原生测试，不持有管理聚合服务。
type ProviderTester interface {
	RunTestBackgroundWithPromptAndUserAgent(context.Context, int64, string, string, string) (*provider.ScheduledTestResult, error)
}

func (s Probe) Select(ctx context.Context, due routing.GroupAvailabilityProbeDueGroup, model string) (int64, error) {
	provider, err := s.selectProbeProvider(ctx, due, model)
	if err != nil {
		return 0, err
	}
	return provider.Record.ID, nil
}

func (s Probe) Test(ctx context.Context, id int64, model, prompt, userAgent string) (*routing.ProbeExecutionResult, error) {
	result, err := s.ProviderTest.RunTestBackgroundWithPromptAndUserAgent(ctx, id, model, prompt, userAgent)
	if result == nil {
		return nil, err
	}
	return &routing.ProbeExecutionResult{Status: result.Status, LatencyMs: result.LatencyMs, ErrorMessage: result.ErrorMessage, StartedAt: result.StartedAt, FinishedAt: result.FinishedAt}, err
}

func (s Probe) selectProbeProvider(ctx context.Context, due routing.GroupAvailabilityProbeDueGroup, modelID string) (*gatewayprovider.ExecutionProvider, error) {
	groupID := due.GroupID
	if s.openAIGateway != nil {
		return s.openAIGateway.SelectProviderForModel(ctx, &groupID, "", modelID)
	}
	if s.gatewaySvc != nil {
		return s.gatewaySvc.SelectProviderForModel(ctx, &groupID, "", modelID)
	}
	return nil, fmt.Errorf("provider selector not configured")
}

// NewProbe 固定平台选择与提供商测试端口，不构造额外测试服务或调度状态。
func NewProbe(test ProviderTester, generic *Generic, compatible *Compatible, gemini *Gemini) Probe {
	return Probe{ProviderTest: test, gatewaySvc: generic, openAIGateway: compatible, geminiCompatSvc: gemini}
}
