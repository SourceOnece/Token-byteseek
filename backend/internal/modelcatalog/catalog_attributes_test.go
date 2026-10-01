package modelcatalog

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// 多个中继的模态可以不同；裸名只能借用唯一公共身份的属性。
const ambiguousImageCatalog = `{
	"models":{"openai/gpt-image-2.5-flare":{
		"name":"GPT Image 2.5 Flare",
		"modalities":{"input":["text","image"],"output":["image"]}
	}},
	"providers":{
		"azure":{"models":{"gpt-image-2.5-flare":{
			"canonical_model_id":"openai/gpt-image-2.5-flare",
			"modalities":{"input":["text"],"output":["image"]},
			"cost":{"input":1,"output":2}
		}}},
		"relay":{"models":{"gpt-image-2.5-flare":{
			"canonical_model_id":"openai/gpt-image-2.5-flare",
			"cost":{"input":3,"output":4}
		}}}
	}
}`

func TestLookupAttributesCanonicalFallback(t *testing.T) {
	for _, variant := range []string{"flare", "sunburst"} {
		t.Run(variant, func(t *testing.T) {
			model := "gpt-image-2.5-" + variant
			catalog, err := Parse([]byte(strings.ReplaceAll(ambiguousImageCatalog, "flare", variant)))
			require.NoError(t, err)
			attributes, found := catalog.LookupAttributes(model, nil)
			require.True(t, found)
			require.Equal(t, []string{"text", "image"}, *attributes.InputModalities)
			require.Equal(t, []string{"image"}, *attributes.OutputModalities)

			// 属性回退不得解除价格歧义，也不能让调用方改写公共快照。
			require.True(t, catalog.RequiresExact(model))
			_, found = catalog.Lookup([]string{model})
			require.False(t, found)
			(*attributes.InputModalities)[0] = "audio"
			again, found := catalog.LookupAttributes(model, nil)
			require.True(t, found)
			require.Equal(t, []string{"text", "image"}, *again.InputModalities)

			qualified, found := catalog.LookupAttributes("azure/"+model, []string{model})
			require.True(t, found)
			require.Equal(t, []string{"text"}, *qualified.InputModalities)
			price, found := catalog.Lookup([]string{"azure/" + model})
			require.True(t, found)
			require.Equal(t, 1.0, *price.Cost.Input)
			// 精确条目即使缺少属性，也不能改用公共模型或其他供应商。
			qualified, found = catalog.LookupAttributes("relay/"+model, []string{model})
			require.True(t, found)
			require.Nil(t, qualified.InputModalities)
			_, found = catalog.LookupAttributes("azure/missing", nil)
			require.False(t, found)
		})
	}
}

func TestLookupAttributesRejectsUncertainCanonical(t *testing.T) {
	for _, canonical := range []string{"", "other/model", "openai/missing"} {
		t.Run(canonical, func(t *testing.T) {
			// 只改变第一个供应商的归属，保留另一条记录和公共资料。
			body := strings.Replace(ambiguousImageCatalog,
				`"canonical_model_id":"openai/gpt-image-2.5-flare"`,
				`"canonical_model_id":"`+canonical+`"`, 1)
			catalog, err := Parse([]byte(body))
			require.NoError(t, err)
			_, found := catalog.LookupAttributes("gpt-image-2.5-flare", nil)
			require.False(t, found)
		})
	}
	// 所有来源一致但没有公共资料时，仍不能从中继挑选属性。
	body := strings.ReplaceAll(ambiguousImageCatalog,
		`"canonical_model_id":"openai/gpt-image-2.5-flare"`,
		`"canonical_model_id":"openai/missing"`)
	catalog, err := Parse([]byte(body))
	require.NoError(t, err)
	_, found := catalog.LookupAttributes("gpt-image-2.5-flare", nil)
	require.False(t, found)
}

func TestLookupAttributesKeepsOriginalEndpoint(t *testing.T) {
	body := strings.Replace(ambiguousImageCatalog, `"azure":`, `"openai":`, 1)
	catalog, err := Parse([]byte(body))
	require.NoError(t, err)
	attributes, found := catalog.LookupAttributes("gpt-image-2.5-flare", nil)
	require.True(t, found)
	require.Equal(t, []string{"text"}, *attributes.InputModalities)
}
