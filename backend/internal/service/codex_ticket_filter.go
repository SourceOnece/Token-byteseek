package service

import (
	"context"
	"strconv"
	"strings"
)

type codexTicketFilterKey struct{}

func WithCodexTicketFilter(ctx context.Context, value string) context.Context {
	return context.WithValue(ctx, codexTicketFilterKey{}, value)
}
func CodexTicketFilter(ctx context.Context) string {
	v, _ := ctx.Value(codexTicketFilterKey{}).(string)
	return v
}
func ValidCodexTicketFilter(v string) bool {
	switch v {
	case "", "on", "off", "configured", "proxy_account", "proxy_gateway", "fixed", "rotate", "dynamic":
		return true
	}
	if strings.HasPrefix(v, "length:") {
		n, e := strconv.Atoi(strings.TrimPrefix(v, "length:"))
		return e == nil && n >= 6 && n <= 8192
	}
	return false
}
