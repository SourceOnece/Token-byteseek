package provider

import (
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/routing"
	"github.com/TokenFlux/TokenRouter/internal/routing/capability"
	"github.com/stretchr/testify/require"
)

func TestSanitizeGroupMessagesDispatchFieldsPreservesExplicitConfig(t *testing.T) {
	t.Parallel()

	group := &routing.Group{
		AllowedProtocols:      []capability.ProtocolID{capability.ProtocolAnthropicMessages},
		AllowMessagesDispatch: true,
		DefaultMappedModel:    "gpt-5.6-sol",
	}

	routing.SanitizeGroupMessagesDispatchFields(group)

	require.True(t, group.AllowMessagesDispatch)
	require.Equal(t, "gpt-5.6-sol", group.DefaultMappedModel)
}
