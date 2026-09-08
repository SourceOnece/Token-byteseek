package repository

import (
	entsql "entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqljson"
	"fmt"
	dbent "github.com/TokenFlux/TokenRouter/ent"
	dbaccount "github.com/TokenFlux/TokenRouter/ent/account"
)

// 筛选发生在 Count/分页前，不能只在浏览器当前页过滤。
func applyCodexQualityFilter(query *dbent.AccountQuery, status string) {
	if status == "" {
		return
	}
	query.Where(dbaccount.PlatformEQ("openai"), dbaccount.TypeIn("oauth", "apikey"), dbaccount.ParentAccountIDIsNil())
	query.Where(qualityFilterPredicate(status))
}

func qualityFilterPredicate(status string) func(*entsql.Selector) {
	return func(selector *entsql.Selector) {
		selector.Where(entsql.ExprP(fmt.Sprintf("(%s<>'oauth' OR LOWER(BTRIM(COALESCE(%s->>'auth_mode',''))) <> 'agentidentity')", selector.C("type"), selector.C("credentials"))))
		table := entsql.Table("codex_quality_tests")
		matching := entsql.Select(table.C("account_id")).From(table).Where(entsql.NotNull(table.C("result")))
		if status == "untested" {
			selector.Where(entsql.NotIn(selector.C("id"), matching))
			return
		}
		matching.Where(sqljson.ValueEQ(table.C("result"), status, sqljson.Path("status")))
		selector.Where(entsql.In(selector.C("id"), matching))
	}
}
