package repository

import (
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCodexQualityFilterSQL(t *testing.T) {
	for _, status := range []string{"full", "degraded", "failed", "untested"} {
		table := entsql.Table("accounts")
		selector := entsql.Dialect(dialect.Postgres).Select(table.C("id")).From(table)
		qualityFilterPredicate(status)(selector)
		query, args := selector.Query()
		t.Logf("status=%s query=%s args=%v", status, query, args)
		require.Contains(t, query, "codex_quality_tests")
		if status == "untested" {
			require.Contains(t, query, "NOT IN")
		} else {
			require.Contains(t, query, "$1")
			require.NotContains(t, query, " = ?")
			require.Contains(t, args, status)
		}
	}
}
