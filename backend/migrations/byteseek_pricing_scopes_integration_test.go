//go:build integration

package migrations_test

import (
	"context"
	"database/sql"
	"io/fs"
	"testing"
	"testing/fstest"

	infra "github.com/TokenFlux/TokenRouter/internal/infra/postgres"
	"github.com/TokenFlux/TokenRouter/migrations"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

// 同名模型在不同旧平台上的金额必须各自保留，不能迁移成其中的较高价格。
func TestByteSeekPricingScopesPreservePrices(t *testing.T) {
	ctx := context.Background()
	pg, err := tcpostgres.Run(ctx, "postgres:18.1-alpine3.23", tcpostgres.WithDatabase("pricing_scopes"), tcpostgres.WithUsername("postgres"), tcpostgres.WithPassword("postgres"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pg.Terminate(ctx)) })
	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	before := fstest.MapFS{}
	entries, err := fs.ReadDir(migrations.FS, ".")
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.Name() >= "283_" { continue }
		body, err := migrations.FS.ReadFile(entry.Name())
		require.NoError(t, err)
		before[entry.Name()] = &fstest.MapFile{Data: body}
	}
	require.NoError(t, infra.ApplyMigrations(ctx, db, before))
	_, err = db.ExecContext(ctx, `
INSERT INTO groups(id,name,platform) VALUES(8001,'openai group','openai'),(8002,'anthropic group','anthropic');
INSERT INTO pricing_configs(id,name,status) VALUES(8001,'shared old prices','active');
INSERT INTO pricing_config_groups(pricing_config_id,group_id) VALUES(8001,8001),(8001,8002);
INSERT INTO pricing_config_model_pricing(pricing_config_id,platform,models,input_price,output_price)
VALUES(8001,'openai','["public-model"]',0.000001,0.000002),(8001,'anthropic','["public-model"]',0.000005,0.000006);
`)
	require.NoError(t, err)
	require.NoError(t, infra.ApplyMigrations(ctx, db, migrations.FS))
	var first, second float64
	var firstID, secondID int64
	query := `SELECT p.input_price,c.pricing_config_id FROM pricing_config_groups c JOIN pricing_config_model_pricing p ON p.pricing_config_id=c.pricing_config_id WHERE c.group_id=$1`
	require.NoError(t, db.QueryRowContext(ctx,query,8001).Scan(&first,&firstID))
	require.NoError(t, db.QueryRowContext(ctx,query,8002).Scan(&second,&secondID))
	require.Equal(t, 0.000001, first)
	require.Equal(t, 0.000005, second)
	require.NotEqual(t, firstID, secondID)
	// 再运行整个迁移集合不会覆盖迁移后的管理员改价。
	_, err = db.ExecContext(ctx,`UPDATE pricing_config_model_pricing SET input_price=0.000003 WHERE pricing_config_id=$1`,firstID)
	require.NoError(t,err)
	require.NoError(t, infra.ApplyMigrations(ctx, db, migrations.FS))
	require.NoError(t, db.QueryRowContext(ctx,query,8001).Scan(&first,&firstID))
	require.Equal(t,0.000003,first)
}
