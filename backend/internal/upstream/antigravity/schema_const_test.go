package antigravity

import (
	"encoding/json"
	"github.com/TokenFlux/TokenRouter/internal/protocol/bridge"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildToolsPreservesStringConst(t *testing.T) {
	for _, tc := range []struct {
		name   string
		schema string
		want   string
	}{
		{"typed", `{"type":"string","const":"browser"}`, `{"type":"string","enum":["browser"]}`},
		{"inferred", `{"const":"browser"}`, `{"type":"string","enum":["browser"]}`},
		{"empty", `{"type":"string","const":""}`, `{"type":"string","enum":[""]}`},
		{"existing enum", `{"type":"string","const":"browser","enum":["browser","shell"]}`, `{"type":"string","enum":["browser"]}`},
		{"conflicting enum", `{"type":"string","const":"browser","enum":["shell"]}`, `{"type":"string","enum":[]}`},
		{"ordinary enum", `{"type":"string","enum":["browser","shell"]}`, `{"type":"string","enum":["browser","shell"]}`},
		{"array items", `{"type":"array","items":{"type":"string","const":"browser"}}`, `{"type":"array","items":{"type":"string","enum":["browser"]}}`},
		{"nested property named const", `{"type":"object","properties":{"const":{"const":"browser"}}}`, `{"type":"object","properties":{"const":{"type":"string","enum":["browser"]}}}`},
		{"schema metadata", `{"type":"string","const":"browser","$schema":"https://json-schema.org/draft/2020-12/schema","description":"Tool kind"}`, `{"type":"string","enum":["browser"],"description":"Tool kind"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var property map[string]any
			require.NoError(t, json.Unmarshal([]byte(tc.schema), &property))
			tools := buildTools([]ClaudeTool{{
				Name: "dispatch",
				InputSchema: map[string]any{
					"type":       "object",
					"properties": map[string]any{"action": property},
				},
			}})
			require.Len(t, tools, 1)
			require.Len(t, tools[0].FunctionDeclarations, 1)
			properties := tools[0].FunctionDeclarations[0].Parameters["properties"].(map[string]any)
			got, err := json.Marshal(properties["action"])
			require.NoError(t, err)
			require.JSONEq(t, tc.want, string(got))
		})
	}
}

// PDF 在协议转换后必须到达 Gemini inlineData，而不是静默丢弃。
func TestBuildParts_DocumentBecomesInlineData(t *testing.T) {
	content := `[{"type":"text","text":"read this"},{"type":"document","source":{"type":"base64","media_type":"application/pdf","data":"JVBERi0="}}]`
	parts, stripped, err := bridge.BuildParts(json.RawMessage(content), map[string]string{}, true)
	require.NoError(t, err)
	require.False(t, stripped)
	require.Len(t, parts, 2)
	require.NotNil(t, parts[1].InlineData)
	require.Equal(t, "application/pdf", parts[1].InlineData.MimeType)
	require.Equal(t, "JVBERi0=", parts[1].InlineData.Data)
}
