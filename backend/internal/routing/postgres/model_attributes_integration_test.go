//go:build integration

package postgres

import (
	"context"
	"sync"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/infra/postgres"
	"github.com/TokenFlux/TokenRouter/internal/routing"
	"github.com/TokenFlux/TokenRouter/internal/testutil/postgrescontainer"
	"github.com/TokenFlux/TokenRouter/migrations"
	"github.com/stretchr/testify/require"
)

func TestModelAttributeMigrationAndConcurrentAssociation(t *testing.T) {
	db := postgrescontainer.New(t)
	ctx := context.Background()
	require.NoError(t, postgres.ApplyMigrations(ctx, db, migrations.FS))
	var groupID int64
	require.NoError(t, db.QueryRowContext(ctx, `INSERT INTO groups(name) VALUES ('attribute-test') RETURNING id`).Scan(&groupID))
	store := NewModelAttributeStore(db)
	var wg sync.WaitGroup
	errors := make(chan error, 2)
	for _, name := range []string{"first", "second"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			value := &routing.ModelAttributeConfig{Name: name, Status: "active", Rules: []routing.ModelAttributeRule{}, GroupIDs: []int64{groupID}}
			errors <- store.Save(ctx, value)
		}()
	}
	wg.Wait()
	close(errors)
	successes, conflicts := 0, 0
	for err := range errors {
		if err == nil {
			successes++
		} else {
			require.ErrorIs(t, err, routing.ErrAttributeConfigConflict)
			conflicts++
		}
	}
	require.Equal(t, 1, successes)
	require.Equal(t, 1, conflicts)
	rows, err := store.List(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	value, err := store.ForGroup(ctx, groupID)
	require.NoError(t, err)
	require.NotNil(t, value)
	value.Status = "disabled"
	require.NoError(t, store.Save(ctx, value))
	inactive, err := store.ForGroup(ctx, groupID)
	require.NoError(t, err)
	require.Nil(t, inactive)
	require.NoError(t, store.Delete(ctx, value.ID))
	var count int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM model_attribute_config_groups WHERE group_id=$1`, groupID).Scan(&count))
	require.Zero(t, count)
}
