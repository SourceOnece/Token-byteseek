package messageforward_test

import (
	"context"
	"strings"

	"github.com/TokenFlux/TokenRouter/internal/egress"
	"github.com/TokenFlux/TokenRouter/internal/gateway"
	gatewayhttp "github.com/TokenFlux/TokenRouter/internal/gateway/httpapi"
	gatewayadapter "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/gateway/provider/messageforward"
	gatewaytestkit "github.com/TokenFlux/TokenRouter/internal/gateway/testkit"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	provideradapter "github.com/TokenFlux/TokenRouter/internal/provider/provider"
	"github.com/TokenFlux/TokenRouter/internal/settings"
	claude "github.com/TokenFlux/TokenRouter/internal/upstream/anthropic"
)

const defaultMaxLineSize = 500 * 1024 * 1024

// 夹具投影静态测试参数，实际请求使用生产 Runtime 和 HTTP Adapter。
func newHTTPRuntimeFixture(options *messageforward.Options, deps messageforward.Dependencies, filter *egress.CompiledHeaderFilter) *gatewayhttp.MessagesExecutor {
	value := messageforward.Options{ResponseReadLimit: 128 * 1024 * 1024}
	if options != nil {
		value = *options
	}
	deps.Search = gatewayadapter.NewSearchTools(nil, nil)
	return gatewayhttp.NewMessagesExecutor(messageforward.NewRuntime(deps, value), filter)
}

func newPartialHealthFixture() *provideradapter.UpstreamHealth {
	return gatewaytestkit.NewHealthObserver(gatewaytestkit.HealthInput{Options: provider.HealthOptions{}})
}

func compileResponseHeaderFilter(options *messageforward.Options) *egress.CompiledHeaderFilter {
	if options == nil {
		return nil
	}
	return egress.CompileHeaderFilter(egress.ResponseHeaderOptions{})
}

// 设置替身只覆盖当前用例实际读取的键，其余方法保留失败行为。
type gatewayTTLSettingRepo struct {
	settings.Repository
	data map[string]string
}

func (r *gatewayTTLSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	if value, ok := r.data[key]; ok {
		return value, nil
	}
	return "", settings.ErrSettingNotFound
}

func (r *gatewayTTLSettingRepo) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string)
	for _, key := range keys {
		if value, ok := r.data[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func newRuntimeSettingsFixture(repo settings.Repository) *gateway.RuntimeSettings {
	return gateway.NewRuntimeSettings(settings.New(repo), settings.ErrSettingNotFound, func() *gateway.BetaPolicySettings {
		return gatewayadapter.GatewayBetaPolicy(claude.DefaultBetaPolicySettings())
	})
}

// 保留原夹具的字符串转义，测试请求字节不变。
func strconvQuote(value string) string {
	return `"` + strings.ReplaceAll(strings.ReplaceAll(value, `\`, `\\`), `"`, `\"`) + `"`
}
