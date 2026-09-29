// Code Assist 类型复用所属模块的定义。
package codeassist

import wire "github.com/TokenFlux/TokenRouter/internal/protocol/gemini"

type (
	LoadCodeAssistRequest  = wire.LoadCodeAssistRequest
	LoadCodeAssistMetadata = wire.LoadCodeAssistMetadata
	TierInfo               = wire.TierInfo
	LoadCodeAssistResponse = wire.LoadCodeAssistResponse
)

type (
	OnboardUserRequest  = wire.OnboardUserRequest
	OnboardUserResponse = wire.OnboardUserResponse
)
