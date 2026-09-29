package postgres_test

import (
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/routing"
	routingpostgres "github.com/TokenFlux/TokenRouter/internal/routing/postgres"

	dbent "github.com/TokenFlux/TokenRouter/ent"
	"github.com/stretchr/testify/require"
)

func TestGroupEntityToService_PreservesImageGenerationControls(t *testing.T) {
	group := &dbent.Group{
		ID:   1,
		Name: "openai-images",

		Status:               routing.StatusActive,
		RateMultiplier:       1,
		AllowImageGeneration: true,
	}

	got := routingpostgres.GroupFromEnt(group)
	require.NotNil(t, got)
	require.True(t, got.AllowImageGeneration)
}
