package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/stretchr/testify/require"
)

// 模型不存在的 400/404 不能证明 Responses 端点不支持，必须保留未知或旧事实。
func TestResponsesProbeModelUnavailableIsInconclusive(t *testing.T) {
	for _, body := range []string{
		`{"error":{"type":"model_not_found","message":"Model codex-auto-review is not supported by any configured account in this group"}}`,
		`{"error":{"code":"model_not_available"}}`,
		`{"error":{"message":"The model missing does not exist"}}`,
	} {
		require.False(t, responsesProbeVerdictIsConclusive(404, []byte(body)))
		require.True(t, decideResponsesProbeSupport(404, []byte(body)))
		updates := make(chan map[string]any, 1)
		account := Account{ID: 96, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "test", "base_url": "https://upstream.example"}}
		repo := &snapshotUpdateAccountRepo{stubOpenAIAccountRepo: stubOpenAIAccountRepo{accounts: []Account{account}}, updateExtraCalls: updates}
		svc := &AccountTestService{accountRepo: repo, cfg: &config.Config{}, httpUpstream: &httpUpstreamRecorder{resp: &http.Response{
			StatusCode: 404, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body)),
		}}}
		svc.ProbeOpenAIAPIKeyResponsesSupport(context.Background(), account.ID)
		select {
		case <-updates:
			t.Fatal("a model failure must not overwrite the account endpoint capability")
		default:
		}
	}
	// 普通端点 404/405 仍视为接口不存在。
	require.True(t, responsesProbeVerdictIsConclusive(404, []byte("404 page not found")))
	require.False(t, decideResponsesProbeSupport(404, []byte("404 page not found")))
	require.False(t, decideResponsesProbeSupport(405, nil))
}

func TestSelectResponsesProbeModelPrefersGeneralTextModel(t *testing.T) {
	account := &Account{Credentials: map[string]any{"model_mapping": map[string]any{
		"codex-auto-review": "codex-auto-review", "gpt-image-2": "gpt-image-2", "gpt-5.5": "gpt-5.5",
	}}}
	require.Equal(t, "gpt-5.5", selectResponsesProbeModel(account))
}
