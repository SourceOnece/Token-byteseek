//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/apikey"
	"github.com/TokenFlux/TokenRouter/internal/identity"
	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/TokenFlux/TokenRouter/internal/usage"
	"github.com/TokenFlux/TokenRouter/migrations"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestResponseModelBatchRoundTrip 使用真实 PostgreSQL 检验批量 SQL 的列和参数顺序。
func TestResponseModelBatchRoundTrip(t *testing.T) {
	ctx := context.Background()
	// 批量写入会提交到共享数据库，测试结束后由夹具清理数据。
	client := testEntClient(t)
	suffix := uuid.NewString()
	user := mustCreateUser(t, client, &identity.User{Email: suffix + "@response-model.test"})
	key := mustCreateApiKey(t, client, &apikey.APIKey{UserID: user.ID, Key: suffix, Name: "model"})
	upstream := mustCreateProvider(t, client, &provider.Record{Name: suffix})
	repo := &Store{}
	keys := []string{}
	prepared := map[string]usageLogInsertPrepared{}
	for _, kind := range []string{"unknown", "same", "different"} {
		log := &usage.UsageLog{UserID: user.ID, APIKeyID: key.ID, ProviderID: upstream.ID, RequestID: suffix + kind, Model: "sent", CreatedAt: time.Now()}
		if kind != "unknown" {
			value := kind
			mismatch := kind == "different"
			log.UpstreamResponseModel = &value
			log.UpstreamModelMismatch = &mismatch
		}
		batchKey := usageLogBatchKey(log.RequestID, log.APIKeyID)
		keys = append(keys, batchKey)
		prepared[batchKey] = prepareUsageLogInsert(log)
	}
	_, states, _, err := repo.batchInsertUsageLogs(integrationDB, keys, prepared)
	require.NoError(t, err)
	for _, key := range keys {
		state := states[key]
		log, err := scanUsageLog(integrationDB.QueryRowContext(ctx, "SELECT "+usageLogSelectColumns+" FROM usage_logs WHERE id=$1", state.ID))
		require.NoError(t, err)
		want := prepared[key].args
		require.Equal(t, want[len(want)-2], nullString(log.UpstreamResponseModel))
		require.Equal(t, want[len(want)-1], log.UpstreamModelMismatch)
	}
	// 尽力批量和单条降级路径也必须传递两列，不能只验证常规批量。
	for _, path := range []string{"best-effort", "fallback"} {
		model, mismatch := "runtime-version", true
		log := &usage.UsageLog{UserID: user.ID, APIKeyID: key.ID, ProviderID: upstream.ID, RequestID: suffix + path, Model: "sent", CreatedAt: time.Now(), UpstreamResponseModel: &model, UpstreamModelMismatch: &mismatch}
		value := prepareUsageLogInsert(log)
		if path == "best-effort" {
			query, args := buildUsageLogBestEffortInsertQuery([]usageLogInsertPrepared{value})
			_, err = integrationDB.ExecContext(ctx, query, args...)
		} else {
			err = execUsageLogInsertNoResult(ctx, integrationDB, value)
		}
		require.NoError(t, err)
		loaded, err := scanUsageLog(integrationDB.QueryRowContext(ctx, "SELECT "+usageLogSelectColumns+" FROM usage_logs WHERE request_id=$1 AND api_key_id=$2", log.RequestID, key.ID))
		require.NoError(t, err)
		require.Equal(t, log.UpstreamResponseModel, loaded.UpstreamResponseModel)
		require.Equal(t, log.UpstreamModelMismatch, loaded.UpstreamModelMismatch)
	}
	// 重放不能覆盖已保存的原始声明。
	_, _, _, err = repo.batchInsertUsageLogs(integrationDB, keys, prepared)
	require.NoError(t, err)
}

// TestResponseModelMigrationPreservesHistory 在事务内还原旧列结构，确认增量迁移保留历史空值。
func (s *UsageLogRepoSuite) TestResponseModelMigrationPreservesHistory() {
	user := mustCreateUser(s.T(), s.client, &identity.User{Email: "response-history@test.local"})
	key := mustCreateApiKey(s.T(), s.client, &apikey.APIKey{UserID: user.ID, Key: "response-history", Name: "history"})
	upstream := mustCreateProvider(s.T(), s.client, &provider.Record{Name: "response-history"})
	log := s.createUsageLog(user, key, upstream, 1, 1, 0, time.Now())
	_, err := s.tx.ExecContext(s.ctx, "ALTER TABLE usage_logs DROP COLUMN upstream_response_model, DROP COLUMN upstream_model_mismatch")
	s.Require().NoError(err)
	migration, err := migrations.FS.ReadFile("285_usage_upstream_response_model.sql")
	s.Require().NoError(err)
	for range 2 {
		_, err = s.tx.ExecContext(s.ctx, string(migration))
		s.Require().NoError(err)
	}
	loaded, err := s.repo.GetByID(s.ctx, log.ID)
	s.Require().NoError(err)
	s.Nil(loaded.UpstreamResponseModel)
	s.Nil(loaded.UpstreamModelMismatch)
}
