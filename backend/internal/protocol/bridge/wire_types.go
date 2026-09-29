package bridge

import (
	"encoding/json"

	"github.com/TokenFlux/TokenRouter/internal/protocol/anthropic"
	"github.com/TokenFlux/TokenRouter/internal/protocol/openai"
)

// 协议类型由各 wire 包唯一拥有；桥接内部复用别名，不复制编解码实现。
type (
	AnthropicRequest      = anthropic.AnthropicRequest
	AnthropicOutputConfig = anthropic.AnthropicOutputConfig
	AnthropicThinking     = anthropic.AnthropicThinking
	AnthropicMessage      = anthropic.AnthropicMessage
	AnthropicContentBlock = anthropic.AnthropicContentBlock
	AnthropicImageSource  = anthropic.AnthropicImageSource
	AnthropicTool         = anthropic.AnthropicTool
)

type AnthropicResponse = anthropic.AnthropicResponse

type (
	AnthropicUsage               = anthropic.AnthropicUsage
	AnthropicStreamEvent         = anthropic.AnthropicStreamEvent
	AnthropicDelta               = anthropic.AnthropicDelta
	ResponsesRequest             = openai.ResponsesRequest
	ResponsesReasoning           = openai.ResponsesReasoning
	ResponsesText                = openai.ResponsesText
	ResponsesInputItem           = openai.ResponsesInputItem
	ResponsesContentPart         = openai.ResponsesContentPart
	ResponsesTool                = openai.ResponsesTool
	ResponsesResponse            = openai.ResponsesResponse
	ResponsesError               = openai.ResponsesError
	ResponsesIncompleteDetails   = openai.ResponsesIncompleteDetails
	ResponsesOutput              = openai.ResponsesOutput
	WebSearchAction              = openai.WebSearchAction
	ResponsesSummary             = openai.ResponsesSummary
	ResponsesUsage               = openai.ResponsesUsage
	ResponsesInputTokensDetails  = openai.ResponsesInputTokensDetails
	ResponsesOutputTokensDetails = openai.ResponsesOutputTokensDetails
	ResponsesStreamEvent         = openai.ResponsesStreamEvent
	ChatCompletionsRequest       = openai.ChatCompletionsRequest
)

type (
	ChatMessage     = openai.ChatMessage
	ChatContentPart = openai.ChatContentPart
	ChatImageURL    = openai.ChatImageURL
)

type (
	ChatTool                = openai.ChatTool
	ChatFunction            = openai.ChatFunction
	ChatToolCall            = openai.ChatToolCall
	ChatFunctionCall        = openai.ChatFunctionCall
	ChatCompletionsResponse = openai.ChatCompletionsResponse
	ChatChoice              = openai.ChatChoice
	ChatUsage               = openai.ChatUsage
	ChatTokenDetails        = openai.ChatTokenDetails
	ChatCompletionsChunk    = openai.ChatCompletionsChunk
	ChatChunkChoice         = openai.ChatChunkChoice
	ChatDelta               = openai.ChatDelta
)

func AnthropicStopReasonPtr(s string) *string {
	return anthropic.AnthropicStopReasonPtr(s)
}

func AnthropicStopReasonString(p *string) string {
	return anthropic.AnthropicStopReasonString(p)
}

const minMaxOutputTokens = 128

func toolSearchCallArgumentsJSON(arguments string) json.RawMessage {
	return openai.ToolSearchCallArgumentsJSON(arguments)
}
