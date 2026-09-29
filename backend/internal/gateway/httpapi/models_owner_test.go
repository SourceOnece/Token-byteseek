package httpapi

import (
	"encoding/json"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 空目录桩：让所有 ID 走本站兜底条目，只验证 owned_by 投影。
type emptyModelsCatalogStub struct{ ModelsCatalog }

func (emptyModelsCatalogStub) OpenAIModels() []OpenAIModel       { return nil }
func (emptyModelsCatalogStub) GrokModels() []GrokModel           { return nil }
func (emptyModelsCatalogStub) ClaudeModels(string) []ClaudeModel { return nil }

// 本站生成的模型条目对外只显示 ByteSeek 标识，不暴露上游项目名。
func TestSiteGeneratedModelsUseByteSeekOwner(t *testing.T) {
	handler := NewModelsHandler(&modelsBackendStub{}, emptyModelsCatalogStub{})
	writers := map[string]func(*gin.Context, []string){
		"unified":   handler.WriteUnifiedModelsList,
		"composite": handler.WriteCompositeModelsList,
	}
	for name, write := range writers {
		t.Run(name, func(t *testing.T) {
			c, w := modelsContext()
			write(c, []string{"codex-auto-review"})
			var body struct {
				Data []map[string]any `json:"data"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			require.Len(t, body.Data, 1)
			require.Equal(t, "byteseek", body.Data[0]["owned_by"])
			require.NotContains(t, w.Body.String(), "tokenrouter")
			require.NotContains(t, w.Body.String(), "token-router")
		})
	}
}
