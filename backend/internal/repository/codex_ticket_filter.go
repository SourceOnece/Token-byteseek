package repository

import (
	entsql "entgo.io/ent/dialect/sql"
	"fmt"
	dbent "github.com/TokenFlux/TokenRouter/ent"
	dbaccount "github.com/TokenFlux/TokenRouter/ent/account"
	"strconv"
	"strings"
)

// 数据库私有设置只用于静态筛选，Count和分页前生效，不将当前页状态误当全池结果。
func applyCodexTicketFilter(q *dbent.AccountQuery, filter string) {
	if filter == "" {
		return
	}
	q.Where(dbaccount.PlatformEQ("openai"), dbaccount.TypeEQ("oauth"), dbaccount.ParentAccountIDIsNil())
	q.Where(func(s *entsql.Selector) {
		s.Where(entsql.ExprP(fmt.Sprintf("LOWER(BTRIM(COALESCE(%s->>'auth_mode',''))) <> 'agentidentity'", s.C("credentials"))))
		root := "COALESCE((SELECT value::jsonb FROM settings WHERE key='codex_ticket_runtime'),'{}'::jsonb)"
		ac := fmt.Sprintf("(%s->'accounts'->(%s)::text)", root, s.C("id"))
		on := fmt.Sprintf("COALESCE(%s->>'mode','inherit') <> 'off'", ac)
		own := fmt.Sprintf("(COALESCE(%s->>'proxy_cipher','')<>'' OR (%s->'proxy_policy' IS NOT NULL AND %s->'proxy_policy'<>'null'::jsonb))", ac, ac, ac)
		mode := fmt.Sprintf("COALESCE(%s->'proxy_policy'->>'mode',CASE WHEN COALESCE(%s->>'proxy_cipher','')<>'' THEN 'fixed' END,%s->'proxy_policy'->>'mode',%s->>'selection_mode','fixed')", ac, ac, root, root)
		var expr string
		switch filter {
		case "on":
			expr = on
		case "off":
			expr = "NOT (" + on + ")"
		case "configured":
			expr = ac + " IS NOT NULL"
		case "proxy_account":
			expr = own
		case "proxy_gateway":
			expr = "NOT " + own
		case "fixed", "rotate", "dynamic":
			expr = mode + "='" + filter + "'"
		default:
			length, _ := strconv.Atoi(strings.TrimPrefix(filter, "length:"))
			expr = fmt.Sprintf("COALESCE(%s->'rules'->>'target_length',%s->>'target_length','292')='%d'", ac, root, length)
		}
		s.Where(entsql.ExprP(expr))
	})
}
