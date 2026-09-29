package httputil

import "strings"

// CanonicalAdminAccountRoute 仅统一 Provider 兼容入口的业务身份。
// 审计仍保存真实请求路径；支付 providers 及相似路径不能被错误归一。
func CanonicalAdminAccountRoute(path string) string {
	const alias = "/api/v1/admin/providers"
	if path == alias || strings.HasPrefix(path, alias+"/") {
		return "/api/v1/admin/accounts" + strings.TrimPrefix(path, alias)
	}
	return path
}
