package selection

import (
	"context"
	"testing"
	"time"

	providercore "github.com/TokenFlux/TokenRouter/internal/provider"

	"github.com/TokenFlux/TokenRouter/internal/billing"
	gatewayprovider "github.com/TokenFlux/TokenRouter/internal/gateway/provider"
	"github.com/TokenFlux/TokenRouter/internal/gateway/requeststate"
	"github.com/TokenFlux/TokenRouter/internal/routing"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/TokenFlux/TokenRouter/internal/scheduler"
	"github.com/stretchr/testify/require"
)

func TestAdvancedSchedulerCoreSelectsNonOpenAIGroupAndMarksResult(t *testing.T) {
	groupID := int64(42)
	group := &routing.Group{ID: groupID, SchedulerType: routing.GroupSchedulerTypeAdvanced}
	ctx := requeststate.WithGroup(context.Background(), group)
	service := newGenericSelectionForTest(GenericDependencies{Reads: Reads{}, Shared: Shared{}}, nil)

	core, scope := service.genericSelector()
	result, selected, err := core.TryAdvanced(ctx, &groupID, "session", scope.loads([]providerWithLoad{
		{
			provider: &gatewayprovider.ExecutionProvider{Record: providercore.Record{Credentials: map[string]any{"model_whitelist": []string{"*"}}, LoadLocation: time.LoadLocation, ID: 101, Platform: capability.PlatformGemini, Priority: 1, Schedulable: true, Status: billing.StatusActive}},
			loadInfo: &scheduler.ProviderLoadInfo{ProviderID: 101, LoadRate: 0},
		},
	}))
	selection := scope.restore(result)

	require.NoError(t, err)
	require.True(t, selected)
	require.NotNil(t, selection)
	require.Equal(t, int64(101), selection.Provider.Record.ID)
	require.True(t, selection.AdvancedScheduler)

	basicCtx := requeststate.WithGroup(context.Background(), &routing.Group{ID: 43, SchedulerType: routing.GroupSchedulerTypeBasic})
	basicSelection, err := service.newSelectionResult(basicCtx, &gatewayprovider.ExecutionProvider{Record: providercore.Record{Credentials: map[string]any{"model_whitelist": []string{"*"}}, LoadLocation: time.LoadLocation, ID: 102}}, true, func() {}, nil)
	require.NoError(t, err)
	require.False(t, basicSelection.AdvancedScheduler)
}
