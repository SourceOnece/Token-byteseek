package service

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// deepSeekChatReasoningPlaceholderText 是 Chat Completions 侧 thinking-mode
// 占位明文。必须是单个空格：DeepSeek 拒绝空串，非空即可通过；LiteLLM 同样注入
// 单个空格。
const deepSeekChatReasoningPlaceholderText = " "

// targetsDeepSeekAPIHost 报告该账号实际上游是否是 DeepSeek 官方 API。
// platform=deepseek 直接命中；platform=openai 但 base_url 指向
// api.deepseek.com 的映射账号同样命中（Codex 把 GPT 模型名映射到 DeepSeek
// 时的典型接入方式）。
func targetsDeepSeekAPIHost(account *Account) bool {
	if account == nil {
		return false
	}
	if account.Platform == PlatformDeepseek {
		return true
	}
	u, err := url.Parse(strings.TrimSpace(account.GetOpenAIBaseURL()))
	if err != nil {
		return false
	}
	ds, err := url.Parse(DefaultDeepseekBaseURL)
	if err != nil {
		return false
	}
	return strings.EqualFold(u.Hostname(), ds.Hostname())
}

// ensureDeepSeekChatReasoningPlaceholders 给缺 reasoning_content 的 assistant
// 消息补单个空格占位。DeepSeek thinking mode 要求历史里每条产生过思维的
// assistant 消息都回传该字段，否则 400
// "The `reasoning_content` in the thinking mode must be passed back to the API"。
//
// 桥接会从 summary / 缓存回注真实明文；这里只填仍为空的缺口，不覆盖已有内容。
// 非 DeepSeek 上游原样返回（字节不变）。
func ensureDeepSeekChatReasoningPlaceholders(account *Account, body []byte) []byte {
	if !targetsDeepSeekAPIHost(account) {
		return body
	}
	messages := gjson.GetBytes(body, "messages")
	if !messages.IsArray() {
		return body
	}
	updated := body
	changed := false
	for i, msg := range messages.Array() {
		if strings.TrimSpace(msg.Get("role").String()) != "assistant" {
			continue
		}
		if msg.Get("reasoning_content").String() != "" {
			continue
		}
		next, err := sjson.SetBytes(updated, "messages."+strconv.Itoa(i)+".reasoning_content", deepSeekChatReasoningPlaceholderText)
		if err != nil {
			return body
		}
		updated = next
		changed = true
	}
	if !changed {
		return body
	}
	return updated
}
