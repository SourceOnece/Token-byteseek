package openai

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultModelsContainsCodexAutoReview(t *testing.T) {
	for _, model := range DefaultModels {
		if model.ID == "codex-auto-review" {
			return
		}
	}
	t.Fatal("默认 OpenAI 模型列表应包含 codex-auto-review")
}

func TestDefaultModelsIncludeBareGPT56Alias(t *testing.T) {
	require.Contains(t, DefaultModelIDs(), "gpt-5.6")
}

// TestDefaultModelsContainsGPT6Astra 确保默认目录及其派生 ID 列表公开 Astra。
func TestDefaultModelsContainsGPT6Astra(t *testing.T) {
	for _, model := range DefaultModels {
		if model.ID != "gpt-6-astra" {
			continue
		}
		require.Equal(t, "GPT-6 Astra", model.DisplayName)
		require.Contains(t, DefaultModelIDs(), model.ID)
		return
	}
	t.Fatal("默认 OpenAI 模型列表应包含 gpt-6-astra")
}

func TestDefaultModelsPreferConcreteGPT56SolForAccountTests(t *testing.T) {
	require.NotEmpty(t, DefaultModels)
	require.Equal(t, "gpt-5.6-sol", DefaultModels[0].ID)
}
