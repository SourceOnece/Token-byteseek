package forward

import (
	"encoding/json"
	protocolanthropic "github.com/TokenFlux/TokenRouter/internal/protocol/anthropic"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAnthropicToResponses_TemperatureStrippedForGPT6Astra(t *testing.T) {
	temp := 0.7
	req := &protocolanthropic.AnthropicRequest{
		Model:       "gpt-6-astra",
		MaxTokens:   1024,
		Messages:    []protocolanthropic.AnthropicMessage{{Role: "user", Content: json.RawMessage(`"Hello"`)}},
		Temperature: &temp,
		TopP:        &temp,
	}

	resp, err := AnthropicToResponses(req)
	require.NoError(t, err)
	require.Nil(t, resp.Temperature, "gpt-6-astra is reasoning-only: temperature must be stripped")
	require.Nil(t, resp.TopP, "gpt-6-astra is reasoning-only: top_p must be stripped")
}
