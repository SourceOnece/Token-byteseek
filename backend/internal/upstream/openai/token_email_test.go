package openai

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTokenDisplayEmail(t *testing.T) {
	for _, tt := range []struct{ json, want string }{
		{`{"email":"one@example.invalid"}`, "one@example.invalid"},
		{`{"https://api.openai.com/profile":{"email":"two@example.invalid"}}`, "two@example.invalid"},
		{`{"email":"A <one@example.invalid>"}`, ""},
		{`{"email":"bad\n@example.invalid"}`, ""},
		{`{"sub":"user_abc"}`, ""},
	} {
		token := "header." + base64.RawURLEncoding.EncodeToString([]byte(tt.json)) + ".signature"
		require.Equal(t, tt.want, TokenDisplayEmail(token))
	}
	require.Empty(t, TokenDisplayEmail("opaque-access-token"))
}
