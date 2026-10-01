package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

// TestOpsWSBrandNegotiation 验证新旧前端均可握手，JWT 永远不能成为响应协议。
func TestOpsWSBrandNegotiation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		_ = conn.Close()
	}))
	defer server.Close()
	for _, protocols := range [][]string{
		{"sub2api-admin", "jwt.fixture"},
		{"tokenrouter-admin", "jwt.fixture"},
		{"sub2api-admin", "tokenrouter-admin", "jwt.fixture"},
	} {
		dialer := websocket.Dialer{Subprotocols: protocols}
		conn, _, err := dialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
		require.NoError(t, err)
		want := "tokenrouter-admin"
		if len(protocols) == 2 && protocols[0] == "sub2api-admin" {
			want = "sub2api-admin"
		}
		require.Equal(t, want, conn.Subprotocol())
		require.NoError(t, conn.Close())
	}
}
