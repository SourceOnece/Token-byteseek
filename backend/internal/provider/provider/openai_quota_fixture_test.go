package provider

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/egress"
	egressprovider "github.com/TokenFlux/TokenRouter/internal/egress/provider"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/upstream/openai"
	"github.com/stretchr/testify/require"
)

type quotaReadFixture interface {
	GetProvider(context.Context, int64) (*provider.Record, error)
}

// 夹具只组装真实用例与技术工厂，不实现查询、恢复或缓存算法。
type quotaFixture struct {
	*provider.OpenAIQuotaService
	factory *OpenAIQuotaFactory
}

func newQuotaForTest(reader quotaReadFixture, transport QoderTransport, token *provider.OpenAITokenSource, profiles *egressprovider.TLSProfiles, routers OpenAITokenRouterReader) *quotaFixture {
	factory := &OpenAIQuotaFactory{Transport: transport, Profiles: profiles, Routers: routers, Tasks: &provider.OpenAITaskCoordinator{}}
	if proxy, ok := reader.(interface {
		GetProxy(context.Context, int64) (*egress.Proxy, error)
	}); ok {
		factory.Proxy = proxy.GetProxy
	}
	options := provider.OpenAIQuotaOptions{Configured: func() bool { return reader != nil && transport != nil }, Read: reader.GetProvider, Client: factory.Client, RedeemID: openai.GenerateOpenAIQuotaRedeemRequestID, Warn: slog.Warn, Info: slog.Info}
	if token != nil {
		options.Token = token.GetAccessToken
	}
	return &quotaFixture{OpenAIQuotaService: provider.NewOpenAIQuotaService(options), factory: factory}
}

func bindQuotaRepositoryForTest(value *quotaFixture, repo *stubQuotaProviderRepo, baseURL string) {
	value.Options.SaveExtra = repo.UpdateExtra
	value.factory.TaskOptions.Read = repo.GetByID
	value.factory.TaskOptions.Register = func(ctx context.Context, record *provider.Record) (string, error) {
		return RegisterAgentIdentityTask(ctx, record, baseURL)
	}
	value.factory.TaskOptions.Persist = func(ctx context.Context, record *provider.Record, credentials map[string]any) error {
		_, err := provider.PersistCredentials(ctx, repo, record, credentials, slog.Warn)
		return err
	}
}

func (r *stubQuotaProviderRepo) Update(ctx context.Context, value *provider.Record) error {
	return r.UpdateCredentials(ctx, value.ID, value.Credentials)
}

func newOpenAITokenSourceForTest(repo provider.RefreshRepository, cache provider.AccessTokenCache, _ *provider.OpenAIAuthorization) *provider.OpenAITokenSource {
	return &provider.OpenAITokenSource{Repository: repo, Cache: cache, Metrics: &provider.OpenAITokenMetricsStore{}, Policy: provider.OpenAIProviderRefreshPolicy(), Debug: slog.Debug, Warn: slog.Warn}
}

func decodeAgentAssertionTask(t *testing.T, header string) string {
	t.Helper()
	encoded := strings.TrimPrefix(header, "AgentAssertion ")
	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	require.NoError(t, err)
	var envelope struct {
		TaskID string `json:"task_id"`
	}
	require.NoError(t, json.Unmarshal(decoded, &envelope))
	return envelope.TaskID
}

type agentIdentityWSInvalidationRecorder struct{ providerIDs []int64 }

func (r *agentIdentityWSInvalidationRecorder) InvalidateAgentIdentityWSConnections(id int64) {
	r.providerIDs = append(r.providerIDs, id)
}
