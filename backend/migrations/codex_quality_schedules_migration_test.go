package migrations

import (
	"os"
	"strings"
	"testing"
)

// 定时检测迁移只新增持久任务与历史表，不自动创建计划或修改账号开关。
func TestQualityScheduleMigrationIsAdditive(t *testing.T) {
	raw, err := os.ReadFile("267_codex_quality_schedules.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToLower(string(raw))
	for _, table := range []string{"codex_quality_schedules", "codex_quality_runs", "codex_quality_run_results"} {
		if !strings.Contains(sql, "create table if not exists "+table) {
			t.Fatalf("missing %s", table)
		}
	}
	if strings.Contains(sql, "update accounts") || strings.Contains(sql, "insert into codex_quality_schedules") {
		t.Fatal("migration must not schedule real tests")
	}
}
