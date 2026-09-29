package httpapi

import (
	"context"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/egress"
	egressadapter "github.com/TokenFlux/TokenRouter/internal/egress/provider"
	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/stretchr/testify/require"
)

func TestTLSFingerprintProfileService_ResolveTLSProfileOpenAI(t *testing.T) {
	svc := &egressadapter.TLSProfiles{}

	openAIOAuth := &gatewayprovider.ExecutionProvider{
		Record: providercore.Record{
			LoadLocation: time.LoadLocation, Platform: capability.PlatformOpenAI,
			Type:  capability.ProviderTypeOAuth,
			Extra: map[string]any{"enable_tls_fingerprint": true},
		},
	}
	require.NotNil(t, svc.ResolveRequestTLS(gatewayprovider.ExecutionTLSSelection(openAIOAuth, nil)), "OpenAI OAuth 开启后应返回内置默认 profile")

	openAIAPIKey := &gatewayprovider.ExecutionProvider{
		Record: providercore.Record{
			LoadLocation: time.LoadLocation, Platform: capability.PlatformOpenAI,
			Type:  capability.ProviderTypeAPIKey,
			Extra: map[string]any{"enable_tls_fingerprint": true},
		},
	}
	require.Nil(t, svc.ResolveRequestTLS(gatewayprovider.ExecutionTLSSelection(openAIAPIKey, nil)), "OpenAI API Key 不应启用 TLS 指纹伪装")
}

func TestTLSFingerprintProfileService_ResolveTLSProfileQoderCosy(t *testing.T) {
	svc := &egressadapter.TLSProfiles{}

	qoderCosy := &gatewayprovider.ExecutionProvider{
		Record: providercore.Record{
			LoadLocation: time.LoadLocation, Platform: capability.PlatformQoder,
			Type:  capability.ProviderTypeCosy,
			Extra: map[string]any{"enable_tls_fingerprint": true},
		},
	}
	require.NotNil(t, svc.ResolveRequestTLS(gatewayprovider.ExecutionTLSSelection(qoderCosy, nil)), "Qoder COSY 开启后应返回内置默认 profile")

	qoderOtherType := &gatewayprovider.ExecutionProvider{
		Record: providercore.Record{
			LoadLocation: time.LoadLocation, Platform: capability.PlatformQoder,
			Type:  capability.ProviderTypeOAuth,
			Extra: map[string]any{"enable_tls_fingerprint": true},
		},
	}
	require.Nil(t, svc.ResolveRequestTLS(gatewayprovider.ExecutionTLSSelection(qoderOtherType, nil)), "非 COSY Qoder 提供商不应启用 TLS 指纹伪装")
}

func TestOpenAIGatewayService_ResolveTLSProfileRouterFallback(t *testing.T) {
	provider := &gatewayprovider.ExecutionProvider{
		Record: providercore.Record{
			LoadLocation: time.LoadLocation, Platform: capability.PlatformOpenAI,
			Type: capability.ProviderTypeOAuth,
			Extra: map[string]any{
				"enable_tls_fingerprint":     true,
				"tls_fingerprint_profile_id": int64(10),
			},
		},
	}
	profileSvc := egressadapter.NewTLSProfiles(egress.NewTLSFingerprintProfileService(&tlsProfileTestStore{profiles: []*egress.TLSFingerprintProfile{{ID: 10, Name: "fixed"}, {ID: 20, Name: "router"}}}, nil))
	profileSvc.Start()

	svc := newResponsesFixture(responsesFixtureInputs{profiles: profileSvc})

	// 路由器命中优先使用规则目标模板。
	routerProfile := svc.Requests.TLSProfile(provider, egress.TLSFingerprintRouterMatchResult{
		Matched:                 true,
		TLSFingerprintProfileID: 20,
	})
	require.NotNil(t, routerProfile)
	require.Equal(t, "router", routerProfile.Name)

	// 规则目标模板不可用时安全回退提供商固定模板。
	fallbackProfile := svc.Requests.TLSProfile(provider, egress.TLSFingerprintRouterMatchResult{
		Matched:                 true,
		TLSFingerprintProfileID: 404,
	})
	require.NotNil(t, fallbackProfile)
	require.Equal(t, "fixed", fallbackProfile.Name)
}

// tlsProfileTestStore 通过相同读取入口提供固定测试策略。
type tlsProfileTestStore struct {
	egress.TLSFingerprintProfileRepository
	profiles []*egress.TLSFingerprintProfile
}

func (s *tlsProfileTestStore) List(context.Context) ([]*egress.TLSFingerprintProfile, error) {
	return s.profiles, nil
}

type tlsRouterTestStore struct {
	egress.TLSFingerprintRouterRepository
	values []*egress.TLSFingerprintRouter
}

func (s *tlsRouterTestStore) List(context.Context) ([]*egress.TLSFingerprintRouter, error) {
	return s.values, nil
}

func newTLSFingerprintRouterTestService(routers ...*egress.TLSFingerprintRouter) *egress.TLSFingerprintRouterService {
	service := egress.NewTLSFingerprintRouterService(&tlsRouterTestStore{values: routers}, nil)
	service.Start()
	return service
}
