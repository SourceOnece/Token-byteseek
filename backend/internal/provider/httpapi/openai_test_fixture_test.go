package httpapi

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/egress"
	"github.com/TokenFlux/TokenRouter/internal/gateway"
	"github.com/TokenFlux/TokenRouter/internal/gateway/media"
	"github.com/TokenFlux/TokenRouter/internal/gateway/provider/modelidentity"
	"github.com/TokenFlux/TokenRouter/internal/infra/httpclient/tlsfingerprint"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/TokenFlux/TokenRouter/internal/upstream/openai"
)

// 夹具只组合真实策略、平台执行与事件用例，不复制测试分支或凭据算法。
func configureOpenAIProbe(executor *provideradapter.OpenAIProviderTest) {
	policy := &provideradapter.OpenAIProbePolicy{Available: true, DefaultBrowserUserAgent: gateway.DefaultOpenAICodexUserAgent}
	if executor.Store != nil {
		policy.Read = executor.Store.GetByID
	}
	if executor.Prepare == nil {
		executor.Prepare = policy.Prepare
	}
	if executor.ApplyRouting == nil {
		executor.ApplyRouting = policy.ApplyTestRouting
	}
	if executor.ResolveTLS == nil {
		executor.ResolveTLS = policy.ResolveTestTLS
	}
	executor.ModelRules = openai.CodexModelRules{ImageOnly: media.IsImageGenerationModel, LastSegment: capability.LastOpenAIModelSegment, CanonicalAlias: capability.CanonicalizeOpenAIModelAliasSpelling, KnownModel: modelidentity.NormalizeOpenAI, SupportsEffort: capability.OpenAIModelSupportsReasoningEffort}
}

func executeOpenAIProbe(t *testing.T, executor *provideradapter.OpenAIProviderTest, output *openAIProbeOutput, value *provider.Record, model, prompt, mode string, types ...string) error {
	t.Helper()
	configureOpenAIProbe(executor)
	run := provideradapter.NewTestRun(output.Request.Context(), output.Request.Header, NewTestEventSink(output.recorder))
	defer run.Cancel()
	return run.Result(executor.Execute(run, value, model, prompt, mode, types...))
}

func executeOpenAIImageAPIKeyProbe(t *testing.T, executor *provideradapter.OpenAIProviderTest, output *openAIProbeOutput, ctx context.Context, value *provider.Record, model, prompt string) error {
	t.Helper()
	configureOpenAIProbe(executor)
	run := provideradapter.NewTestRun(output.Request.Context(), output.Request.Header, NewTestEventSink(output.recorder))
	defer run.Cancel()
	return run.Result(executor.ExecuteImageAPIKey(run, ctx, value, model, prompt))
}

func executeOpenAIImageOAuthProbe(t *testing.T, executor *provideradapter.OpenAIProviderTest, output *openAIProbeOutput, ctx context.Context, value *provider.Record, model, prompt string) error {
	t.Helper()
	configureOpenAIProbe(executor)
	run := provideradapter.NewTestRun(output.Request.Context(), output.Request.Header, NewTestEventSink(output.recorder))
	defer run.Cancel()
	return run.Result(executor.ExecuteImageOAuth(run, ctx, value, model, prompt))
}

func openAIProbeCore(executor *provideradapter.OpenAIProviderTest) *provider.TestService {
	configureOpenAIProbe(executor)
	targets := &provideradapter.TestTargets{Read: executor.Store.GetByID, OpenAI: executor, CN: &provideradapter.CNProviderTest{Transport: executor.Transport, Store: executor.Store, ValidateURL: executor.ValidateURL, Responses: executor}}
	return provider.NewTestService(targets, provider.TestOptions{Now: time.Now, Error: func(string) {}, WriteError: func(error) {}})
}

func executeOpenAIProbeRequest(t *testing.T, executor *provideradapter.OpenAIProviderTest, output *openAIProbeOutput, id int64, model, prompt, mode string) error {
	t.Helper()
	return openAIProbeCore(executor).Test(output.Request.Context(), provider.TestRequest{ProviderID: id, Model: model, Prompt: prompt, Mode: mode}, NewTestEventSink(output.recorder))
}

type openAIProbeTransport struct {
	requests       []*http.Request
	bodies         [][]byte
	responses      []*http.Response
	resp           *http.Response
	err            error
	lastReq        *http.Request
	lastBody       []byte
	lastTLSProfile *tlsfingerprint.Profile
}

func (f *openAIProbeTransport) DoWithTLS(req *http.Request, _ string, _ int64, _ int, profile *tlsfingerprint.Profile) (*http.Response, error) {
	f.lastReq = req
	f.requests = append(f.requests, req)
	f.lastTLSProfile = profile
	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		f.lastBody = body
		f.bodies = append(f.bodies, append([]byte(nil), body...))
		if err := req.Body.Close(); err != nil {
			return nil, err
		}
		req.Body = io.NopCloser(bytes.NewReader(body))
	}
	if f.err != nil {
		return nil, f.err
	}
	if len(f.responses) > 0 {
		response := f.responses[0]
		f.responses = f.responses[1:]
		return response, nil
	}
	return f.resp, nil
}

// 固定协议测试沿用真实原生分派，存储只负责提供目标记录。
func newCNProtocolTestCore(read func(context.Context, int64) (*provider.Record, error), transport provideradapter.QoderTransport) *provider.TestService {
	policy := egress.OperatorURLPolicy{}
	openaiExecutor := &provideradapter.OpenAIProviderTest{Transport: transport, ValidateURL: policy.Validate}
	configureOpenAIProbe(openaiExecutor)
	targets := &provideradapter.TestTargets{Read: read, CN: &provideradapter.CNProviderTest{Transport: transport, ValidateURL: policy.Validate, Responses: openaiExecutor}}
	return provider.NewTestService(targets, provider.TestOptions{Now: time.Now, Error: func(string) {}, WriteError: func(error) {}})
}
