package openaiforward

import (
	"context"
	"errors"
	"testing"

	"github.com/tidwall/sjson"
)

// 到凭据阶段即停止，避免任何真实上游请求；验证拒绝是否在正确的模型映射时点执行。
type gpt61RawGuardPorts struct {
	RawChatPorts
	credentialReached bool
	rejection         int
}

func (p *gpt61RawGuardPorts) Profile() MessagesProfile         { return MessagesProfile{OpenAI: true} }
func (*gpt61RawGuardPorts) BillingModel(string, string) string { return "gpt-6.1-sol" }
func (*gpt61RawGuardPorts) UpstreamModel(s string) string      { return s }
func (*gpt61RawGuardPorts) ObserveModel(string)                {}
func (*gpt61RawGuardPorts) ReplaceModel(b []byte, m string) []byte {
	v, _ := sjson.SetBytes(b, "model", m)
	return v
}
func (*gpt61RawGuardPorts) NormalizeGLM(b []byte, _ string) ([]byte, bool) { return b, false }
func (*gpt61RawGuardPorts) FastRaw(_ context.Context, _ string, b []byte) ([]byte, error) {
	return b, nil
}
func (*gpt61RawGuardPorts) ServiceTier([]byte) *string                             { return nil }
func (*gpt61RawGuardPorts) EffectiveEffort([]byte, []byte, ...string) *string      { return nil }
func (*gpt61RawGuardPorts) ThinkingFallback(v *string, _ []byte, _ string) *string { return v }
func (p *gpt61RawGuardPorts) Error(status int, _ string, _ string)                 { p.rejection = status }

func (p *gpt61RawGuardPorts) RawCredential(context.Context) (string, string, error) {
	p.credentialReached = true
	return "", "", errors.New("stop before network")
}

func TestGPT61MappedRawGuards(t *testing.T) {
	for _, body := range []string{
		`{"model":"public-alias","reasoning_effort":"none","messages":[{"role":"user","content":"test"}]}`,
		`{"model":"gpt-6.1-sol","tools":[{"type":"function","function":{"name":"test","parameters":{"type":"object"}}}],"messages":[{"role":"user","content":"test"}]}`,
	} {
		p := &gpt61RawGuardPorts{}
		_, _ = RunRawChat(context.Background(), []byte(body), "", p)
		t.Logf("rejection=%d credential_stage_reached=%v", p.rejection, p.credentialReached)
		if p.credentialReached {
			t.Error("required GPT-6.1 validation missing before credentials/network")
		}
	}
}
