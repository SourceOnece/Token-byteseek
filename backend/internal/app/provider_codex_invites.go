package app

import (
	"log/slog"

	"github.com/TokenFlux/TokenRouter/internal/egress"
	egressprovider "github.com/TokenFlux/TokenRouter/internal/egress/provider"
	"github.com/TokenFlux/TokenRouter/internal/infra/httpclient"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
	"github.com/google/uuid"
)

// provideCodexInvites 直接装配提供商用例及平台端口，复用请求侧令牌与传输实例。
func provideCodexInvites(admin *provider.Admin, proxy egress.ProxyRepository, token *provider.OpenAITokenSource, transport httpclient.UpstreamTransport, profiles *egressprovider.TLSProfiles, routers provideradapter.OpenAITokenRouterReader) *provider.CodexInviteResetService {
	factory := provideradapter.CodexInviteFactory{Token: token, Proxy: proxy.GetByID, Transport: transport, Profiles: profiles, Routers: routers}
	return &provider.CodexInviteResetService{Options: provider.CodexInviteResetOptions{
		Read: admin.GetProvider, Client: factory.Client, NewID: uuid.NewString, Warn: slog.Warn,
	}}
}
