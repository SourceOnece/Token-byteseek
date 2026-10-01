package openai

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// 兼容 TokenFlux 历史工具调用，不改变新会话别名，也保留 allowed_tools 同步。
func TestTokenRouterPythonToolHistoryCompatibility(t *testing.T) {
	declaration := map[string]any{"type": "function", "name": "python"}
	allowed := map[string]any{"type": "function", "name": "python"}
	history := map[string]any{"type": "function_call", "name": tokenRouterPythonToolAlias}
	body := map[string]any{
		"tools":       []any{declaration},
		"tool_choice": map[string]any{"type": "allowed_tools", "tools": []any{allowed}},
		"input":       []any{history},
	}
	reverse, changed, err := AliasOpenAIOAuthReservedToolNames(body)
	require.NoError(t, err)
	require.True(t, changed)
	require.Equal(t, tokenRouterPythonToolAlias, declaration["name"])
	require.Equal(t, tokenRouterPythonToolAlias, allowed["name"])
	require.Equal(t, "python", reverse[tokenRouterPythonToolAlias])
	require.Equal(t, "python__sub2api", AliasOpenAIOAuthReservedToolName("python"))
}

// 客户端显式同名工具不能被当成网关别名改写或误合并。
func TestTokenRouterAliasExplicitDeclarationStaysSeparate(t *testing.T) {
	decl := map[string]any{"type": "function", "name": tokenRouterPythonToolAlias}
	python := map[string]any{"type": "function", "name": "python"}
	body := map[string]any{"tools": []any{decl, python}, "input": []any{map[string]any{"type": "function_call", "name": tokenRouterPythonToolAlias}}}
	reverse, _, err := AliasOpenAIOAuthReservedToolNames(body)
	require.NoError(t, err)
	require.Equal(t, tokenRouterPythonToolAlias, decl["name"])
	require.Equal(t, CodexPythonToolAlias, python["name"])
	require.Equal(t, "python", reverse[CodexPythonToolAlias])
	require.NotContains(t, reverse, tokenRouterPythonToolAlias)
}
