package scheduler

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSelectTopKOpenAICandidates(t *testing.T) {
	candidates := []CandidateScore{
		{
			Provider: &ScoreProvider{ID: 11, Priority: 2},
			LoadInfo: &ProviderLoadInfo{LoadRate: 10, WaitingCount: 1},
			Score:    10.0,
		},
		{
			Provider: &ScoreProvider{ID: 12, Priority: 1},
			LoadInfo: &ProviderLoadInfo{LoadRate: 20, WaitingCount: 1},
			Score:    9.5,
		},
		{
			Provider: &ScoreProvider{ID: 13, Priority: 1},
			LoadInfo: &ProviderLoadInfo{LoadRate: 30, WaitingCount: 0},
			Score:    10.0,
		},
		{
			Provider: &ScoreProvider{ID: 14, Priority: 0},
			LoadInfo: &ProviderLoadInfo{LoadRate: 40, WaitingCount: 0},
			Score:    8.0,
		},
	}

	top2 := SelectTopK(candidates, 2)
	require.Len(t, top2, 2)
	require.Equal(t, int64(13), top2[0].Provider.ID)
	require.Equal(t, int64(11), top2[1].Provider.ID)

	topAll := SelectTopK(candidates, 8)
	require.Len(t, topAll, len(candidates))
	require.Equal(t, int64(13), topAll[0].Provider.ID)
	require.Equal(t, int64(11), topAll[1].Provider.ID)
	require.Equal(t, int64(12), topAll[2].Provider.ID)
	require.Equal(t, int64(14), topAll[3].Provider.ID)
}

func TestBuildOpenAIWeightedSelectionOrder_DeterministicBySessionSeed(t *testing.T) {
	candidates := []CandidateScore{
		{
			Provider: &ScoreProvider{ID: 101},
			LoadInfo: &ProviderLoadInfo{LoadRate: 10, WaitingCount: 0},
			Score:    4.2,
		},
		{
			Provider: &ScoreProvider{ID: 102},
			LoadInfo: &ProviderLoadInfo{LoadRate: 30, WaitingCount: 1},
			Score:    3.5,
		},
		{
			Provider: &ScoreProvider{ID: 103},
			LoadInfo: &ProviderLoadInfo{LoadRate: 50, WaitingCount: 2},
			Score:    2.1,
		},
	}
	req := ScoreInput{
		GroupID:        scoreGroupIDForTest(99),
		SessionHash:    "session_seed_fixed",
		RequestedModel: "gpt-5.1",
	}

	first := BuildWeightedSelectionOrder(candidates, req)
	second := BuildWeightedSelectionOrder(candidates, req)
	require.Len(t, first, len(candidates))
	require.Len(t, second, len(candidates))
	for i := range first {
		require.Equal(t, first[i].Provider.ID, second[i].Provider.ID)
	}
}

func TestDeriveOpenAISelectionSeed_NoAffinityAddsEntropy(t *testing.T) {
	req := ScoreInput{
		RequestedModel: "gpt-5.1",
	}
	seed1 := SelectionSeed(req)
	time.Sleep(1 * time.Millisecond)
	seed2 := SelectionSeed(req)
	require.NotZero(t, seed1)
	require.NotZero(t, seed2)
	require.NotEqual(t, seed1, seed2)
}

func TestBuildOpenAIWeightedSelectionOrder_HandlesInvalidScores(t *testing.T) {
	candidates := []CandidateScore{
		{
			Provider: &ScoreProvider{ID: 901},
			LoadInfo: &ProviderLoadInfo{LoadRate: 5, WaitingCount: 0},
			Score:    math.NaN(),
		},
		{
			Provider: &ScoreProvider{ID: 902},
			LoadInfo: &ProviderLoadInfo{LoadRate: 5, WaitingCount: 0},
			Score:    math.Inf(1),
		},
		{
			Provider: &ScoreProvider{ID: 903},
			LoadInfo: &ProviderLoadInfo{LoadRate: 5, WaitingCount: 0},
			Score:    -1,
		},
	}
	req := ScoreInput{
		SessionHash: "seed_invalid_scores",
	}

	order := BuildWeightedSelectionOrder(candidates, req)
	require.Len(t, order, len(candidates))
	seen := map[int64]struct{}{}
	for _, item := range order {
		seen[item.Provider.ID] = struct{}{}
	}
	require.Len(t, seen, len(candidates))
}

func TestOpenAISelectionRNG_SeedZeroStillWorks(t *testing.T) {
	rng := NewSelectionRNG(0)
	v1 := rng.NextUint64()
	v2 := rng.NextUint64()
	require.NotEqual(t, v1, v2)
	require.GreaterOrEqual(t, rng.NextFloat64(), 0.0)
	require.Less(t, rng.NextFloat64(), 1.0)
}

// 仅构造原种子测试的分组值。
func scoreGroupIDForTest(value int64) *int64 { return &value }
