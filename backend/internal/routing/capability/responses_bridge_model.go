package capability

import (
	"strconv"
	"strings"
)

// 这些型号判定保留旧文本桥接的独立语义，不扩大原生能力或复用管理员映射策略。
func ResponsesBridgeSupportsMaxEffort(model string) bool {
	return isResponsesBridgeModelAtLeastVersion(model, 5, 6)
}

func isResponsesBridgeModelAtLeastVersion(model string, minMajor, minMinor int) bool {
	major, minor, ok := parseResponsesBridgeModelVersion(model)
	if !ok {
		return false
	}
	if major != minMajor {
		return major > minMajor
	}
	return minor >= minMinor
}

func parseResponsesBridgeModelVersion(model string) (major int, minor int, ok bool) {
	normalized := normalizeResponsesBridgeModel(model)
	if normalized == "" || !strings.HasPrefix(normalized, "gpt-") {
		return 0, 0, false
	}

	rest := strings.TrimPrefix(normalized, "gpt-")
	majorEnd := 0
	for majorEnd < len(rest) && rest[majorEnd] >= '0' && rest[majorEnd] <= '9' {
		majorEnd++
	}
	if majorEnd == 0 {
		return 0, 0, false
	}

	major, err := strconv.Atoi(rest[:majorEnd])
	if err != nil {
		return 0, 0, false
	}

	minor = 0
	if majorEnd < len(rest) && rest[majorEnd] == '.' {
		minorStart := majorEnd + 1
		minorEnd := minorStart
		for minorEnd < len(rest) && rest[minorEnd] >= '0' && rest[minorEnd] <= '9' {
			minorEnd++
		}
		if minorEnd == minorStart {
			return 0, 0, false
		}
		minor, err = strconv.Atoi(rest[minorStart:minorEnd])
		if err != nil {
			return 0, 0, false
		}
	}

	return major, minor, true
}

func normalizeResponsesBridgeModel(model string) string {
	// 桥接只判断能力，不把这个投影写回请求模型。
	return strings.ToLower(LastOpenAIModelSegment(model))
}

// ResponsesBridgeDropsSampling 判断模型是否为 Responses API 下不支持 temperature/top_p 的推理模型。
// GPT-5 及后续数字代际保持同一约束，不能用字符串前缀漏掉 GPT-6。
func ResponsesBridgeDropsSampling(model string) bool {
	major, _, ok := parseResponsesBridgeModelVersion(model)
	return ok && major >= 5
}
