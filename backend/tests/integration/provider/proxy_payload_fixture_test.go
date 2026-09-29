package provider_test

import (
	"database/sql/driver"
	"encoding/json"
	"reflect"
)

type proxyProviderIDsPayloadMatcher struct {
	want []int64
}

func (m proxyProviderIDsPayloadMatcher) Match(value driver.Value) bool {
	raw, ok := value.([]byte)
	if !ok {
		return false
	}
	var payload struct {
		ProviderIDs []int64 `json:"provider_ids"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return false
	}
	return reflect.DeepEqual(m.want, payload.ProviderIDs)
}
