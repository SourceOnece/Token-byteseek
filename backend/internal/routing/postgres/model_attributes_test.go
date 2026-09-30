package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/TokenFlux/TokenRouter/internal/routing"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestAttributeAssociationConflictRollsBackEntireConfig(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	store := NewModelAttributeStore(db)
	config := &routing.ModelAttributeConfig{Name: "new", Status: "active", GroupIDs: []int64{7}, Rules: []routing.ModelAttributeRule{}}
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO model_attribute_configs").WithArgs("new", "", "active", "[]").WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(5, time.Now(), time.Now()))
	mock.ExpectExec("DELETE FROM model_attribute_config_groups").WithArgs(int64(5)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO model_attribute_config_groups").WithArgs(int64(5), int64(7)).WillReturnError(&pq.Error{Code: "23505"})
	mock.ExpectRollback()
	require.ErrorIs(t, store.Save(context.Background(), config), routing.ErrAttributeConfigConflict)
	require.NoError(t, mock.ExpectationsWereMet())
}
