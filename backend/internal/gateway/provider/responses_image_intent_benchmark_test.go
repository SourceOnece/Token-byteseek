package provider_test

import (
	"strings"
	"testing"

	textflow "github.com/TokenFlux/TokenRouter/internal/gateway/text"

	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
)

var openAIResponsesImageIntentRoutingBenchmarkSink provider.OpenAIEndpointCapability

func BenchmarkOpenAIResponsesImageIntentRouting_LargeToolsBody(b *testing.B) {
	body := buildLargeOpenAIResponsesToolsBody(32 << 20)
	if gatewayprovider.ImageIntent().IsExplicitImageGenerationIntent("/v1/responses", "gpt-5.4", body) {
		b.Fatal("large tools body must not have explicit image intent")
	}
	platform := capability.PlatformOpenAI

	b.Run("reuse_once", func(b *testing.B) {
		b.SetBytes(int64(len(body)))
		b.ReportAllocs()
		for range b.N {
			imageIntent := gatewayprovider.ImageIntent().IsExplicitImageGenerationIntent("/v1/responses", "gpt-5.4", body)
			openAIResponsesImageIntentRoutingBenchmarkSink = textflow.ResponsesCapability(imageIntent, platform)
		}
	})

	b.Run("rescan_twice", func(b *testing.B) {
		b.SetBytes(int64(len(body)))
		b.ReportAllocs()
		for range b.N {
			imageIntent := gatewayprovider.ImageIntent().IsExplicitImageGenerationIntent("/v1/responses", "gpt-5.4", body)
			// 对照优化前路径：路由阶段会再次扫描同一份未修改的 body。
			requiredCapability := provider.OpenAIEndpointCapabilityTextGeneration
			if gatewayprovider.ImageIntent().IsExplicitImageGenerationIntent("/v1/responses", "gpt-5.4", body) && platform == capability.PlatformOpenAI {
				requiredCapability = provider.OpenAIEndpointCapabilityResponses
			}
			if imageIntent && requiredCapability != provider.OpenAIEndpointCapabilityResponses {
				b.Fatal("explicit image intent must require Responses")
			}
			openAIResponsesImageIntentRoutingBenchmarkSink = requiredCapability
		}
	})
}

func buildLargeOpenAIResponsesToolsBody(targetBytes int) []byte {
	var builder strings.Builder
	builder.Grow(targetBytes + 256)
	_, _ = builder.WriteString(`{"model":"gpt-5.4","tools":[{"type":"function","name":"search","description":"`)
	_, _ = builder.WriteString(strings.Repeat("x", targetBytes))
	_, _ = builder.WriteString(`"}],"tool_choice":"auto","input":"write code"}`)
	return []byte(builder.String())
}
