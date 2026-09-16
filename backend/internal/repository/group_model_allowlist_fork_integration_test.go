//go:build integration

package repository

import (
	"context"
	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/stretchr/testify/require"
	"testing"
)

// 独立列默认关闭，旧展示列表不迁移成权限；普通/复合 Key 都读取完整规则并获得失效事件。
func TestGroupModelAllowlistForkProjectionAndOutbox(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	client := tx.Client()
	user := mustCreateUser(t, client, &service.User{Email: "allowlist-fork@example.test"})
	groupRepo := newGroupRepositoryWithSQL(client, tx)
	group := &service.Group{Name: "allowlist-fork", Platform: service.PlatformOpenAI, Status: service.StatusActive, RateMultiplier: 1,
		ModelsListConfig: service.GroupModelsListConfig{Enabled: true, Models: []string{"display-only"}}}
	require.NoError(t, groupRepo.Create(ctx, group))
	loaded, err := groupRepo.GetByID(ctx, group.ID)
	require.NoError(t, err)
	require.False(t, loaded.ModelAllowlist.Enabled)
	require.Equal(t, group.ModelsListConfig, loaded.ModelsListConfig)
	keys := newAPIKeyRepositoryWithSQL(client, tx)
	normal := &service.APIKey{UserID: user.ID, Name: "normal", Key: "sk-allowlist-normal-test", Status: service.StatusActive, GroupID: &group.ID}
	composite := &service.APIKey{UserID: user.ID, Name: "composite", Key: "sk-allowlist-composite-test", Status: service.StatusActive, IsComposite: true}
	require.NoError(t, keys.Create(ctx, normal))
	require.NoError(t, keys.Create(ctx, composite))
	_, err = tx.ExecContext(ctx, "INSERT INTO api_key_composite_groups (api_key_id,group_id,prefix,normalized_prefix) VALUES ($1,$2,'pool','pool')", composite.ID, group.ID)
	require.NoError(t, err)
	group.ModelAllowlist = service.GroupModelAllowlist{Enabled: true, Models: []string{"gpt-*"}}
	require.NoError(t, groupRepo.Update(ctx, group))
	normalAuth, err := keys.GetByKeyForAuth(ctx, normal.Key)
	require.NoError(t, err)
	require.Equal(t, group.ModelAllowlist, normalAuth.Group.ModelAllowlist)
	compositeAuth, err := keys.GetByKeyForAuth(ctx, composite.Key)
	require.NoError(t, err)
	require.Len(t, compositeAuth.CompositeGroups, 1)
	require.Equal(t, group.ModelAllowlist, compositeAuth.CompositeGroups[0].Group.ModelAllowlist)
	require.Equal(t, group.ModelsListConfig, normalAuth.Group.ModelsListConfig)
	for _, key := range []string{normal.Key, composite.Key} {
		var count int
		err := scanSingleRow(ctx, tx, "SELECT count(*) FROM auth_cache_invalidation_outbox WHERE cache_key=encode(sha256(convert_to($1,'UTF8')),'hex')", []any{key}, &count)
		require.NoError(t, err)
		require.Positive(t, count)
	}
}
