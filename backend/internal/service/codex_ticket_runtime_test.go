//go:build unit

package service

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// 此运行替身不嵌入旧网关，用实际采集/复验/存票/调度流程证明接口可替换。
type independentTicketRuntime struct {
	store                      CodexTicketAccountStore
	upstream                   *ticketVerifiedUpstream
	slots, released, snapshots int
	denySlot                   bool
	credentials                []int64
	headers                    []int64
}

func (r *independentTicketRuntime) TicketAccountStore() CodexTicketAccountStore { return r.store }
func (r *independentTicketRuntime) TicketSnapshotAccount(ctx context.Context, id int64) (*Account, error) {
	r.snapshots++
	return r.store.GetByID(ctx, id)
}
func (r *independentTicketRuntime) GetAccessToken(_ context.Context, a *Account) (string, string, error) {
	r.credentials = append(r.credentials, a.ID)
	return a.GetOpenAIAccessToken(), "oauth", nil
}
func (r *independentTicketRuntime) TicketAccountHeaders(_ context.Context, headers http.Header, a *Account) error {
	r.headers = append(r.headers, a.ID)
	setOpenAIChatGPTAccountHeaders(headers, a)
	return nil
}
func (r *independentTicketRuntime) TicketSlotTTL() time.Duration { return time.Minute }
func (r *independentTicketRuntime) AcquireTicketSlot(context.Context, *Account) (*AcquireResult, error) {
	r.slots++
	return &AcquireResult{Acquired: !r.denySlot, ReleaseFunc: func() { r.released++ }}, nil
}
func (r *independentTicketRuntime) TicketTransportAvailable() bool { return r.upstream != nil }
func (r *independentTicketRuntime) SendTicketHarvest(req *http.Request, proxy string, _ *Account) (*http.Response, error) {
	return r.upstream.call(req, proxy, false)
}
func (r *independentTicketRuntime) SendTicketVerification(req *http.Request, proxy string, _ *Account) (*http.Response, error) {
	return r.upstream.call(req, proxy, true)
}

func TestCodexTicketIndependentRuntimeCollection(t *testing.T) {
	for _, manual := range []bool{false, true} {
		for _, tc := range []struct {
			name    string
			initial bool
			length  int
			want    bool
			calls   int
		}{
			{"qualified", false, 292, true, 2},
			{"signal", true, 312, false, 1},
			{"other", true, 356, true, 1},
		} {
			t.Run(tc.name+map[bool]string{false: "/auto", true: "/manual"}[manual], func(t *testing.T) {
				source, repo, up := setupVerifiedTicketTest(t, true)
				repo.accounts[0].Schedulable = tc.initial
				up.responses = []*http.Response{verifiedResponse(200, tc.length, "gpt-6-astra"), verifiedResponse(200, 0, "gpt-6-astra")}
				runtime := &independentTicketRuntime{store: repo, upstream: up}
				s := NewCodexTicketService(runtime, source.settings, source.cache, source.cipher, source.ipProber, source.proxyRepo)
				s.config.Store(source.config.Load())
				require.Nil(t, s.done, "构造阶段不能启动后台采集")
				a, err := repo.GetByID(context.Background(), 1)
				require.NoError(t, err)
				if manual {
					batch, err := s.PrepareManualCollection(context.Background(), manualRequest(s))
					require.NoError(t, err)
					batch.Execute(func(string, any) bool { return true })
				} else {
					s.probe(context.Background(), s.config.Load(), a, "gpt-6-astra")
				}
				updated, err := repo.GetByID(context.Background(), 1)
				require.NoError(t, err)
				require.Equal(t, tc.want, updated.Schedulable)
				require.Len(t, up.calls, tc.calls)
				require.Equal(t, 1, runtime.slots)
				require.Equal(t, 1, runtime.released)
				require.Equal(t, []int64{1}, runtime.credentials)
				require.Equal(t, []int64{1}, runtime.headers)
				if tc.name == "qualified" {
					require.False(t, up.calls[0].verify)
					require.Empty(t, up.calls[0].state)
					require.True(t, up.calls[1].verify)
					require.Len(t, up.calls[1].state, 292)
					require.Equal(t, a.Proxy.URL(), up.calls[1].proxy)
					// 摘要缺令牌时依旧经运行接口补齐，不把已有票误判成缺票。
					slim := *updated
					slim.Credentials = map[string]any{}
					require.False(t, s.Blocks(context.Background(), &slim, "gpt-6-astra"))
					require.Equal(t, 1, runtime.snapshots)
				}
			})
		}
	}
}

func TestCodexTicketIndependentRuntimeRejectsBusySlot(t *testing.T) {
	source, repo, up := setupVerifiedTicketTest(t, true)
	runtime := &independentTicketRuntime{store: repo, upstream: up, denySlot: true}
	s := NewCodexTicketService(runtime, source.settings, source.cache, source.cipher, nil, nil)
	s.config.Store(source.config.Load())
	a, err := repo.GetByID(context.Background(), 1)
	require.NoError(t, err)
	s.probe(context.Background(), s.config.Load(), a, "gpt-6-astra")
	require.Empty(t, up.calls)
	require.Zero(t, repo.writes)
	require.Zero(t, runtime.released)
}
