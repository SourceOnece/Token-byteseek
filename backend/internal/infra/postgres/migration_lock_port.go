package postgres

import (
	"context"
	"database/sql"
)

// 备份与迁移共用会话锁，防止导出过程中改表造成不一致快照。
func AcquireMigrationLock(ctx context.Context, conn *sql.Conn) error {
	return pgAdvisoryLock(ctx, conn)
}

func ReleaseMigrationLock(ctx context.Context, conn *sql.Conn) error {
	return pgAdvisoryUnlock(ctx, conn)
}
