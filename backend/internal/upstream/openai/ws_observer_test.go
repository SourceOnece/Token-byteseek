package openai

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

type handshakeObserverProbe struct {
	state   string
	headers int
	models  []string
}

func (o *handshakeObserverProbe) ForHeaders(h http.Header) WSHandshakeObserver {
	if h.Get("x-codex-turn-state") != o.state {
		return nil
	}
	return o
}
func (o *handshakeObserverProbe) ObserveHeader(http.Header) { o.headers++ }
func (o *handshakeObserverProbe) ObserveJSON(_ []byte, _, model string) {
	o.models = append(o.models, model)
}

// 复用连接必须继续观察原握手收据，不能把下一张票认作该连接实际发出的票。
func TestWSHandshakeObserverStaysWithPhysicalConnection(t *testing.T) {
	p := newStartedWSConnPoolForTest(&WSPoolOptions{MaxConnsPerProvider: 1})
	t.Cleanup(p.Close)
	p.SetClientDialerForTest(&openAIWSFakeDialer{})
	a, b := &handshakeObserverProbe{state: "first"}, &handshakeObserverProbe{state: "second"}
	req := WSAcquireRequest{Provider: &WSPoolProvider{ID: 7, Type: "oauth"}, WSURL: "wss://example.test/responses", Observer: a, Headers: http.Header{}}
	req.Headers.Set("x-codex-turn-state", a.state)
	first, err := p.Acquire(context.Background(), req)
	require.NoError(t, err)
	firstID := first.ConnID()
	first.Release()
	req.Observer = b
	req.Headers.Set("x-codex-turn-state", b.state)
	second, err := p.Acquire(context.Background(), req)
	require.NoError(t, err)
	t.Cleanup(second.Release)
	require.Equal(t, firstID, second.ConnID())
	second.ObserveTicket([]byte(`{"type":"response.completed"}`), "gpt-6-astra")
	require.Equal(t, 1, a.headers)
	require.Equal(t, []string{"gpt-6-astra"}, a.models)
	require.Zero(t, b.headers)
	require.Empty(t, b.models)
}

func TestWSHandshakeObserverRejectsOverwrittenState(t *testing.T) {
	p := newStartedWSConnPoolForTest(&WSPoolOptions{})
	t.Cleanup(p.Close)
	p.SetClientDialerForTest(&openAIWSFakeDialer{})
	o := &handshakeObserverProbe{state: "first"}
	req := WSAcquireRequest{Provider: &WSPoolProvider{ID: 7, Type: "oauth"}, WSURL: "wss://example.test/responses", Observer: o, Headers: http.Header{}}
	req.Headers.Set("x-codex-turn-state", o.state)
	req.HeadersFactory = func(_ context.Context, h http.Header) (http.Header, error) {
		h.Set("x-codex-turn-state", "continuation")
		return h, nil
	}
	lease, err := p.Acquire(context.Background(), req)
	require.NoError(t, err)
	t.Cleanup(lease.Release)
	lease.ObserveTicket([]byte(`{"type":"response.completed"}`), "gpt-6-astra")
	require.Zero(t, o.headers)
	require.Empty(t, o.models)
}
