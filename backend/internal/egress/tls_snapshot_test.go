package egress

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// 缓存写入和返回都必须隔离，不因多个请求共享来源而串改策略。
func TestTLSProfileSnapshotIsolation(t *testing.T) {
	source := &TLSFingerprintProfile{ID: 1, Name: "profile", CipherSuites: []uint16{4865}, ALPNProtocols: []string{"h2"}}
	svc := NewTLSFingerprintProfileService(nil, nil)
	svc.setLocalCache([]*TLSFingerprintProfile{source})
	first := svc.GetProfileByID(1)
	first.CipherSuites[0] = 4866
	first.ALPNProtocols[0] = "http/1.1"
	again := svc.GetProfileByID(1)
	require.Equal(t, uint16(4865), again.CipherSuites[0])
	require.Equal(t, "h2", again.ALPNProtocols[0])
	source.CipherSuites[0] = 4867
	require.Equal(t, uint16(4865), svc.GetProfileByID(1).CipherSuites[0])
}

func TestTLSRouterSnapshotIsolation(t *testing.T) {
	id := int64(1)
	source := &TLSFingerprintRouter{ID: 1, Name: "router", Enabled: true, ChatGPTOAuthTokenTLSFingerprintProfileID: &id}
	svc := NewTLSFingerprintRouterService(nil, nil)
	svc.setLocalCache([]*TLSFingerprintRouter{source})
	first := svc.GetRuntimeRouter(1)
	*first.ChatGPTOAuthTokenTLSFingerprintProfileID = 2
	require.Equal(t, int64(1), *svc.GetRuntimeRouter(1).ChatGPTOAuthTokenTLSFingerprintProfileID)
	id = 3
	require.Equal(t, int64(1), *svc.GetRuntimeRouter(1).ChatGPTOAuthTokenTLSFingerprintProfileID)
}
