package modelcatalog

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOfflineCatalog(t *testing.T) {
	body, err := Offline()
	require.NoError(t, err)
	catalog, err := Parse(body)
	require.NoError(t, err)
	entry, ok := catalog.Lookup([]string{"claude-sonnet-4-5"})
	require.True(t, ok)
	require.Equal(t, "anthropic", entry.Provider)
	require.Equal(t, 3.0, *entry.Cost.Input)
	require.NotNil(t, entry.Attributes.Context)
}

func TestCatalogIdentityAndUnpricedAttributes(t *testing.T) {
	catalog, err := Parse([]byte(`{"models":{"origin/m":{"name":"Original"}},"providers":{"relay":{"models":{"m":{"name":"Relay","cost":{"input":9,"output":9}},"unpriced":{"name":"No price","reasoning":false}}},"origin":{"models":{"m":{"name":"Original","cost":{"input":0,"output":2}}}},"other":{"models":{"unpriced":{"name":"Other"}}}}}`))
	require.NoError(t, err)
	original, ok := catalog.Lookup([]string{"m"})
	require.True(t, ok)
	require.Equal(t, 0.0, *original.Cost.Input)
	relay, ok := catalog.Lookup([]string{"relay/m"})
	require.True(t, ok)
	require.Equal(t, 9.0, *relay.Cost.Input)
	_, ambiguous := catalog.Lookup([]string{"unpriced"})
	require.False(t, ambiguous)
	noPrice, ok := catalog.Lookup([]string{"relay/unpriced"})
	require.True(t, ok)
	require.Nil(t, noPrice.Cost.Input)
	require.NotNil(t, noPrice.Attributes.Reasoning)
	require.False(t, *noPrice.Attributes.Reasoning)
}

func TestAttributeMergeAndCommon(t *testing.T) {
	yes, no := true, false
	context, smaller := 100, 50
	modalities, empty := []string{"text", "image", "pdf"}, []string{}
	base := Attributes{Context: &context, Reasoning: &yes, InputModalities: &modalities}
	patched := Merge(base, Attributes{Reasoning: &no, InputModalities: &empty})
	require.Equal(t, 100, *patched.Context)
	require.False(t, *patched.Reasoning)
	require.NotNil(t, patched.InputModalities)
	require.Empty(t, *patched.InputModalities)
	common, different := Common([]Attributes{base, {Context: &smaller, Reasoning: &no}})
	require.True(t, different)
	require.Equal(t, 50, *common.Context)
	require.False(t, *common.Reasoning)
	require.Nil(t, common.InputModalities)
}
