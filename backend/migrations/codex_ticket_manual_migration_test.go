package migrations

import (
	"os"
	"strings"
	"testing"
)

// 迁移只增加手动采集日志，不更改既有账号/票据/调度，不写任何真实任务。
func TestCodexTicketManualMigrationAdditive(t *testing.T) {
	raw, err := os.ReadFile("275_codex_ticket_manual_runs.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToLower(string(raw))
	for _, table := range []string{"codex_ticket_manual_runs", "codex_ticket_manual_events"} {
		if !strings.Contains(sql, "create table if not exists "+table) {
			t.Fatal(table)
		}
	}
	for _, forbidden := range []string{"update accounts", "alter table accounts", "drop table", "insert into"} {
		if strings.Contains(sql, forbidden) {
			t.Fatal(forbidden)
		}
	}
}
