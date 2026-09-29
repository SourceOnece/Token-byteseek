package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/apikey"
	"github.com/TokenFlux/TokenRouter/internal/identity"
	"github.com/TokenFlux/TokenRouter/internal/protocol"
	"github.com/TokenFlux/TokenRouter/internal/routing"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 未消费请求不应读取正文，也不拥有取得槽位或资金检查的端口。
type modelForbiddenBody struct{}

func (modelForbiddenBody) Read([]byte) (int, error) { panic("models endpoint read request body") }
func (modelForbiddenBody) Close() error             { return nil }

type modelsBackendStub struct {
	ModelsBackend
	key               *apikey.APIKey
	result            routing.RequestableModelsResult
	byGroup           map[int64]routing.RequestableModelsResult
	forced            string
	resolvedPlatforms []string
	response          *ModelHTTPResponse
	selectErr         error
	antigravity       bool
	paths             []string
	observations      int
}

func (p *modelsBackendStub) Access(*gin.Context) (*apikey.APIKey, bool) { return p.key, p.key != nil }

func (p *modelsBackendStub) ForcedPlatform(*gin.Context) (string, bool) {
	return p.forced, p.forced != ""
}
func (p *modelsBackendStub) Available() bool { return true }
func (p *modelsBackendStub) Resolve(_ context.Context, id *int64, platform string) routing.RequestableModelsResult {
	p.resolvedPlatforms = append(p.resolvedPlatforms, platform)
	if id != nil && p.byGroup != nil {
		return p.byGroup[*id]
	}
	return p.result
}

func (p *modelsBackendStub) SelectGemini(context.Context, *int64) (GeminiModelReader, error) {
	if p.selectErr != nil {
		return nil, p.selectErr
	}
	return p, nil
}

func (p *modelsBackendStub) Read(_ context.Context, path string) (*ModelHTTPResponse, error) {
	p.paths = append(p.paths, path)
	return p.response, nil
}

func (p *modelsBackendStub) HasAntigravity(context.Context, *int64) (bool, error) {
	return p.antigravity, nil
}
func (p *modelsBackendStub) CapacityLimited(*gin.Context, error) { p.observations++ }
func (p *modelsBackendStub) SafeModelSegment(m string) bool      { return m != "bad/model" }

// 未被约定回退分支调用的目录方法保持未实现，避免空成功恢复默认列表。
type modelsCatalogStub struct {
	ModelsCatalog
	fallbacks int
}

func (p *modelsCatalogStub) GeminiList(bool) GeminiModelsList {
	p.fallbacks++
	return GeminiModelsList{Models: []GeminiModel{{Name: "models/fallback"}}}
}

func (p *modelsCatalogStub) GeminiModel(name string, _ bool) GeminiModel {
	p.fallbacks++
	return GeminiModel{Name: "models/" + name}
}
func (p *modelsCatalogStub) HasGeminiFallback(name string) bool { return name == "known" }
func modelsContext() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	c.Request.Body = modelForbiddenBody{}
	return c, w
}

func TestModelsBoundEmptyDoesNotFallBackOrConsume(t *testing.T) {
	p := &modelsBackendStub{key: prefaceKey()}
	c, w := modelsContext()
	NewModelsHandler(p, &modelsCatalogStub{}).Models(c)
	require.Equal(t, 200, w.Code)
	require.JSONEq(t, `{"object":"list","data":[]}`, w.Body.String())
	require.Empty(t, p.paths)
}

// 空能力目录不能用任何上游目录或内置模型补全。
func TestGeminiModelsEmptyCapabilityDoesNotQueryUpstream(t *testing.T) {
	p := &modelsBackendStub{key: prefaceKey(), response: &ModelHTTPResponse{StatusCode: 200, Body: []byte(`{"models":[{"name":"models/phantom"}]}`)}}
	c, w := modelsContext()
	NewModelsHandler(p, &modelsCatalogStub{}).GeminiV1BetaListModels(c)
	require.Equal(t, 200, w.Code)
	require.JSONEq(t, `{"models":[]}`, w.Body.String())
	require.Empty(t, p.paths)
}

