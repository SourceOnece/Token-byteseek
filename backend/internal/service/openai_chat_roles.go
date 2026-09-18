package service

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"
)

func requiresSystemChatRole(account *Account, targetURL string) bool {
	if account == nil || account.Type != AccountTypeAPIKey {
		return false
	}
	switch account.Platform {
	case PlatformDeepseek, PlatformKimi, PlatformZhipu:
		return true
	}
	// OpenAI 兼容账号只按实际目的地的精确域名识别，不能根据请求模型名误匹配。
	u, err := url.Parse(targetURL)
	if err != nil {
		return false
	}
	switch strings.ToLower(u.Hostname()) {
	case "api.deepseek.com", "api.kimi.com", "api.moonshot.cn", "api.moonshot.ai", "open.bigmodel.cn", "api.z.ai":
		return true
	}
	return false
}

// 选定账号与端点后才适配原生 Chat：只将 developer 改为 system，不重排或合并消息。
// 保留未知字段和数值精度，不改原正文，防止换号重试继承其它平台的改写。
func normalizeStrictChatDeveloperRoles(account *Account, targetURL string, body []byte) ([]byte, error) {
	if !requiresSystemChatRole(account, targetURL) {
		return body, nil
	}
	var root map[string]json.RawMessage
	if json.Unmarshal(body, &root) != nil || root == nil {
		return nil, errors.New("chat role normalization: invalid request object")
	}
	raw, exists := root["messages"]
	if !exists {
		return body, nil
	}
	var messages []json.RawMessage
	if json.Unmarshal(raw, &messages) != nil {
		return nil, errors.New("chat role normalization: invalid messages array")
	}
	changed := false
	for i, raw := range messages {
		var message map[string]json.RawMessage
		if json.Unmarshal(raw, &message) != nil || message == nil {
			return nil, errors.New("chat role normalization: invalid message object")
		}
		var role string
		if json.Unmarshal(message["role"], &role) != nil {
			return nil, errors.New("chat role normalization: invalid message role")
		}
		if role != "developer" {
			continue
		}
		message["role"] = json.RawMessage(`"system"`)
		updated, err := json.Marshal(message)
		if err != nil {
			return nil, errors.New("chat role normalization: cannot encode message")
		}
		messages[i] = updated
		changed = true
	}
	if !changed {
		return body, nil
	}
	updated, err := json.Marshal(messages)
	if err != nil {
		return nil, errors.New("chat role normalization: cannot encode messages")
	}
	root["messages"] = updated
	return json.Marshal(root)
}
