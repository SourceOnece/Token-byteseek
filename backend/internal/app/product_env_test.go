package app

import "testing"

// TestProductEnvCompatibility 验证调试变量继承旧值，显式关闭不被旧值覆盖。
func TestProductEnvCompatibility(t *testing.T) {
	t.Setenv("SUB2API_DEBUG_MODEL_ROUTING", "true")
	t.Setenv("TOKENROUTER_DEBUG_MODEL_ROUTING", "")
	if !selectionOptions(nil).DebugRouting {
		t.Fatal("legacy routing flag was ignored")
	}
	for _, value := range []string{"0", "false"} {
		t.Setenv("TOKENROUTER_DEBUG_MODEL_ROUTING", value)
		if selectionOptions(nil).DebugRouting {
			t.Fatalf("new value %q must disable routing debug", value)
		}
	}
}
