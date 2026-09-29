package httpapi

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
	"github.com/TokenFlux/TokenRouter/internal/upstream/openai"

	"github.com/TokenFlux/TokenRouter/internal/billing"
	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/stretchr/testify/require"
)

func TestProviderTestServiceOpenAICompactAgentIdentityUsesFreshAssertion(t *testing.T) {
	key, privateKey := newProbeAgentKey(t)
	provider := providercore.Record{
		ID:          21,
		Name:        "agent-identity",
		Platform:    capability.PlatformOpenAI,
		Type:        capability.ProviderTypeOAuth,
		Status:      billing.StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{
			"auth_mode":                  providercore.OpenAIAuthModeAgentIdentity,
			"agent_runtime_id":           key.RuntimeID,
			"agent_private_key":          privateKey,
			"task_id":                    key.TaskID,
			"chatgpt_account_id":         "provider-agent-test",
			"chatgpt_account_is_fedramp": true,
		},
	}
	repo := &probeAgentStore{provider: &provider}
	upstream := &openAIProbeTransport{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(compactionTestV2SSESuccessBody)),
	}}
	svc := &provideradapter.OpenAIProviderTest{Store: repo, Transport: upstream, EnsureTask: probeAgentTasks(repo, "", nil).Ensure}

	rec := httptest.NewRecorder()
	c := &openAIProbeOutput{recorder: rec}
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/providers/21/test", bytes.NewReader(nil))

	require.NoError(t, executeOpenAIProbeRequest(t, svc, c, provider.ID, "gpt-5.4", "", providercore.ProviderTestModeCompact))
	require.Equal(t, "AgentAssertion", strings.SplitN(upstream.lastReq.Header.Get("Authorization"), " ", 2)[0])
	require.Equal(t, "provider-agent-test", upstream.lastReq.Header.Get("chatgpt-account-id"))
	require.Equal(t, "true", upstream.lastReq.Header.Get("x-openai-fedramp"))
	require.NotContains(t, upstream.lastReq.Header.Get("Authorization"), privateKey)
}

func TestProviderTestServiceOpenAICompactAgentIdentityRecoversInvalidTaskOnce(t *testing.T) {
	key, privateKey := newProbeAgentKey(t)
	provider := &providercore.Record{
		ID:          22,
		Name:        "agent-identity-recovery",
		Platform:    capability.PlatformOpenAI,
		Type:        capability.ProviderTypeOAuth,
		Status:      billing.StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{
			"auth_mode":          providercore.OpenAIAuthModeAgentIdentity,
			"agent_runtime_id":   key.RuntimeID,
			"agent_private_key":  privateKey,
			"task_id":            "task-compact-old",
			"chatgpt_account_id": "provider-agent-compact-recovery",
		},
	}
	repo := &probeAgentStore{provider: provider}
	registerCalls := 0
	registerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		registerCalls++
		_, _ = io.WriteString(w, `{"task_id":"task-compact-new"}`)
	}))
	defer registerServer.Close()

	upstream := &openAIProbeTransport{responses: []*http.Response{
		{StatusCode: http.StatusUnauthorized, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"error":{"code":"invalid_task_id"}}`))},
		{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(compactionTestV2SSESuccessBody))},
	}}
	invalidator := &probeAgentInvalidations{}
	svc := &provideradapter.OpenAIProviderTest{Store: repo, Transport: upstream, EnsureTask: probeAgentTasks(repo, registerServer.URL, invalidator).Ensure}
	rec := httptest.NewRecorder()
	c := &openAIProbeOutput{recorder: rec}
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/providers/22/test", bytes.NewReader(nil))

	require.NoError(t, executeOpenAIProbeRequest(t, svc, c, provider.ID, "gpt-5.4", "", providercore.ProviderTestModeCompact))
	require.Equal(t, 1, registerCalls)
	require.Len(t, upstream.requests, 2)
	require.Equal(t, "task-compact-new", provider.GetCredential("task_id"))
	require.Equal(t, 0, repo.setErrorCalls)
	require.Equal(t, []int64{provider.ID}, invalidator.providerIDs)
}

// 本地密钥仅用于真实签名器，测试不访问外部认证服务。
func newProbeAgentKey(t *testing.T) (openai.AgentIdentityKey, string) {
	t.Helper()
	_, private, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(private)
	require.NoError(t, err)
	return openai.AgentIdentityKey{RuntimeID: "runtime-test", TaskID: "task-test", PrivateKey: private}, base64.StdEncoding.EncodeToString(der)
}

type probeAgentStore struct {
	provideradapter.OpenAIProviderTestStore
	provider      *providercore.Record
	setErrorCalls int
}

func (r *probeAgentStore) GetByID(context.Context, int64) (*providercore.Record, error) {
	return providercore.CloneRecord(r.provider), nil
}

func (r *probeAgentStore) UpdateCredentials(_ context.Context, _ int64, credentials map[string]any) error {
	r.provider.Credentials = credentials
	return nil
}

func (*probeAgentStore) Update(context.Context, *providercore.Record) error {
	return fmt.Errorf("unexpected full provider update")
}
func (*probeAgentStore) UpdateExtra(context.Context, int64, map[string]any) error { return nil }
func (r *probeAgentStore) SetError(context.Context, int64, string) error {
	r.setErrorCalls++
	return nil
}

type probeAgentInvalidations struct{ providerIDs []int64 }

func (r *probeAgentInvalidations) Invalidate(id int64) { r.providerIDs = append(r.providerIDs, id) }
func probeAgentTasks(store *probeAgentStore, endpoint string, invalidator *probeAgentInvalidations) *provideradapter.ProbeTasks {
	options := providercore.OpenAITaskOptions{
		Read: store.GetByID,
		Register: func(ctx context.Context, value *providercore.Record) (string, error) {
			return provideradapter.RegisterAgentIdentityTask(ctx, value, endpoint)
		},
		Persist: func(ctx context.Context, value *providercore.Record, credentials map[string]any) error {
			_, err := providercore.PersistCredentials(ctx, store, value, credentials, nil)
			return err
		},
	}
	if invalidator != nil {
		options.Invalidate = invalidator.Invalidate
	}
	return &provideradapter.ProbeTasks{Coordinator: &providercore.OpenAITaskCoordinator{}, Options: options}
}
