package postgres

import (
	"context"
	"database/sql"

	"github.com/TokenFlux/TokenRouter/migrations"
)

// applyEmbeddedMigrations 为存储测试应用完整的发布迁移集合。
func applyEmbeddedMigrations(ctx context.Context, db *sql.DB) error {
	return ApplyMigrations(ctx, db, migrations.FS)
}
