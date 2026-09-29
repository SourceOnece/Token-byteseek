package capability

import (
	"fmt"
	"slices"
)

// NormalizeNativeProtocols 在显式候选投影上校验并按目录排序，空集合保持为空。
func NormalizeNativeProtocols(provider ProviderProtocols) ([]ProtocolID, error) {
	options := NativeProtocolOptions(provider.Platform, provider.Type, provider.AuthMode)
	seen := make(map[ProtocolID]bool)
	for _, protocol := range provider.Enabled {
		if !slices.Contains(options, protocol) || seen[protocol] {
			return nil, fmt.Errorf("unsupported or duplicated native protocol %q", protocol)
		}
		seen[protocol] = true
	}
	normalized := []ProtocolID{}
	for _, protocol := range options {
		if seen[protocol] {
			normalized = append(normalized, protocol)
		}
	}
	return normalized, nil
}

// ValidateProtocolFallbacks 只接受已有的一跳转换边，不执行隐式多级搜索。
func ValidateProtocolFallbacks(_ string, fallbacks map[ProtocolID][]ProtocolID) error {
	for source, targets := range fallbacks {
		if !slices.Contains(canonicalGroupClientProtocols, source) {
			return fmt.Errorf("unknown source protocol %q", source)
		}
		seen := make(map[ProtocolID]bool)
		for _, target := range targets {
			if seen[target] || !slices.Contains(AutomaticProtocolFallbackTargets(source), target) {
				return fmt.Errorf("unsupported or duplicated conversion %s -> %s", source, target)
			}
			seen[target] = true
		}
	}
	return nil
}

// NormalizeResponsesImagePolicy 保留缺省继承和四种显式状态。
func NormalizeResponsesImagePolicy(policy string) (string, error) {
	switch policy {
	case "":
		return "inherit", nil
	case "inherit", "enabled", "disabled", "block":
		return policy, nil
	default:
		return "", fmt.Errorf("invalid Responses image policy")
	}
}
