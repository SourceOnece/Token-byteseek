package protocol

import "slices"

// CloneFallbacks 隔离每个入口的目标列表，并保留省略、空列表和有序列表的区别。
func CloneFallbacks(source map[ProtocolID][]ProtocolID) map[ProtocolID][]ProtocolID {
	if source == nil {
		return nil
	}
	result := make(map[ProtocolID][]ProtocolID, len(source))
	for key, targets := range source {
		result[key] = slices.Clone(targets)
	}
	return result
}
