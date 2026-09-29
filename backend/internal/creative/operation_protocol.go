// 操作目录只决定任务允许的协议意图，候选解析仍由 routing 提供。
package creative

import (
	"slices"

	"github.com/TokenFlux/TokenRouter/internal/protocol"
)

func OperationProtocol(platform, operation string) protocol.ProtocolID {
	if platform == PlatformGemini {
		return protocol.ProtocolGeminiGenerateContent
	}
	if operation == CreativeOperationGenerate {
		return protocol.ProtocolImagesGenerations
	}
	return protocol.ProtocolImagesEdits
}

func OperationsForGroup(explicit bool, allows func(protocol.ProtocolID) bool) map[string][]string {
	out := make(map[string][]string)
	for _, platform := range []string{PlatformOpenAI, PlatformGemini, PlatformGrok} {
		operations := CreativeOperationsForPlatform(platform)
		if explicit {
			operations = slices.DeleteFunc(operations, func(operation string) bool { return !allows(OperationProtocol(platform, operation)) })
		}
		out[platform] = operations
	}
	return out
}
