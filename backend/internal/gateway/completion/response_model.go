package completion

import "strings"

// recordedResponseModel 以新链的原始声明为准，旧图片/定制链仍可提供兼容字段。
func recordedResponseModel(result *Result) string {
	if result == nil {
		return ""
	}
	if model := strings.TrimSpace(result.UpstreamResponseModel); model != "" {
		return model
	}
	return strings.TrimSpace(result.ResponseModel)
}

// responseModelMismatch 只比较最终出站模型和原始响应声明，不改变计费模型。
func responseModelMismatch(result *Result) *bool {
	observed := recordedResponseModel(result)
	if observed == "" {
		return nil
	}
	sent := strings.TrimSpace(result.UpstreamModel)
	if sent == "" {
		sent = strings.TrimSpace(result.Model)
	}
	mismatch := !strings.EqualFold(sent, observed)
	return &mismatch
}
