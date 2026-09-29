package codexticket

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type gatewayTicketCache struct {
	CodexTicketCache
	values       map[string]string
	fail         bool
	observations []string
}

func (c *gatewayTicketCache) Get(_ context.Context, key string) (string, error) {
	if c.fail {
		return "", errors.New("fixture offline")
	}
	return c.values[key], nil
}
func (c *gatewayTicketCache) ObserveTicket(_ context.Context, key, encoded, reason string, revoke bool, _ time.Time) (bool, error) {
	if c.values[key] != encoded {
		return false, nil
	}
	c.observations = append(c.observations, reason)
	if revoke {
		delete(c.values, key)
	}
	return true, nil
}

// 合成加密替身只用于测键隔离；生产仍由应用注入 AES-GCM 实现。
type gatewayTicketCipher struct{}

func (gatewayTicketCipher) Encrypt(raw string) (string, error) {
	return base64.StdEncoding.EncodeToString([]byte(raw)), nil
}
func (gatewayTicketCipher) Decrypt(raw string) (string, error) {
	b, e := base64.StdEncoding.DecodeString(raw)
	return string(b), e
}

func gatewayTicketFixture(t *testing.T) (*CodexTicketService, *Account, *codexTicketConfig, string) {
	t.Helper()
	a := &Account{ID: 41, Platform: "openai", Type: "oauth", Status: "active", Schedulable: true, Credentials: map[string]any{"access_token": "synthetic-a", "chatgpt_account_id": "workspace-a"}}
	cfg := &codexTicketConfig{accountID: a.ID, Enabled: true, Generation: "synthetic-generation", TargetLength: 292, Models: []string{"gpt-6-astra"}, WatchdogMode: "recover", loadedAt: time.Now()}
	cache := &gatewayTicketCache{values: map[string]string{}}
	s := &CodexTicketService{cache: cache, cipher: gatewayTicketCipher{}}
	s.config.Store(cfg)
	state := "gAAAAA" + strings.Repeat("x", 286)
	raw, err := json.Marshal(codexTicketValue{State: state, ExpiresAt: time.Now().Add(time.Hour)})
	require.NoError(t, err)
	encoded, err := s.cipher.Encrypt(string(raw))
	require.NoError(t, err)
	cache.values[codexTicketKey(cfg, a, "gpt-6-astra", "synthetic-a")] = encoded
	return s, a, cfg, state
}

func ticketRequest(t *testing.T) *http.Request {
	t.Helper()
	r, err := http.NewRequest(http.MethodPost, "https://example.test/responses", nil)
	require.NoError(t, err)
	r.Header.Set("Authorization", "Bearer synthetic-a")
	return r
}

func TestTicketHTTPInjectsAndObservesWithoutRewriting(t *testing.T) {
	s, a, _, state := gatewayTicketFixture(t)
	req := ticketRequest(t)
	req.Header["x-codex-turn-state"] = []string{"client-state"}
	require.NoError(t, s.ApplyRequest(context.Background(), a, "gpt-6-astra", req))
	require.Equal(t, state, req.Header.Get(openAICodexTurnStateHeader))
	count := 0
	for key := range req.Header {
		if strings.EqualFold(key, openAICodexTurnStateHeader) {
			count++
		}
	}
	require.Equal(t, 1, count)
	body := "data: {\"type\":\"response.completed\",\"response\":{\"model\":\"gpt-5.6-luna\",\"status\":\"completed\"}}\n\n"
	resp := &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}
	s.ObserveResponse(req, resp)
	read, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, body, string(read))
	cache, ok := s.cache.(*gatewayTicketCache)
	require.True(t, ok)
	require.Equal(t, []string{"model_mismatch"}, cache.observations)
	require.True(t, s.Blocks(context.Background(), a, "gpt-6-astra"))
}

func TestTicketOffAndOutsideScopeLeaveHeadersUntouched(t *testing.T) {
	for _, mode := range []string{"global_off", "other_model", "apikey", "other_platform"} {
		t.Run(mode, func(t *testing.T) {
			s, a, cfg, _ := gatewayTicketFixture(t)
			model := "gpt-6-astra"
			switch mode {
			case "global_off":
				cfg.Enabled = false
			case "other_model":
				model = "gpt-5.6-sol"
			case "apikey":
				a.Type = "apikey"
			case "other_platform":
				a.Platform = "anthropic"
			}
			req := ticketRequest(t)
			req.Header.Set(openAICodexTurnStateHeader, "original")
			require.NoError(t, s.ApplyRequest(context.Background(), a, model, req))
			require.Equal(t, "original", req.Header.Get(openAICodexTurnStateHeader))
			require.False(t, s.Blocks(context.Background(), a, model))
		})
	}
}

func TestTicketIdentityChangesCannotBorrowCachedState(t *testing.T) {
	for _, mutation := range []string{"account", "token", "workspace", "configuration"} {
		t.Run(mutation, func(t *testing.T) {
			s, a, cfg, _ := gatewayTicketFixture(t)
			req := ticketRequest(t)
			switch mutation {
			case "account":
				a.ID++
				cfg.accountID = a.ID
			case "token":
				req.Header.Set("Authorization", "Bearer synthetic-b")
			case "workspace":
				a.Credentials["chatgpt_account_id"] = "workspace-b"
			case "configuration":
				cfg.Generation = "new-generation"
			}
			require.ErrorIs(t, s.ApplyRequest(context.Background(), a, "gpt-6-astra", req), ErrCodexTicketUnavailable)
			require.Empty(t, req.Header.Get(openAICodexTurnStateHeader))
		})
	}
}

func TestLateTicketResponseDoesNotRevokeReplacement(t *testing.T) {
	s, a, cfg, _ := gatewayTicketFixture(t)
	req := ticketRequest(t)
	require.NoError(t, s.ApplyRequest(context.Background(), a, "gpt-6-astra", req))
	key := codexTicketKey(cfg, a, "gpt-6-astra", "synthetic-a")
	cache, ok := s.cache.(*gatewayTicketCache)
	require.True(t, ok)
	cache.values[key] = "replacement"
	resp := &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"object":"response","status":"completed","model":"different"}`))}
	s.ObserveResponse(req, resp)
	_, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, "replacement", cache.values[key])
	require.Empty(t, cache.observations)
}
