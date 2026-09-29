package anthropic

import (
	"strings"
)

var claudeEffortFamilies = []struct {
	family string
	levels []string
}{
	{family: "claude-sonnet-5-5", levels: []string{"low", "medium", "high", "xhigh", "max"}},
	{family: "claude-sonnet-5", levels: []string{"low", "medium", "high", "xhigh", "max"}},
	{family: "claude-opus-5-5", levels: []string{"low", "medium", "high", "xhigh", "max"}},
	{family: "claude-opus-5", levels: []string{"low", "medium", "high", "xhigh", "max"}},
	{family: "claude-fable-5", levels: []string{"low", "medium", "high", "xhigh", "max"}},
	{family: "claude-sonnet-4-6", levels: []string{"low", "medium", "high", "max"}},
	{family: "claude-opus-4-8", levels: []string{"low", "medium", "high", "xhigh", "max"}},
	{family: "claude-opus-4-7", levels: []string{"low", "medium", "high", "xhigh", "max"}},
	{family: "claude-opus-4-6", levels: []string{"low", "medium", "high", "max"}},
}

func normalizeEffortModelID(model string) string {
	id := strings.ToLower(strings.TrimSpace(model))
	id = strings.TrimPrefix(id, "models/")
	if i := strings.LastIndexByte(id, '/'); i >= 0 {
		id = strings.TrimPrefix(strings.TrimSpace(id[i+1:]), "models/")
	}
	for _, prefix := range []string{"us.", "eu.", "apac.", "jp.", "au.", "us-gov.", "global.", "anthropic."} {
		id = strings.TrimPrefix(id, prefix)
	}
	id = strings.TrimSuffix(id, "-thinking")
	id = strings.ReplaceAll(id, "claude-opus-5.5", "claude-opus-5-5")
	id = strings.ReplaceAll(id, "claude-sonnet-5.5", "claude-sonnet-5-5")
	if len(id) >= 9 && id[len(id)-9] == '-' {
		numeric := true
		for _, r := range id[len(id)-8:] {
			if r < '0' || r > '9' {
				numeric = false
				break
			}
		}
		if numeric {
			id = id[:len(id)-9]
		}
	}
	return id
}

// EffortLevelsForModel 返回 Claude 新模型支持的推理档位。
func EffortLevelsForModel(model string) []string {
	id := normalizeEffortModelID(model)
	for _, item := range claudeEffortFamilies {
		if id == item.family || strings.HasPrefix(id, item.family+"-") {
			return append([]string(nil), item.levels...)
		}
	}
	return nil
}

func IsOpus55(model string) bool   { return normalizeEffortModelID(model) == "claude-opus-5-5" }
func IsSonnet55(model string) bool { return normalizeEffortModelID(model) == "claude-sonnet-5-5" }
