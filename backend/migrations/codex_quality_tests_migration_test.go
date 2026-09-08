package migrations

import (
	"os"
	"strings"
	"testing"
)

// 新迁移只建立独立结果表，不重写已有账号或分组配置。
func TestCodexQualityMigrationIsAdditive(t *testing.T) {
	body, err := os.ReadFile("266_codex_quality_tests.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToLower(string(body))
	for _, want := range []string{"create table if not exists codex_quality_tests", "references accounts(id)", "lease_until", "result jsonb"} {
		if !strings.Contains(sql, want) {
			t.Fatalf("missing migration contract %q", want)
		}
	}
	if strings.Contains(sql, "update accounts") || strings.Contains(sql, "drop table") {
		t.Fatal("migration must not change existing accounts")
	}
}
