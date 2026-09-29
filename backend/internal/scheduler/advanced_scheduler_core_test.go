package scheduler

import (
	"fmt"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/scheduler/policy"

	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/stretchr/testify/require"
)

func advancedSchedulerTestWeights() policy.ScoreWeights {
	return policy.ScoreWeights{
		Priority:  1,
		Load:      1,
		Queue:     1,
		ErrorRate: 1,
		TTFT:      1,
	}
}

func TestAdvancedSchedulerCoreUsesRuntimeFeedbackAndNeutralOptionalSignals(t *testing.T) {
	providers := []*ScoreProvider{
		{ID: 11, Priority: 1, Platform: capability.PlatformGemini},
		{ID: 12, Priority: 1, Platform: capability.PlatformGemini},
	}
	loadMap := map[int64]*ProviderLoadInfo{
		11: {ProviderID: 11, LoadRate: 20, WaitingCount: 0},
		12: {ProviderID: 12, LoadRate: 20, WaitingCount: 0},
	}
	stats := NewRuntimeStats(time.Now)
	for range 8 {
		stats.Report(11, false, nil)
		stats.Report(12, true, nil)
	}

	candidates, _ := ScoreCandidates(
		providers,
		loadMap,
		stats,
		advancedSchedulerTestWeights(),
		ScoreInput{},
		time.Now(),
	)
	require.Len(t, candidates, 2)
	require.Greater(t, candidates[1].Score, candidates[0].Score)
	require.False(t, candidates[0].HasTTFT)
	require.False(t, candidates[1].HasTTFT)
}

func TestAdvancedSchedulerCoreTreatsMissingErrorRateAsZero(t *testing.T) {
	providers := []*ScoreProvider{
		{ID: 21, Priority: 1, Platform: capability.PlatformGemini},
		{ID: 22, Priority: 1, Platform: capability.PlatformGemini},
	}
	weights := policy.ScoreWeights{
		Load:      1,
		ErrorRate: 1,
		TTFT:      1,
		Reset:     1,
	}

	// 提供商 21 缺失负载，提供商 22 的已知负载恰好处于中性位置。两者均没有
	// 错误反馈、TTFT 或窗口信息，因此错误率都按 0% 处理且最终分数一致。
	candidates, skew := ScoreCandidates(
		providers,
		map[int64]*ProviderLoadInfo{22: {ProviderID: 22, LoadRate: 50}},
		nil,
		weights,
		ScoreInput{},
		time.Now(),
	)

	require.Len(t, candidates, 2)
	require.InDelta(t, candidates[0].Score, candidates[1].Score, 0.000001)
	require.Equal(t, 0.0, skew, "只有一个已知负载样本时不能计算出偏斜")
	require.False(t, candidates[0].LoadKnown)
	require.True(t, candidates[1].LoadKnown)
	require.InDelta(t, 1.0, candidates[0].Factors.ErrorRate, 0.000001)
	require.InDelta(t, 1.0, candidates[1].Factors.ErrorRate, 0.000001)
}

func TestAdvancedSchedulerCoreTopKUsesStableOrderForMixedKnownAndUnknownLoads(t *testing.T) {
	base, _ := ScoreCandidates(
		[]*ScoreProvider{
			{ID: 1, Priority: 5},
			{ID: 2, Priority: 5},
			{ID: 3, Priority: 5},
			{ID: 4, Priority: 5},
		},
		map[int64]*ProviderLoadInfo{
			1: {ProviderID: 1, LoadRate: 99, WaitingCount: 9},
			3: {ProviderID: 3, LoadRate: 1, WaitingCount: 0},
		},
		nil,
		policy.ScoreWeights{},
		ScoreInput{},
		time.Now(),
	)
	require.Len(t, base, 4)
	require.True(t, base[0].LoadKnown)
	require.False(t, base[1].LoadKnown)
	require.True(t, base[2].LoadKnown)
	require.False(t, base[3].LoadKnown)

	// 遍历全部输入排列，确保全零权重下已知/未知负载混排不会改变同分 Top-K。
	permutation := []int{0, 1, 2, 3}
	var verifyPermutations func(int)
	verifyPermutations = func(position int) {
		if position == len(permutation) {
			candidates := make([]CandidateScore, 0, len(permutation))
			for _, index := range permutation {
				candidates = append(candidates, base[index])
			}
			topK := SelectTopK(candidates, 2)
			require.Len(t, topK, 2)
			require.Equal(t, []int64{1, 2}, []int64{topK[0].Provider.ID, topK[1].Provider.ID}, "输入顺序=%v", permutation)
			return
		}
		for index := position; index < len(permutation); index++ {
			permutation[position], permutation[index] = permutation[index], permutation[position]
			verifyPermutations(position + 1)
			permutation[position], permutation[index] = permutation[index], permutation[position]
		}
	}
	verifyPermutations(0)
}

func TestAdvancedSchedulerCoreRanksFirstFailureBelowUnknownProvider(t *testing.T) {
	failed := &ScoreProvider{ID: 31, Priority: 1, Platform: capability.PlatformGemini}
	unknown := &ScoreProvider{ID: 32, Priority: 1, Platform: capability.PlatformGemini}
	stats := NewRuntimeStats(time.Now)
	stats.Report(failed.ID, false, nil)

	candidates, _ := ScoreCandidates(
		[]*ScoreProvider{failed, unknown},
		nil,
		stats,
		policy.ScoreWeights{ErrorRate: 1},
		ScoreInput{},
		time.Now(),
	)

	require.Len(t, candidates, 2)
	require.Less(t, candidates[0].Score, candidates[1].Score)
	require.Equal(t, unknown.ID, candidates[1].Provider.ID)
}

func TestAdvancedSchedulerCoreUsesWeightedSamplingForStickyCandidate(t *testing.T) {
	candidates := []CandidateScore{
		{Provider: &ScoreProvider{ID: 1, Priority: 1}, LoadInfo: &ProviderLoadInfo{}, Score: 10},
		{Provider: &ScoreProvider{ID: 2, Priority: 1}, LoadInfo: &ProviderLoadInfo{}, Score: 9},
		{Provider: &ScoreProvider{ID: 3, Priority: 1}, LoadInfo: &ProviderLoadInfo{}, Score: 1},
	}

	var observedSticky, observedNonSticky bool
	for index := 0; index < 128; index++ {
		order := BuildSelectionOrder(candidates, ScoreInput{
			SessionHash:      fmt.Sprintf("weighted-sticky-%d", index),
			StickyWeighted:   true,
			StickyProviderID: 2,
			TopK:             2,
		})

		require.Len(t, order, 2)
		require.ElementsMatch(t, []int64{1, 2}, []int64{order[0].Provider.ID, order[1].Provider.ID})
		observedSticky = observedSticky || order[0].Provider.ID == 2
		observedNonSticky = observedNonSticky || order[0].Provider.ID == 1
	}
	require.True(t, observedSticky, "粘性加分提供商仍应有机会被抽中")
	require.True(t, observedNonSticky, "粘性加权不能退化为强制置首")
}
