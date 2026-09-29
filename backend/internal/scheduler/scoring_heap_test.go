package scheduler

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAIProviderCandidateHeap_PushPopAndInvalidType(t *testing.T) {
	h := candidateHeap{}
	h.Push(CandidateScore{
		Provider: &ScoreProvider{ID: 7001},
		LoadInfo: &ProviderLoadInfo{LoadRate: 0, WaitingCount: 0},
		Score:    1.0,
	})
	require.Equal(t, 1, h.Len())
	popped, ok := h.Pop().(CandidateScore)
	require.True(t, ok)
	require.Equal(t, int64(7001), popped.Provider.ID)
	require.Equal(t, 0, h.Len())

	require.Panics(t, func() {
		h.Push("bad_element_type")
	})
}
