package capability

import (
	"testing"
)

// GPT-5 及以后型号复用数值代际判定，避免新型号错误透传采样参数而触发 400。
func TestIsReasoningModelCoversLaterGenerations(t *testing.T) {
	for _, tc := range []struct {
		model string
		want  bool
	}{
		{"gpt-6-astra", true},
		{"gpt-6", true},
		{"gpt-7-whatever", true},
		{"gpt-6-sol", true},
		{"gpt-5.5", true},
		{"gpt-5.2", true},
		{"gpt-5", true},
		{"GPT-6-Astra", true},
		{"  gpt-6-astra  ", true},
		{"gpt-4o", false},
		{"gpt-4.1", false},
		{"gpt-image-1", false},
		{"gpt-audio", false},
		{"claude-opus-4-6", false},
		{"gemini-3.1-pro", false},
		{"", false},
	} {
		if got := ResponsesBridgeDropsSampling(tc.model); got != tc.want {
			t.Errorf("ResponsesBridgeDropsSampling(%q) = %v, want %v", tc.model, got, tc.want)
		}
	}
}

func TestOpenAIModelGeneration(t *testing.T) {
	for _, tc := range []struct {
		model     string
		wantMajor int
		wantOK    bool
	}{
		{"gpt-6-astra", 6, true},
		{"gpt-5.5", 5, true},
		{"gpt-4o", 4, true},
		{"gpt-10-future", 10, true},
		{"gpt-image-1", 0, false},
		{"claude-opus-4-6", 0, false},
		{"", 0, false},
	} {
		major, _, ok := parseResponsesBridgeModelVersion(tc.model)
		if major != tc.wantMajor || ok != tc.wantOK {
			t.Errorf("openAIModelGeneration(%q) = (%d, %v), want (%d, %v)",
				tc.model, major, ok, tc.wantMajor, tc.wantOK)
		}
	}
}
