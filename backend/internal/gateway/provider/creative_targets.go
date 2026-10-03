// 创作目标在单次任务尝试内绑定凭据、出站参数及供应商执行能力。
package provider

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/TokenFlux/TokenRouter/internal/egress"
	"github.com/TokenFlux/TokenRouter/internal/infra/httpclient"
	"github.com/TokenFlux/TokenRouter/internal/infra/httpclient/tlsfingerprint"

	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"

	"github.com/TokenFlux/TokenRouter/internal/creative"
	"github.com/TokenFlux/TokenRouter/internal/upstream"

	creativeprovider "github.com/TokenFlux/TokenRouter/internal/creative/provider"
	gemininative "github.com/TokenFlux/TokenRouter/internal/upstream/gemini"
	geminicli "github.com/TokenFlux/TokenRouter/internal/upstream/gemini/codeassist"
	"github.com/TokenFlux/TokenRouter/internal/upstream/grok"
	"github.com/TokenFlux/TokenRouter/internal/upstream/openai"
	"github.com/TokenFlux/TokenRouter/internal/upstream/vertex"
)

// CreativeTargets 固定任务所需的技术依赖，每次选号后只创建该提供商的受控目标。
type CreativeTargets struct {
	Requests interface {
		ImagesURL(*ExecutionProvider, string) (string, error)
		ValidateBaseURL(string) (string, error)
		TLSProfile(*ExecutionProvider, ...egress.TLSFingerprintRouterMatchResult) *tlsfingerprint.Profile
		MatchTLSInput(func() string, *ExecutionProvider) egress.TLSFingerprintRouterMatchResult
	}
	Providers    ExecutionProviderReader
	ClientPolicy *provideradapter.OpenAIProbePolicy
	Credentials  *providercore.OpenAIExecutionCredentials
	Identity     *ExecutionAgentIdentity
	Transport    httpclient.UpstreamTransport
	Routes       GrokRoutes
	GeminiTokens *providercore.GeminiTokenSource
	Enter        func() (func(), error)
}

