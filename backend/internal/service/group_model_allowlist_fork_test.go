//go:build unit

package service

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestModelAllowlistSelectionPreservesClientAlias(t *testing.T) {
	group := &Group{ModelAllowlist: GroupModelAllowlist{Enabled: true, Models: []string{"public"}}}
	ctx := WithAPIKeyModelRedirectTrace(context.Background(), NewAPIKeyModelRedirectTrace("public", "public", "internal"))
	require.NoError(t, validateGroupModelAllowlistForSelection(ctx, group, "internal"))
	require.Error(t, validateGroupModelAllowlistForSelection(context.Background(), group, "internal"))
	group.ModelAllowlist.Enabled = false
	require.NoError(t, validateGroupModelAllowlistForSelection(ctx, group, "anything"))
}

func TestModelAllowlistSnapshotIsolation(t *testing.T) {
	group := &Group{ID: 1, ModelAllowlist: GroupModelAllowlist{Enabled: true, Models: []string{"allowed"}}, ModelsListConfig: GroupModelsListConfig{Enabled: true, Models: []string{"display"}}}
	snapshot := authGroupSnapshotFromGroup(group)
	group.ModelAllowlist.Models[0] = "changed"
	require.Equal(t, []string{"allowed"}, snapshot.ModelAllowlist.Models)
	restored := groupFromAuthSnapshot(snapshot)
	restored.ModelAllowlist.Models[0] = "changed again"
	require.Equal(t, []string{"allowed"}, snapshot.ModelAllowlist.Models)
	require.Equal(t, []string{"display"}, restored.ModelsListConfig.Models)
	svc := &APIKeyService{}
	_, ok, err := svc.applyAuthCacheEntry("test-only", &APIKeyAuthCacheEntry{Snapshot: &APIKeyAuthSnapshot{Version: 37}})
	require.NoError(t, err)
	require.False(t, ok)
}
