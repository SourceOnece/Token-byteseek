package ws

import (
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"strings"
)

type ContextWindowBoundary struct {
	WindowID                           string
	Changed, PreviousResponseIDRemoved bool
}

// 明确的新窗口是新根；不清账号票据、业务代理或身份，只断开本连接旧续聊。
func NormalizeContextWindowBoundary(payload []byte, previous string) ([]byte, ContextWindowBoundary, error) {
	current := strings.TrimSpace(gjson.GetBytes(payload, "client_metadata.x-codex-window-id").String())
	if current == "" {
		current = strings.TrimSpace(gjson.Get(gjson.GetBytes(payload, "client_metadata.x-codex-turn-metadata").String(), "window_id").String())
	}
	boundary := ContextWindowBoundary{WindowID: current}
	if previous == "" || current == "" || current == previous {
		return payload, boundary, nil
	}
	boundary.Changed = true
	if !gjson.GetBytes(payload, "previous_response_id").Exists() {
		return payload, boundary, nil
	}
	next, err := sjson.DeleteBytes(payload, "previous_response_id")
	boundary.PreviousResponseIDRemoved = err == nil
	return next, boundary, err
}
