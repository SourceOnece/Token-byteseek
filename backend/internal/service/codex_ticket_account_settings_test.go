//go:build unit

package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/stretchr/testify/require"
)

// 使用独立账号仓储，不把管理员设置写进账号凭据或调度状态。
func TestCodexTicketAccountOverrides(t *testing.T) {
	s, cache, _ := newTicketTestService()
	enableTicketTest(t, s)
	a, b := ticketAccount(), ticketAccount()
	b.ID = 2
	b.Credentials = map[string]any{"access_token": "second"}
	s.gateway = &OpenAIGatewayService{accountRepo: &ticketAccountStub{accounts: []Account{a, b}}}
	ctx := context.Background()
	root := s.config.Load()
	beforeA := codexTicketKey(root, &a, "gpt-6-astra", "fake-token")
	beforeB := codexTicketKey(root, &b, "gpt-6-astra", "second")
	seedTicket(t, s, &b, "gpt-6-astra", "second")
	off := "off"
	proxy := "socks5h://user-{sid}:synthetic-password@account-proxy.invalid:1080"
	result, err := s.UpdateAccountSettings(ctx, CodexTicketAccountsUpdate{AccountIDs: []int64{1}, Patch: CodexTicketAccountPatch{Mode: &off, HarvestProxyURL: &proxy}})
	require.NoError(t, err)
	require.False(t, result[0].EffectiveEnabled)
	require.True(t, result[0].ProxyConfigured)
	raw, _ := json.Marshal(result)
	require.NotContains(t, string(raw), "synthetic-password")
	require.NotContains(t, string(raw), "cipher")
	require.Equal(t, root.Generation, s.config.Load().Generation, "改一号不使全池票据换代")
	require.Equal(t, beforeB, codexTicketKey(ticketConfigForAccount(s.config.Load(), 2), &b, "gpt-6-astra", "second"))
	require.NotEqual(t, beforeA, codexTicketKey(ticketConfigForAccount(s.config.Load(), 1), &a, "gpt-6-astra", "fake-token"))
	gets := cache.gets
	require.False(t, s.Blocks(ctx, &a, "gpt-6-astra"))
	require.Equal(t, gets, cache.gets)
	require.False(t, s.Blocks(ctx, &b, "gpt-6-astra"))
	on := "on"
	result, err = s.UpdateAccountSettings(ctx, CodexTicketAccountsUpdate{AccountIDs: []int64{1}, Patch: CodexTicketAccountPatch{Mode: &on}})
	require.NoError(t, err)
	require.True(t, result[0].ProxyConfigured, "没勾代理不删除")
	cfg := s.enabledAccountConfig(1)
	require.Equal(t, "account", cfg.proxies()[0].ID)
	clear := ""
	_, err = s.UpdateAccountSettings(ctx, CodexTicketAccountsUpdate{AccountIDs: []int64{1}, Patch: CodexTicketAccountPatch{HarvestProxyURL: &clear}})
	require.NoError(t, err)
	require.Equal(t, "legacy", s.enabledAccountConfig(1).proxies()[0].ID)
	no := false
	_, err = s.Update(ctx, CodexTicketSettingsUpdate{Enabled: &no})
	require.NoError(t, err)
	require.Nil(t, s.enabledAccountConfig(1))
	require.True(t, a.Schedulable)
	require.True(t, b.Schedulable)
}

func TestCodexTicketAccountBatchPartialPatchAndValidation(t *testing.T) {
	s, _, _ := newTicketTestService()
	enableTicketTest(t, s)
	a, b := ticketAccount(), ticketAccount()
	b.ID = 2
	s.gateway = &OpenAIGatewayService{accountRepo: &ticketAccountStub{accounts: []Account{a, b}}}
	ctx := context.Background()
	proxy := "http://u:synthetic@p.invalid:8080"
	mode := "recover_length"
	_, err := s.UpdateAccountSettings(ctx, CodexTicketAccountsUpdate{AccountIDs: []int64{1, 2, 2}, Patch: CodexTicketAccountPatch{HarvestProxyURL: &proxy}})
	require.NoError(t, err)
	before := s.config.Load()
	r, err := s.UpdateAccountSettings(ctx, CodexTicketAccountsUpdate{AccountIDs: []int64{1, 2}, Patch: CodexTicketAccountPatch{WatchdogMode: &mode}})
	require.NoError(t, err)
	require.Len(t, r, 2)
	require.True(t, r[0].ProxyConfigured)
	require.Equal(t, "inherit", r[0].Mode)
	require.NotEqual(t, before.Accounts["1"].Revision, s.config.Load().Accounts["1"].Revision)
	stale := "stale"
	off := "off"
	for _, input := range []CodexTicketAccountsUpdate{
		{AccountIDs: []int64{1, 3}, Patch: CodexTicketAccountPatch{Mode: &off}},
		{AccountIDs: []int64{1}, Patch: CodexTicketAccountPatch{}},
		{AccountIDs: []int64{1}, Patch: CodexTicketAccountPatch{Mode: &off}, Revision: &stale},
	} {
		previous := s.config.Load()
		_, err = s.UpdateAccountSettings(ctx, input)
		require.Error(t, err)
		require.Same(t, previous, s.config.Load())
	}
}

