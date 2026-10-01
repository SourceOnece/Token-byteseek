package modelcatalog

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestOfflineEmbeddingOrigin 验证实际内嵌目录中缺少 canonical 关联的原厂模型。
func TestOfflineEmbeddingOrigin(t *testing.T) {
	body, err := Offline()
	require.NoError(t, err)
	catalog, err := Parse(body)
	require.NoError(t, err)
	for _, model := range []string{"text-embedding-3-small", "text-embedding-3-large", "text-embedding-ada-002"} {
		t.Run(model, func(t *testing.T) {
			entry, found := catalog.Lookup([]string{model})
			require.True(t, found)
			require.True(t, entry.FirstParty)
			require.Equal(t, "openai", entry.Provider)
			require.False(t, catalog.Ambiguous[model])
			qualified, found := catalog.Lookup([]string{"openai/" + model})
			require.True(t, found)
			require.Equal(t, qualified.Cost, entry.Cost)
		})
	}
}

func TestKnownOriginWithoutCanonicalAndExplicitForeignOrigin(t *testing.T) {
	catalog, err := Parse([]byte(`{
		"providers": {
			"openai": {"models": {
				"embedding": {"cost":{"input":1,"output":0}},
				"foreign": {"canonical_model_id":"author/foreign","cost":{"input":9,"output":9}}
			}},
			"azure": {"models":{"embedding":{"cost":{"input":2,"output":0}}}},
			"author": {"models":{"foreign":{"canonical_model_id":"author/foreign","cost":{"input":3,"output":4}}}},
			"relay-a": {"models":{"unknown":{"cost":{"input":5,"output":6}}}},
			"relay-b": {"models":{"unknown":{"cost":{"input":7,"output":8}}}}
		}
	}`))
	require.NoError(t, err)
	entry, found := catalog.Lookup([]string{"embedding"})
	require.True(t, found)
	require.Equal(t, "openai", entry.Provider)
	entry, found = catalog.Lookup([]string{"azure/embedding"})
	require.True(t, found)
	require.False(t, entry.FirstParty)
	require.Equal(t, 2.0, *entry.Cost.Input)
	entry, found = catalog.Lookup([]string{"foreign"})
	require.True(t, found)
	require.Equal(t, "author", entry.Provider)
	require.True(t, catalog.Ambiguous["unknown"])
}

func TestMistralOriginRespectsForeignCanonical(t *testing.T) {
	catalog, err := Parse([]byte(`{"providers":{
		"mistral":{"models":{
			"devstral-latest":{"cost":{"input":0.4,"output":2}},
			"glm-test":{"canonical_model_id":"zhipuai/glm-test","cost":{"input":9,"output":9}}
		}},
		"requesty":{"models":{"devstral-latest":{"cost":{"input":0.44,"output":2.2}}}},
		"zai":{"models":{"glm-test":{"canonical_model_id":"zhipuai/glm-test","cost":{"input":1,"output":2}}}}
	}}`))
	require.NoError(t, err)
	entry, found := catalog.Lookup([]string{"devstral-latest"})
	require.True(t, found)
	require.Equal(t, "mistral", entry.Provider)
	entry, found = catalog.Lookup([]string{"glm-test"})
	require.True(t, found)
	require.Equal(t, "zai", entry.Provider)
	entry, found = catalog.Lookup([]string{"mistral/glm-test"})
	require.True(t, found)
	require.False(t, entry.FirstParty)
}
