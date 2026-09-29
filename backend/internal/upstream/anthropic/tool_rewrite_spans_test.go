package anthropic

import (
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"strings"
	"testing"
)

// sub2api 27a9421e6 回归：仅改工具引用，保留输入正文和重复键语义。
func TestApplyToolNameRewriteToBody_Spans(t *testing.T) {
	body := []byte(`{"tools":[{"name":"sessions_\"quoted"},{"name":"session_last","defer_loading":true,"cache_control":{"type":"ephemeral"}}],"tool_choice":{"type":"tool","name":"sessions_\"quoted"},"messages":[{"content":[{"type":"tool_use","name":"sessions_\"quoted","input":{"name":"sessions_\"quoted"}},{"type":"tool_use"}]}],"metadata":{"name":"sessions_\"quoted"}}`)
	out := ApplyToolNameRewriteToBody(body, BuildToolNameRewriteFromBody(body))
	require.True(t, gjson.ValidBytes(out))
	for _, path := range []string{"tools.0.name", "tool_choice.name", "messages.0.content.0.name"} {
		require.Equal(t, `cc_sess_"quoted`, gjson.GetBytes(out, path).String())
	}
	require.Equal(t, `sessions_"quoted`, gjson.GetBytes(out, "messages.0.content.0.input.name").String())
	require.Equal(t, `sessions_"quoted`, gjson.GetBytes(out, "metadata.name").String())
	require.False(t, gjson.GetBytes(out, "tools.1.cache_control").Exists())
	require.Equal(t, DefaultCacheControlTTL, gjson.GetBytes(out, "tools.0.cache_control.ttl").String())
	require.False(t, gjson.GetBytes(out, "messages.0.content.1.name").Exists())
}

func TestApplyToolNameRewriteToBody_DuplicateNames(t *testing.T) {
	body := []byte(`{"tools":[{"name":"sessions_first","name":"sessions_second"}],"messages":[{"content":[{"type":"tool_use","name":"sessions_first","name":"sessions_second"}]}]}`)
	out := ApplyToolNameRewriteToBody(body, BuildToolNameRewriteFromBody(body))
	require.True(t, gjson.ValidBytes(out))
	require.Equal(t, 2, strings.Count(string(out), `"name":"cc_sess_first","name":"sessions_second"`))
}

func TestApplyToolNameRewriteToBody_NonStringReferences(t *testing.T) {
	body := []byte(`{"tools":[{"name":"123"}],"tool_choice":{"type":"tool","name":123},"messages":[{"content":[{"type":"tool_use","name":123},{"type":"tool_use","name":true}]}]}`)
	rw := &ToolNameRewrite{Forward: map[string]string{"123": "replacement", "true": "also_replaced"}}
	out := ApplyToolNameRewriteToBody(body, rw)
	require.True(t, gjson.ValidBytes(out))
	require.Equal(t, "replacement", gjson.GetBytes(out, "tools.0.name").String())
	require.Equal(t, `"replacement"`, gjson.GetBytes(out, "tool_choice.name").Raw)
	require.Equal(t, `"replacement"`, gjson.GetBytes(out, "messages.0.content.0.name").Raw)
	require.Equal(t, `"also_replaced"`, gjson.GetBytes(out, "messages.0.content.1.name").Raw)
}

func makeLongToolHistory() []byte {
	var b strings.Builder
	_, _ = b.WriteString(`{"tools":[{"name":"sessions_list"}],"tool_choice":{"type":"tool","name":"sessions_list"},"messages":[`)
	for i := 0; i < 1000; i++ {
		if i > 0 {
			_ = b.WriteByte(',')
		}
		_, _ = b.WriteString(`{"content":[{"type":"tool_use","name":"sessions_list","input":{"name":"sessions_list"}}]}`)
	}
	_, _ = b.WriteString(`]}`)
	return []byte(b.String())
}

func TestApplyToolNameRewriteToBody_LongHistory(t *testing.T) {
	body := makeLongToolHistory()
	out := ApplyToolNameRewriteToBody(body, BuildToolNameRewriteFromBody(body))
	require.True(t, gjson.ValidBytes(out))
	require.Equal(t, "cc_sess_list", gjson.GetBytes(out, "messages.999.content.0.name").String())
	require.Equal(t, "sessions_list", gjson.GetBytes(out, "messages.999.content.0.input.name").String())
}

func BenchmarkApplyToolNameRewriteToBody_LongHistory(b *testing.B) {
	body := makeLongToolHistory()
	rw := BuildToolNameRewriteFromBody(body)
	b.ReportAllocs()
	b.SetBytes(int64(len(body)))
	for i := 0; i < b.N; i++ {
		_ = ApplyToolNameRewriteToBody(body, rw)
	}
}
