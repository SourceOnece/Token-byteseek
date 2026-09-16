//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreativeGroupModelAllowlistRejectsBeforeHold(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		svc := newCreativeTestService()
		groups := svc.GroupRepo.(*creativeFakeGroupRepo)
		group := groups.byID[12]
		group.ModelAllowlist = GroupModelAllowlist{Enabled: enabled, Models: []string{"different-model"}}
		groups.active = []Group{*group}
		models, err := svc.ListModels(context.Background(), 7)
		require.NoError(t, err)
		_, err = svc.CreateRun(context.Background(), testCreativeScope(7), validCreateParams(), "allowlist-test")
		if enabled {
			require.Empty(t, models.Data)
			require.ErrorIs(t, err, ErrCreativeInvalidModel)
			require.Zero(t, svc.BillingRepo.(*creativeFakeBillingRepo).reserveN)
			require.Empty(t, svc.Repo.(*creativeFakeRunRepo).createParams)
		} else {
			require.NotEmpty(t, models.Data)
			require.NoError(t, err)
		}
	}
}

func TestCreativeGroupModelAllowlistUsesPublicAlias(t *testing.T) {
	for _, platform := range []string{PlatformOpenAI, PlatformGemini, PlatformGrok} {
		t.Run(platform, func(t *testing.T) {
			upstream := map[string]string{PlatformOpenAI: "gpt-image-2", PlatformGemini: "gemini-3.1-flash-image", PlatformGrok: "grok-imagine-image-2.0"}[platform]
			group := &Group{ID: 12, Platform: platform, ModelAllowlist: GroupModelAllowlist{Enabled: true, Models: []string{"public-image"}}}
			svc := &CreativePublicService{AccountRepo: &creativeFakeAccountRepo{byGroup: map[int64][]Account{12: {{
				ID: 55, Platform: platform, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true,
				Credentials: map[string]any{"model_mapping": map[string]any{"public-image": upstream, "hidden-image": upstream}},
			}}}}}
			models, err := svc.creativeModelsForGroup(context.Background(), group)
			require.NoError(t, err)
			require.Equal(t, map[string]string{"public-image": upstream}, models)
			executor := &CreativeExecutor{groupRepo: &creativeFakeGroupRepo{byID: map[int64]*Group{12: group}}}
			_, err = executor.resolveGroupPlatform(context.Background(), 12, "public-image")
			require.NoError(t, err)
			_, err = executor.Prepare(context.Background(), CreativeRun{GroupID: 12, Model: "hidden-image"})
			require.ErrorContains(t, err, "not allowed")
			var upstreamErr *CreativeUpstreamError
			require.ErrorAs(t, err, &upstreamErr)
			require.False(t, upstreamErr.Retryable)
		})
	}
}

func TestBatchImageGroupModelAllowlistServiceBoundary(t *testing.T) {
	group := &Group{ID: 12, Platform: PlatformGemini, AllowBatchImageGeneration: true, ModelAllowlist: GroupModelAllowlist{Enabled: true, Models: []string{"public-image"}}}
	svc := &BatchImagePublicService{GroupRepo: &publicBatchImageGroupRepo{groups: map[int64]*Group{12: group}}}
	require.ErrorIs(t, svc.ensureGroupAllowsBatchImage(context.Background(), &group.ID, "gemini-3.1-flash-image"), ErrBatchImageInvalidModel)
	require.NoError(t, svc.ensureGroupAllowsBatchImage(context.Background(), &group.ID, "public-image"))
	ctx := WithAPIKeyModelRedirectTrace(context.Background(), NewAPIKeyModelRedirectTrace("public-image", "public-image", "gemini-3.1-flash-image"))
	require.NoError(t, svc.ensureGroupAllowsBatchImage(ctx, &group.ID, "gemini-3.1-flash-image"))
	group.ModelAllowlist.Enabled = false
	require.NoError(t, svc.ensureGroupAllowsBatchImage(context.Background(), &group.ID, "gemini-3.1-flash-image"))
}