// ForProvider 不读取凭据或启动请求，各端口继续在实际执行时求值。
func (gateway *CreativeTargets) ForProvider(provider *ExecutionProvider) *creativeprovider.Target {
	target := &creativeprovider.Target{}
	if gateway == nil || provider == nil {
		return target
	}
	oauth := provider.View().IsOpenAIOAuthLike()
	var routerMatch []egress.TLSFingerprintRouterMatchResult
	var oauthMatch egress.TLSFingerprintRouterMatchResult
	if oauth && gateway.Requests != nil {
		oauthMatch = gateway.Requests.MatchTLSInput(openai.CodexCanonicalUserAgent, provider)
		routerMatch = append(routerMatch, oauthMatch)
	}
	token := func(ctx context.Context) (string, error) {
		value, _, err := gateway.Credentials.Resolve(ctx, ExecutionRecord(provider))
		return value, err
	}
	// 创作台必须取得图片字节，URL 下载不受普通 API 的可选回填开关控制。
	imageDownload := openai.ImageBackfillOptions{
		ValidateURL: func(raw string) (string, error) {
			return gateway.Requests.ValidateBaseURL(raw)
		},
		Do: func(req *http.Request) (*http.Response, error) {
			return gateway.Transport.Do(req, creativeTargetProxyURL(provider), provider.Record.ID, provider.Record.Concurrency)
		},
	}
	target.OpenAI = &creativeprovider.OpenAIOptions{
		OAuth: oauth,
		BuildOAuth: func(ctx context.Context, run creative.CreativeRun, body []byte, token, targetURL string) (*http.Request, error) {
			return gateway.creativeOAuthRequest(ctx, provider, run, body, token, targetURL, oauthMatch)
		},
		FetchImage: imageDownload.FetchBase64,
		Token:      token,
		URL: func(endpoint string) (string, error) {
			targetURL, err := gateway.Requests.ImagesURL(provider, endpoint)
			if err != nil {
				return "", creative.CreativeNonRetryableError("creative openai base url invalid: %s", err.Error())
			}
			return targetURL, nil
		},
		Prepare: func(req *http.Request) *http.Request {
			return req.WithContext(upstream.WithHTTPUpstreamProfile(req.Context(), upstream.HTTPUpstreamProfileOpenAI))
		},
		AuthHeaders: func(ctx context.Context, token string) (http.Header, error) {
			return gateway.Identity.Headers(ctx, provider, token)
		},
		ApplyHeaders: BindExecutionHeaders(provider),
		Do: func(req *http.Request) (*http.Response, error) {
			return gateway.Transport.DoWithTLS(req, creativeTargetProxyURL(provider), provider.Record.ID, provider.Record.Concurrency, gateway.Requests.TLSProfile(provider, routerMatch...))
		},
	}
	target.Grok = &creativeprovider.GrokOptions{
		OAuth: provider.View().IsGrokOAuth(),
		Token: token,
		URL: func(endpoint grok.GrokMediaEndpoint) (string, error) {
			return gateway.Routes.Media(provider, endpoint, "")
		},
		Prepare: func(req *http.Request) *http.Request {
			return req.WithContext(upstream.WithHTTPUpstreamProfile(req.Context(), upstream.HTTPUpstreamProfileGrok))
		},
		ApplyHeaders: BindExecutionHeaders(provider),
		Do: func(req *http.Request) (*http.Response, error) {
			return gateway.Transport.Do(req, creativeTargetProxyURL(provider), provider.Record.ID, provider.Record.Concurrency)
		},
	}
	target.Gemini = func(model string) gemininative.ImageOptions {
		options := gemininative.ImageOptions{
			Mode:                  gemininative.CredentialMode(provider.Record.Type),
			Model:                 model,
			ProjectID:             provider.View().GetCredential("project_id"),
			APIKey:                func() string { return provider.View().GetCredential("api_key") },
			BaseURL:               func() string { return provider.View().GetGeminiBaseURL(geminicli.AIStudioBaseURL) },
			ValidateURL:           gateway.Requests.ValidateBaseURL,
			ValidateGeminiBaseURL: gateway.validateGeminiBaseURL,
			VertexURL: func() (string, error) {
				return vertex.BuildVertexGeminiURL(ExecutionProtocolRecord(provider).VertexProjectID(vertex.ServiceAccountProjectID), ExecutionProtocolRecord(provider).VertexLocation(model), model, "generateContent", false)
			},
			ApplyHeaders: BindExecutionHeaders(provider),
			Do: func(req *http.Request) (*http.Response, error) {
				return gateway.Transport.Do(req, creativeTargetProxyURL(provider), provider.Record.ID, provider.Record.Concurrency)
			},
			HTTPError:    func(status int, message string) error { return creative.CreativeHTTPStatusError(status, message) },
			Invalid:      func(format string, args ...any) error { return creative.CreativeNonRetryableError(format, args...) },
			ErrorMessage: upstream.ExtractErrorMessage,
			Enter:        gateway.Enter,
		}
		if gateway.GeminiTokens != nil {
			options.Token = func(ctx context.Context) (string, error) {
				return ExecutionToken(ctx, gateway.GeminiTokens, provider)
			}
		}
		return options
	}
	return target
}

// validateGeminiBaseURL 校验显式目标，错误不会改投到默认地址。
func (s *CreativeTargets) validateGeminiBaseURL(raw string) (string, error) {
	validated, err := s.Requests.ValidateBaseURL(raw)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(validated) == "" {
		return "", errors.New("gemini base url is empty")
	}
	return validated, nil
}

func creativeTargetProxyURL(value *ExecutionProvider) string {
	if value == nil || value.Record.ProxyID == nil || value.Record.Proxy == nil {
		return ""
	}
	return value.Record.Proxy.URL()
}
