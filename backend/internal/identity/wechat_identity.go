// 微信身份查询同时识别当前渠道键和历史渠道键。
package identity

import (
	"strings"
)

const (
	WeChatOAuthProviderKey       = "wechat-main"
	WeChatOAuthLegacyProviderKey = "wechat"
)

func WeChatCompatibleProviderKeys(providerKey string) []string {
	preferred := strings.TrimSpace(providerKey)
	if preferred == "" {
		preferred = WeChatOAuthProviderKey
	}
	keys := []string{preferred}
	if !strings.EqualFold(preferred, WeChatOAuthLegacyProviderKey) {
		keys = append(keys, WeChatOAuthLegacyProviderKey)
	}
	return keys
}