func TestCodexTicketProxySessionTemplate(t *testing.T) {
	raw := "socks5h://name-%7Bsid%7D:pw%40%7Brandom%7D@proxy.invalid:1080"
	a, b := expandTicketProxySession(raw), expandTicketProxySession(raw)
	require.NotEqual(t, a, b)
	u, err := url.Parse(a)
	require.NoError(t, err)
	password, _ := u.User.Password()
	require.Equal(t, strings.TrimPrefix(u.User.Username(), "name-"), strings.TrimPrefix(password, "pw@"))
	require.Equal(t, "proxy.invalid:1080", u.Host)
	require.Equal(t, "http://user:pass@proxy.invalid:8080", expandTicketProxySession("http://user:pass@proxy.invalid:8080"))
}

// Redis替身同样按密文比较，保证模型观察不能废弃刚更新的另一张票。
type ticketWatchdogCacheStub struct{ *ticketCacheStub }

func (c *ticketWatchdogCacheStub) ObserveTicket(_ context.Context, key, expected, reason string, revoke bool, at time.Time) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.values[key] != expected {
		return false, nil
	}
	action := "observed"
	if revoke {
		delete(c.values, key)
		action = "revoked"
	}
	raw, _ := json.Marshal(CodexTicketWatchdogStatus{Count: 1, Reason: reason, Action: action, CheckedAt: at})
	c.values["watchdog:"+key] = string(raw)
	return true, nil
}

func TestCodexTicketWatchdogModes(t *testing.T) {
	for _, mode := range []string{"observe", "off", "recover_length", "recover_model", "recover"} {
		t.Run(mode, func(t *testing.T) {
			s, c, _ := newTicketTestService()
			enableTicketTest(t, s)
			s.cache = &ticketWatchdogCacheStub{c}
			length := 312
			_, err := s.Update(context.Background(), CodexTicketSettingsUpdate{WatchdogMode: &mode, DegradedSignalLength: &length})
			require.NoError(t, err)
			a := ticketAccount()
			seedTicket(t, s, &a, "gpt-6-astra", "fake-token")
			h := http.Header{"Authorization": {"Bearer fake-token"}}
			receipt, err := s.applyWithReceipt(context.Background(), &a, "gpt-6-astra", h)
			require.NoError(t, err)
			if mode == "off" {
				require.Nil(t, receipt)
				return
			}
			receipt.observeHeader(http.Header{http.CanonicalHeaderKey(openAICodexTurnStateHeader): {"gAAAAA" + strings.Repeat("x", 306)}})
			_, exists := c.values[receipt.key]
			require.Equal(t, mode != "recover_length" && mode != "recover", exists)
			status := safeTicketWatchdog(c.values["watchdog:"+receipt.key], mode)
			require.Equal(t, "length_signal", status.Reason)
			if !exists {
				return
			}
			receipt.observeJSON([]byte(`{"type":"response.completed","response":{"model":"gpt-5.6-luna","status":"completed"}}`), "", "gpt-6-astra")
			_, exists = c.values[receipt.key]
			require.Equal(t, mode != "recover_model", exists)
		})
	}
}

func TestCodexTicketWatchdogTransparentBodiesAndStaleReceipt(t *testing.T) {
	for _, name := range []string{"sse", "json", "wrong_model_context", "incomplete", "oversized", "stale", "disabled", "client_header"} {
		t.Run(name, func(t *testing.T) {
			s, c, _ := newTicketTestService()
			enableTicketTest(t, s)
			s.cache = &ticketWatchdogCacheStub{c}
			mode := "recover_model"
			_, err := s.Update(context.Background(), CodexTicketSettingsUpdate{WatchdogMode: &mode})
			require.NoError(t, err)
			a := ticketAccount()
			seedTicket(t, s, &a, "gpt-6-astra", "fake-token")
			req := httptest.NewRequest(http.MethodPost, "/responses", nil)
			req.Header.Set("Authorization", "Bearer fake-token")
			if name != "client_header" {
				require.NoError(t, s.ApplyRequest(context.Background(), &a, "gpt-6-astra", req))
			}
			key := codexTicketKey(s.enabledAccountConfig(1), &a, "gpt-6-astra", "fake-token")
			payload := `{"type":"response.completed","response":{"status":"completed","model":"gpt-5.6-luna"}}`
			body := "event: response.completed\ndata: " + payload + "\n\n"
			switch name {
			case "json":
				body = `{"object":"response","status":"completed","model":"gpt-5.6-luna"}`
			case "incomplete":
				body = `data: {"type":"response.failed","response":{"model":"gpt-5.6-luna"}}` + "\n\n"
			case "oversized":
				body = "data: " + strings.Repeat("x", 1024*1024+1) + "\n\n"
			case "stale":
				c.values[key] = "newer-encrypted-ticket"
			case "disabled":
				no := false
				_, err = s.Update(context.Background(), CodexTicketSettingsUpdate{Enabled: &no})
				require.NoError(t, err)
			case "wrong_model_context":
				r := req.Context().Value(ticketReceiptContextKey{}).(*codexTicketReceipt)
				r.observeJSON([]byte(payload), "", "gpt-5.6-sol")
				require.Contains(t, c.values, key)
				return
			}
			resp := &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}
			s.ObserveResponse(req, resp)
			// 分割成小块读取，原始内容和EOF必须与未包裹的响应一致。
			var got strings.Builder
			buffer := make([]byte, 7)
			for {
				n, e := resp.Body.Read(buffer)
				got.Write(buffer[:n])
				if e != nil {
					require.Equal(t, io.EOF, e)
					break
				}
			}
			require.NoError(t, resp.Body.Close())
			require.Equal(t, body, got.String())
			_, exists := c.values[key]
			require.Equal(t, name != "sse" && name != "json", exists)
		})
	}
}

