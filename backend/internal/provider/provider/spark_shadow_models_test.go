package provider

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultSparkShadowModelMapping(t *testing.T) {
	mapping := DefaultSparkShadowModels()

	require.Len(t, mapping, 1, "Spark 只公开完整原生型号")
	require.Equal(t, "gpt-5.3-codex-spark", mapping["gpt-5.3-codex-spark"], "恒等映射：base 映射到自身")
}

func TestSparkModelVariantsDerivedFromAliases(t *testing.T) {
	got := sparkModelVariants()
	require.ElementsMatch(t, []string{
		"gpt-5.3-codex-spark",
	}, got, "spark 变体应从 codexModelMap 派生，避免默认影子映射漂移")
}
