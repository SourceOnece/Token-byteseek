//go:build integration

package rediscache_test

import (
	"encoding/json"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/gateway/session"
	"github.com/TokenFlux/TokenRouter/internal/protocol/openai"
	"github.com/stretchr/testify/require"
)

// 原生请求回填和响应记录共用现有 Redis 键与七天有效期。
func (s *GatewayCacheSuite) TestReasoningHistoryNativeRoundTrip() {
	store, ok := s.cache.(session.ReasoningContentCache)
	require.True(s.T(), ok)
	history := (&session.ReasoningHistory{Cache: store}).Scoped("fixture-user-key-provider")
	history.FromInput(json.RawMessage(`[{"type":"reasoning","id":"ri_request","summary":[{"type":"summary_text","text":"request history"}]}]`))
	require.Equal(s.T(), "request history", history.Lookup("ri_request"))

	var output []openai.ResponsesOutput
	require.NoError(s.T(), json.Unmarshal([]byte(`[{"type":"reasoning","id":"ri_response","summary":[{"type":"summary_text","text":" response history "}]}]`), &output))
	history.FromOutput(output)
	require.Equal(s.T(), "response history", history.Lookup("ri_response"))
	for _, id := range []string{"ri_request", "ri_response"} {
		ttl, err := s.RDB.TTL(s.Ctx, "reasoning_content:"+session.ScopedReasoningKey("fixture-user-key-provider", id)).Result()
		require.NoError(s.T(), err)
		s.AssertTTLWithin(ttl, time.Second, 7*24*time.Hour)
	}
	require.Empty(s.T(), history.Lookup("missing_reasoning"))
	// 相同条目 ID 不得跨用户、Key 或提供商读取。
	require.Empty(s.T(), history.Scoped("other-user-key-provider").Lookup("ri_request"))
	require.Empty(s.T(), (&session.ReasoningHistory{Cache: store}).Lookup("ri_request"))
}
