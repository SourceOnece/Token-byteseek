package bridge

import wire "github.com/TokenFlux/TokenRouter/internal/protocol/openai"

// 兼容转换先校验显式禁用推理，防止转换时被默认档位覆盖。
func validateGPT61ChatEffort(req *ChatCompletionsRequest) error {
	return wire.ValidateGPT61SolEffort(req.Model, req.ReasoningEffort)
}

func validateGPT61AnthropicEffort(req *AnthropicRequest) error {
	if req.OutputConfig != nil {
		if err := wire.ValidateGPT61SolEffort(req.Model, req.OutputConfig.Effort); err != nil {
			return err
		}
	}
	if req.Thinking != nil && req.Thinking.Type == "disabled" {
		return wire.ValidateGPT61SolEffort(req.Model, "none")
	}
	return nil
}
