package repository

import (
	"fmt"
	"strings"

	entsql "entgo.io/ent/dialect/sql"
	dbpredicate "github.com/TokenFlux/TokenRouter/ent/predicate"
)

// 邮箱搜索与列表显示的回退顺序一致，只读取邮箱键，不能搜索令牌等凭据全文。
// @project-doc docs/interfaces/http_api.md#admin_account_search
func accountEmailContainsFold(search string) dbpredicate.Account {
	pattern := "%" + strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(search) + "%"
	return func(s *entsql.Selector) {
		email := fmt.Sprintf(`COALESCE(
			NULLIF(%s->>'email',''), NULLIF(%s->>'email_address',''), NULLIF(%s->>'email',''),
			(SELECT email_parent.credentials->>'email' FROM accounts AS email_parent
			 WHERE email_parent.id=%s AND email_parent.deleted_at IS NULL), '')`,
			s.C("credentials"), s.C("extra"), s.C("extra"), s.C("parent_account_id"))
		s.Where(entsql.P(func(b *entsql.Builder) {
			b.WriteString(email + " ILIKE ").Arg(pattern).WriteString(` ESCAPE E'\\'`)
		}))
	}
}
