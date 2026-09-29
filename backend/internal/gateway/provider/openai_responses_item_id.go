package provider

import (
	"fmt"
	"strings"

	"github.com/TokenFlux/TokenRouter/internal/upstream/openai"

	openaiprotocol "github.com/TokenFlux/TokenRouter/internal/protocol/openai"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func SanitizeOpenAIResponsesInputItemIDs(body []byte) ([]byte, bool, error) {
	input := gjson.GetBytes(body, "input")
	if !input.IsArray() {
		return body, false, nil
	}

	type inputItem struct {
		raw         string
		stripID     bool
		stripCallID bool
	}

	items := make([]inputItem, 0)
	input.ForEach(func(_, item gjson.Result) bool {
		parsed := inputItem{raw: item.Raw}
		if item.IsObject() {
			itemType := item.Get("type")
			id := item.Get("id")
			trimmedItemType := strings.TrimSpace(itemType.String())
			parsed.stripCallID = item.Get("call_id").Exists() && openaiprotocol.ShouldStripNonPairCallID(trimmedItemType)
			if id.Type == gjson.String {
				parsed.stripID = openai.ShouldStripOpenAIResponsesInputItemID(trimmedItemType, id.String())
			}
		}
		items = append(items, parsed)
		return true
	})
	hasSanitization := false
	for _, item := range items {
		if item.stripID || item.stripCallID {
			hasSanitization = true
			break
		}
	}
	if !hasSanitization {
		return body, false, nil
	}

	rebuiltItems := make([]string, 0, len(items))
	for index, item := range items {
		if !item.stripID && !item.stripCallID {
			rebuiltItems = append(rebuiltItems, item.raw)
			continue
		}
		itemBody := []byte(item.raw)
		if item.stripID {
			var err error
			itemBody, err = sjson.DeleteBytes(itemBody, "id")
			if err != nil {
				return nil, false, fmt.Errorf("delete input.%d.id: %w", index, err)
			}
		}
		if item.stripCallID {
			var err error
			itemBody, err = sjson.DeleteBytes(itemBody, "call_id")
			if err != nil {
				return nil, false, fmt.Errorf("delete input.%d.call_id: %w", index, err)
			}
		}
		rebuiltItems = append(rebuiltItems, string(itemBody))
	}
	return replaceOpenAIRawInput(body, input, rebuiltItems), true, nil
}
