package provider

import (
	"testing"
)

func TestNormalizeProviderTestMode(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "", want: ProviderTestModeDefault},
		{input: "default", want: ProviderTestModeDefault},
		{input: " compact ", want: ProviderTestModeCompact},
		{input: "COMPACT", want: ProviderTestModeCompact},
		{input: " legacy_compact ", want: ProviderTestModeLegacyCompact},
		{input: "unknown", want: ProviderTestModeDefault},
	}

	for _, tt := range tests {
		if got := NormalizeProviderTestMode(tt.input); got != tt.want {
			t.Fatalf("normalizeProviderTestMode(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestResolveProviderTestModeAndType(t *testing.T) {
	tests := []struct {
		name      string
		mode      string
		testTypes []string
		wantMode  string
		wantType  string
		explicit  bool
	}{
		{name: "explicit image", mode: "default", testTypes: []string{"image"}, wantMode: ProviderTestModeDefault, wantType: ProviderTestTypeImage, explicit: true},
		{name: "explicit text", mode: "default", testTypes: []string{"text"}, wantMode: ProviderTestModeDefault, wantType: ProviderTestTypeText, explicit: true},
		{name: "legacy compact", mode: "compact", wantMode: ProviderTestModeCompact, wantType: "", explicit: false},
		{name: "mode alias", mode: "image", wantMode: ProviderTestModeDefault, wantType: ProviderTestTypeImage, explicit: true},
		{name: "swapped new call", mode: "image", testTypes: []string{"compact"}, wantMode: ProviderTestModeCompact, wantType: ProviderTestTypeImage, explicit: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode, testType, explicit := ResolveProviderTestModeAndType(tt.mode, tt.testTypes...)
			if mode != tt.wantMode || testType != tt.wantType || explicit != tt.explicit {
				t.Fatalf("resolveProviderTestModeAndType(%q, %#v) = (%q, %q, %v), want (%q, %q, %v)", tt.mode, tt.testTypes, mode, testType, explicit, tt.wantMode, tt.wantType, tt.explicit)
			}
		})
	}
}