func TestCodexTicketWSReceiptBelongsToPhysicalHandshake(t *testing.T) {
	s, cache, _ := newTicketTestService()
	enableTicketTest(t, s)
	s.cache = &ticketWatchdogCacheStub{cache}
	a := ticketAccount()
	seedTicket(t, s, &a, "gpt-6-astra", "fake-token")
	headers := http.Header{"Authorization": {"Bearer fake-token"}}
	r, err := s.applyWithReceipt(context.Background(), &a, "gpt-6-astra", headers)
	require.NoError(t, err)
	require.NotNil(t, r)
	pool := newOpenAIWSConnPool(&config.Config{})
	pool.setClientDialerForTest(&openAIWSFakeDialer{})
	conn, err := pool.dialConn(context.Background(), openAIWSAcquireRequest{Account: &a, WSURL: "wss://example.invalid", Headers: headers, ticketReceipt: r})
	require.NoError(t, err)
	defer conn.close()
	require.Same(t, r, conn.ticketReceipt)
	// 回合状态若被上游更新，不得把后来重新握手的连接仍算到旧票上。
	headers.Set(openAICodexTurnStateHeader, "upstream-next-turn")
	next, err := pool.dialConn(context.Background(), openAIWSAcquireRequest{Account: &a, WSURL: "wss://example.invalid", Headers: headers, ticketReceipt: r})
	require.NoError(t, err)
	defer next.close()
	require.Nil(t, next.ticketReceipt)
	require.Same(t, r, conn.ticketReceipt, "新握手/新票不能覆盖既有连接的收据")
}

func TestCodexTicketAccountProxyUsedByManualAndAutomatic(t *testing.T) {
	s, _, u := setupTicketManualTest(t, 292)
	proxyCalls := make(chan string, 16)
	s.gateway.httpUpstream = &ticketAccountProxyRecorder{HTTPUpstream: u, base: u, calls: proxyCalls}
	proxy := "http://user-{sid}:synthetic@own-proxy.invalid:8080"
	_, err := s.UpdateAccountSettings(context.Background(), CodexTicketAccountsUpdate{AccountIDs: []int64{1}, Patch: CodexTicketAccountPatch{HarvestProxyURL: &proxy}})
	require.NoError(t, err)
	u.before = func(req *http.Request) { require.NotEmpty(t, req.Header.Get("Authorization")) }
	ctx := context.Background()
	batch, err := s.PrepareManualCollection(ctx, manualRequest(s))
	require.NoError(t, err)
	batch.Execute(func(string, any) bool { return true })
	status, err := s.Status(ctx, []int64{1})
	require.NoError(t, err)
	require.Equal(t, "account", status.Items[0].Settings.ProxySource)
	for _, m := range status.Items[0].Models {
		require.Equal(t, "ready", m.State)
		require.Equal(t, "account", m.Diagnostic.ProxyID)
	}
	// 清除测试票据使自动轮次也实际采集，不仅验证状态接口返回的标识。
	cache := s.cache.(*ticketCacheStub)
	cache.mu.Lock()
	cache.values = map[string]string{}
	cache.mu.Unlock()
	s.harvest(ctx)
	require.Len(t, proxyCalls, 4)
	for len(proxyCalls) > 0 {
		proxy := <-proxyCalls
		require.Contains(t, proxy, "own-proxy.invalid:8080")
		require.NotContains(t, proxy, "{sid}")
		require.NotContains(t, proxy, "%7Bsid")
	}
}

type ticketAccountProxyRecorder struct {
	HTTPUpstream
	base  *ticketUpstreamStub
	calls chan string
}

func (r *ticketAccountProxyRecorder) Do(req *http.Request, proxy string, id int64, concurrency int) (*http.Response, error) {
	r.calls <- proxy
	return r.base.Do(req, proxy, id, concurrency)
}
