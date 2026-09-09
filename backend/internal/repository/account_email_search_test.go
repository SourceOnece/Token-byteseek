package repository

import (
	"context"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/TokenFlux/TokenRouter/ent"
	dbaccount "github.com/TokenFlux/TokenRouter/ent/account"
	"github.com/stretchr/testify/require"
)

func TestAccountEmailSearchBoundLiteralAndFallback(t *testing.T) {
	for _, tt := range []struct{ search, pattern string }{
		{"User@Example.com", "%User@Example.com%"},
		{`a_b%\@example.com`, `%a\_b\%\\@example.com%`},
		{"o'hara@example.com", "%o'hara@example.com%"},
	} {
		s := entsql.Dialect(dialect.Postgres).Select("id").From(entsql.Table("accounts"))
		accountEmailContainsFold(tt.search)(s)
		query, args := s.Query()
		require.Contains(t, query, `ILIKE $1 ESCAPE E'\\'`)
		require.Equal(t, []any{tt.pattern}, args)
		require.Contains(t, query, "email_parent.deleted_at IS NULL")
		require.Contains(t, query, "COALESCE(")
		require.NotContains(t, query, "access_token")
		require.NotContains(t, query, tt.search)
		t.Logf("SQL=%s", query)
	}
}

func TestAccountEmailSearchSharesCountAndPagePredicate(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	repo := newAccountRepositoryWithSQL(client, db, nil)
	q := repo.accountListFilteredQuery("openai", "oauth", "", "USER@example.com", 0, "")
	mock.ExpectQuery(`(?s)SELECT COUNT.*platform.*AND.*type.*AND.*name.*ILIKE.*OR.*COALESCE.*email_parent.*ILIKE \$4`).WithArgs("openai", "oauth", "%user@example.com%", "%USER@example.com%").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	n, err := q.Clone().Count(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, n)
	mock.ExpectQuery(`(?s)SELECT.*platform.*AND.*type.*AND.*name.*ILIKE.*OR.*COALESCE.*email_parent.*ILIKE \$4.*LIMIT.*OFFSET`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(9))
	ids, err := q.Offset(1).Limit(1).Select(dbaccount.FieldID).IDs(context.Background())
	require.NoError(t, err)
	require.Equal(t, []int64{9}, ids)
	// 空查询不增加邮箱表达式，避免无搜索时产生额外 JSON 读取。
	mock.ExpectQuery(`^SELECT COUNT\(.*FROM "accounts" WHERE "accounts"."deleted_at" IS NULL$`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	_, err = repo.accountListFilteredQuery("", "", "", "", 0, "").Count(context.Background())
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
