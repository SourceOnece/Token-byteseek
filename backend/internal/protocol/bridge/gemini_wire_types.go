package bridge

import (
	"github.com/TokenFlux/TokenRouter/internal/protocol/anthropic"
	"github.com/TokenFlux/TokenRouter/internal/protocol/gemini"
)

// 兼容 wire 变体保持原字段与省略语义，桥接不复制其编解码。

type ClaudeRequest = anthropic.ClaudeRequest

type ClaudeMessage = anthropic.ClaudeMessage

type ClaudeTool = anthropic.ClaudeTool

type ContentBlock = anthropic.ContentBlock

type ClaudeResponse = anthropic.ClaudeResponse

type ClaudeContentItem = anthropic.ClaudeContentItem

type ClaudeUsage = anthropic.ClaudeUsage

type GeminiContent = gemini.GeminiContent

type GeminiPart = gemini.GeminiPart

type GeminiInlineData = gemini.GeminiInlineData

type GeminiFunctionCall = gemini.GeminiFunctionCall

type GeminiFunctionResponse = gemini.GeminiFunctionResponse

type GeminiGenerationConfig = gemini.GeminiGenerationConfig

type GeminiThinkingConfig = gemini.GeminiThinkingConfig

type GeminiToolDeclaration = gemini.GeminiToolDeclaration

type GeminiCodeExecution = gemini.GeminiCodeExecution

type GeminiFunctionDecl = gemini.GeminiFunctionDecl

type GeminiGoogleSearch = gemini.GeminiGoogleSearch

type GeminiEnhancedContent = gemini.GeminiEnhancedContent

type GeminiImageSearch = gemini.GeminiImageSearch

type GeminiResponse = gemini.GeminiResponse

type GeminiGroundingMetadata = gemini.GeminiGroundingMetadata

type GeminiGroundingChunk = gemini.GeminiGroundingChunk