func TestGeminiModelsUseUnifiedProtocolCapabilities(t *testing.T) {
	p := &modelsBackendStub{key: prefaceKey(), selectErr: errors.New("must not select an upstream"), result: routing.RequestableModelsResult{Models: []routing.RequestableModel{
		{ID: "gpt-5.4", Protocols: []protocol.ProtocolID{protocol.ProtocolOpenAIResponses}},
		{ID: "gemini-2.5-flash", Protocols: []protocol.ProtocolID{protocol.ProtocolGeminiGenerateContent, protocol.ProtocolOpenAIResponses}},
		{ID: "gemini-*", Protocols: []protocol.ProtocolID{protocol.ProtocolGeminiGenerateContent}},
	}}}
	p.key.ModelMapping = map[string]string{"my-model": "gemini-2.5-flash", "hidden": "gpt-5.4", "wild-*": "gemini-2.5-flash"}
	c, w := modelsContext()
	NewModelsHandler(p, &modelsCatalogStub{}).GeminiV1BetaListModels(c)
	require.Equal(t, 200, w.Code)
	require.JSONEq(t, `{"models":[{"name":"models/gemini-2.5-flash"},{"name":"models/my-model","displayName":"my-model"}]}`, w.Body.String())
	require.Empty(t, p.paths)
}

func TestGeminiModelGetRequiresRequestableModel(t *testing.T) {
	for _, name := range []string{"known", "unknown", "bad/model"} {
		t.Run(name, func(t *testing.T) {
			p := &modelsBackendStub{key: prefaceKey(), result: routing.RequestableModelsResult{Models: []routing.RequestableModel{{ID: "known", Protocols: []protocol.ProtocolID{protocol.ProtocolGeminiGenerateContent}}}}}
			c, w := modelsContext()
			c.Params = gin.Params{{Key: "model", Value: "/" + name}}
			NewModelsHandler(p, &modelsCatalogStub{}).GeminiV1BetaGetModel(c)
			want := map[string]int{"known": 200, "unknown": 404, "bad/model": 400}[name]
			require.Equal(t, want, w.Code)
			require.Empty(t, p.paths)
		})
	}
}

func TestGeminiModelsCompositePermissionsAndForcedPlatform(t *testing.T) {
	allowed := &routing.Group{ID: 1, Status: "active", AllowedProtocols: []protocol.ProtocolID{protocol.ProtocolGeminiGenerateContent}}
	disabled := &routing.Group{ID: 2, Status: "active", AllowedProtocols: []protocol.ProtocolID{protocol.ProtocolOpenAIResponses}}
	p := &modelsBackendStub{forced: "antigravity", key: &apikey.APIKey{IsComposite: true, User: &identity.User{ID: 7}, CompositeGroups: []apikey.APIKeyCompositeGroup{{GroupID: 1, Prefix: "mixed", Group: allowed}, {GroupID: 2, Prefix: "text", Group: disabled}}}, byGroup: map[int64]routing.RequestableModelsResult{1: {Models: []routing.RequestableModel{{ID: "gemini-2.5-flash", Protocols: []protocol.ProtocolID{protocol.ProtocolGeminiGenerateContent}}}}, 2: {Models: []routing.RequestableModel{{ID: "gpt-5.4", Protocols: []protocol.ProtocolID{protocol.ProtocolOpenAIResponses}}}}}}
	c, w := modelsContext()
	handler := NewModelsHandler(p, &modelsCatalogStub{})
	handler.GeminiV1BetaListModels(c)
	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), "models/mixed/gemini-2.5-flash")
	require.NotContains(t, w.Body.String(), "models/text/")
	require.Equal(t, []string{"antigravity"}, p.resolvedPlatforms)
	c, w = modelsContext()
	c.Params = gin.Params{{Key: "model", Value: "/mixed/gemini-2.5-flash"}}
	handler.GeminiV1BetaGetModel(c)
	require.Equal(t, 200, w.Code)
	p.key = &apikey.APIKey{Group: disabled}
	c, w = modelsContext()
	handler.GeminiV1BetaListModels(c)
	require.Equal(t, 403, w.Code, "强制提供商平台不能绕过分组协议权限")
}
