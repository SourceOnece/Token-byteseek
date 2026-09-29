package provider

import (
	"log/slog"

	egressprovider "github.com/TokenFlux/TokenRouter/internal/egress/provider"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/google/uuid"
)

// 测试将原固定应答接入实际提供商用例与平台客户端，保留完整请求断言。
func newCodexInviteForTest(reader codexInviteResetAdminServiceStub, transport QoderTransport, token *provider.OpenAITokenSource, profiles *egressprovider.TLSProfiles, routers OpenAITokenRouterReader) *provider.CodexInviteResetService {
	factory := CodexInviteFactory{Token: token, Proxy: reader.GetProxy, Transport: transport, Profiles: profiles, Routers: routers}
	return &provider.CodexInviteResetService{Options: provider.CodexInviteResetOptions{
		Read: reader.GetProvider, Client: factory.Client, NewID: uuid.NewString, Warn: slog.Warn,
	}}
}
