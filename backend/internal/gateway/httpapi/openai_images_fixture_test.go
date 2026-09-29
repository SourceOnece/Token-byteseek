package httpapi

import (
	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/infra/httpclient"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
)

// imagesFixtureInputs 只提供图片执行实际使用的传输、提供商存储和输出预算。
type imagesFixtureInputs struct {
	observer                       *provideradapter.UpstreamHealth
	transport                      httpclient.UpstreamTransport
	store                          gatewayprovider.ExecutionProviderStore
	allowHTTP                      bool
	ImageStreamDataIntervalTimeout int
	ImageStreamKeepaliveInterval   int
}

func newImagesFixture(v imagesFixtureInputs) *OpenAIImagesExecutor {
	auxiliary := newAuxiliaryFixture(auxiliaryFixtureInputs{transport: v.transport, store: v.store, allowHTTP: v.allowHTTP, observer: v.observer})
	auxiliary.Output.Options.ImageStreamDataIntervalTimeout = v.ImageStreamDataIntervalTimeout
	auxiliary.Output.Options.ImageStreamKeepaliveInterval = v.ImageStreamKeepaliveInterval
	return &OpenAIImagesExecutor{Requests: auxiliary.Requests, Output: auxiliary.Output, Cooldown: &provideradapter.ImageToolCooldown{Store: v.store}}
}
