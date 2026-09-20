package service

import (
	"context"
	"strconv"
	"strings"
	"sync"
)

type codexTicketFilterKey struct{}
type codexTicketMatchedIDsKey struct{}

// 仅同一个管理员请求复用匹配ID，不跨请求缓存账号凭据或实时票据状态。
type codexTicketFilterRequest struct {
	value   string
	mu      sync.Mutex
	matches map[string][]int64
}

func WithCodexTicketFilter(ctx context.Context, value string) context.Context {
	return context.WithValue(ctx, codexTicketFilterKey{}, &codexTicketFilterRequest{value: value, matches: map[string][]int64{}})
}
func CodexTicketFilter(ctx context.Context) string {
	v, _ := ctx.Value(codexTicketFilterKey{}).(*codexTicketFilterRequest)
	if v == nil {
		return ""
	}
	return v.value
}
func ValidCodexTicketFilter(v string) bool {
	_, _, ok := ParseCodexTicketFilter(v)
	return ok
}

// 旧length仍表示配置值；新actual_length只表示左侧观察值，可与一种静态条件组合。
func ParseCodexTicketFilter(v string) (static string, actualLength int, valid bool) {
	if strings.Contains(v, ",") {
		parts := strings.Split(v, ",")
		if len(parts) != 2 || parts[0] == "" || !validCodexTicketStaticFilter(parts[0]) {
			return "", 0, false
		}
		_, actual, ok := ParseCodexTicketFilter(parts[1])
		return parts[0], actual, ok && actual > 0
	}
	if validCodexTicketStaticFilter(v) {
		return v, 0, true
	}
	for _, prefix := range []string{"length:", "actual_length:"} {
		if !strings.HasPrefix(v, prefix) {
			continue
		}
		raw := strings.TrimPrefix(v, prefix)
		n, err := strconv.Atoi(raw)
		if err != nil || n < 6 || n > 8192 || (prefix == "actual_length:" && raw != strconv.Itoa(n)) {
			return "", 0, false
		}
		if prefix == "length:" {
			return v, 0, true
		}
		return "", n, true
	}
	return "", 0, false
}

func validCodexTicketStaticFilter(v string) bool {
	switch v {
	case "", "on", "off", "configured", "proxy_account", "proxy_gateway", "fixed", "rotate", "dynamic":
		return true
	}
	return false
}

// 仓储必须在Count/Offset前应用解析后的ID，nil表示未解析或无匹配，绝不放宽为全量。
func CodexTicketMatchedIDs(ctx context.Context) []int64 {
	ids, _ := ctx.Value(codexTicketMatchedIDsKey{}).([]int64)
	return ids
}
