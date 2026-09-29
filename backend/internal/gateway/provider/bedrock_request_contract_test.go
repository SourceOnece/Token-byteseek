package provider_test

import (
	"testing"
	"time"

	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	providercore "github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/TokenFlux/TokenRouter/internal/upstream/bedrock"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// 以下合同直接验证所属模块，保留原输入与断言。
// 默认别名必须生成官方请求地址，且无版本后缀的模型仍保留新版缓存能力。
func TestResolveBedrockModelID_OfficialVersionlessModels(t *testing.T) {
	t.Parallel()
	for _, model := range []string{"claude-opus-4-7", "claude-opus-4-8", "claude-opus-5", "claude-sonnet-5"} {
		t.Run(model, func(t *testing.T) {
			for _, scope := range []struct {
				name, region, prefix string
				forceGlobal          bool
			}{
				{name: "区域推理", region: "eu-west-1", prefix: "eu"},
				{name: "全局推理", region: "us-east-1", prefix: "global", forceGlobal: true},
			} {
				t.Run(scope.name, func(t *testing.T) {
					provider := &gatewayprovider.ExecutionProvider{Record: providercore.Record{LoadLocation: time.LoadLocation, Platform: capability.PlatformAnthropic, Type: capability.ProviderTypeBedrock, Credentials: map[string]any{
						"aws_region": scope.region,
					}}}
					if scope.forceGlobal {
						provider.Record.Credentials["aws_force_global"] = "true"
					}
					modelID, ok := gatewayprovider.ExecutionModelPolicy(provider).Bedrock(model)
					require.True(t, ok)
					wantID := scope.prefix + ".anthropic." + model
					require.Equal(t, wantID, modelID)
					require.Equal(t, "https://bedrock-runtime."+scope.region+".amazonaws.com/model/"+wantID+"/invoke", bedrock.BuildBedrockURL(scope.region, modelID, false))

					body, err := bedrock.PrepareBedrockRequestBody([]byte(`{"system":[{"type":"text","text":"system","cache_control":{"type":"ephemeral","ttl":"1h"}}],"messages":[{"role":"user","content":"hello"}],"max_tokens":16}`), modelID, "")
					require.NoError(t, err)
					require.Equal(t, "1h", gjson.GetBytes(body, "system.0.cache_control.ttl").String())
				})
			}
		})
	}
}
