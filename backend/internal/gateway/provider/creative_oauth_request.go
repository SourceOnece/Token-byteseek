package provider

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/TokenFlux/TokenRouter/internal/creative"
	"github.com/TokenFlux/TokenRouter/internal/egress"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
	"github.com/TokenFlux/TokenRouter/internal/upstream"
	"github.com/TokenFlux/TokenRouter/internal/upstream/openai"
)

// creativeOAuthRequest 以当前任务和凭据建立 Codex 图片请求。
// 每个任务的会话同时绑定用户、托管 Key 和提供商，换号后重新计算。
func (g *CreativeTargets) creativeOAuthRequest(ctx context.Context, selected *ExecutionProvider, run creative.CreativeRun, body []byte, token, targetURL string, match egress.TLSFingerprintRouterMatchResult) (*http.Request, error) {
	if selected == nil || !selected.View().IsOpenAIOAuthLike() || g.Identity == nil || (selected.View().IsShadow() && g.Providers == nil) {
		return nil, errors.New("creative OAuth authentication is not configured")
	}
	credential, err := CredentialProvider(ctx, g.Providers, selected)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(upstream.WithHTTPUpstreamProfile(ctx, upstream.HTTPUpstreamProfileOpenAI), http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Host = "chatgpt.com"
	headers, err := g.Identity.Headers(ctx, credential, token)
	if err != nil {
		return nil, err
	}
	req.Header = headers.Clone()
	provideradapter.SetChatGPTAccountHeaders(req.Header, credential.View())
	req.Header.Set("User-Agent", openai.CodexCanonicalUserAgent())
	req.Header.Set("originator", openai.ResolveCodexOutboundIdentity("").Originator)
	if g.ClientPolicy != nil {
		g.ClientPolicy.ApplyUserAgent(ctx, credential.View(), req, false, match)
	} else if ua := strings.TrimSpace(credential.View().GetOpenAIUserAgent()); ua != "" {
		req.Header.Set("User-Agent", ua)
	}
	namespace := provideradapter.CodexIdentityNamespace(credential.View())
	if namespace == "" {
		namespace = fmt.Sprintf("creative-provider:%d", credential.Record.ID)
	}
	session := fmt.Sprintf("creative:%d:%d:%s", run.UserID, run.APIKeyID, run.RunID)
	isolated := openai.IsolateOpenAIUpstreamSessionID(run.APIKeyID, namespace, session)
	req.Header.Set("session_id", isolated)
	req.Header.Set("conversation_id", isolated)
	openai.ApplyCodexProviderIdentityHeaders(req.Header, namespace, run.APIKeyID)
	ids := provideradapter.CodexFingerprintIDs(credential.View(), isolated, credential.View().GetCodexFingerprintMode())
	openai.ApplyCodexFingerprintHeaders(req.Header, ids)
	openai.EnforceCodexIdentityHeaders(req.Header)
	BindExecutionHeaders(selected)(req.Header)
	return req, nil
}
