package service

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/provider"
	"github.com/stretchr/testify/require"
)

// 逐个核心字段填入非零哨兵后往返，避免手动映射漏掉凭据、版本、母号或冷却字段。
func TestProviderRecordRoundTripPreservesCoreAndLegacyCache(t *testing.T) {
	a := &Account{}
	core := reflect.TypeOf(provider.Record{})
	legacy := reflect.ValueOf(a).Elem()
	now := time.Date(2026, 9, 29, 1, 2, 3, 456789123, time.FixedZone("TW", 8*3600))
	for i := 0; i < core.NumField(); i++ {
		name := core.Field(i).Name
		if name == "ParentProviderID" {
			name = "ParentAccountID"
		}
		field := legacy.FieldByName(name)
		require.True(t, field.IsValid(), name)
		switch field.Kind() {
		case reflect.String:
			field.SetString(name)
		case reflect.Int, reflect.Int64:
			field.SetInt(int64(i + 17))
		case reflect.Bool:
			field.SetBool(true)
		case reflect.Map:
			field.Set(reflect.ValueOf(map[string]any{"nested": map[string]any{"sentinel": name}}))
		case reflect.Slice:
			field.Set(reflect.ValueOf([]int64{7, 13}))
		case reflect.Struct:
			field.Set(reflect.ValueOf(now))
		case reflect.Pointer:
			value := reflect.New(field.Type().Elem())
			if field.Type() == reflect.TypeOf((*time.Time)(nil)) {
				value.Elem().Set(reflect.ValueOf(now))
			} else {
				switch value.Elem().Kind() {
				case reflect.String:
					value.Elem().SetString(name)
				case reflect.Int, reflect.Int64:
					value.Elem().SetInt(int64(i + 1))
				case reflect.Float64:
					value.Elem().SetFloat(0.125)
				case reflect.Struct:
					value.Elem().FieldByName("ID").SetInt(19)
				}
			}
			field.Set(value)
		default:
			t.Fatalf("uncovered field %s", name)
		}
	}
	copy := AccountFromProviderRecord(a.ProviderRecord())
	require.Equal(t, a, copy)
	// Provider默认JSON隐藏凭据，旧缓存仍必须包含凭据；类型迁移不能破坏调度摘要补全。
	raw, err := json.Marshal(copy)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"Credentials":{"nested":{"sentinel":"Credentials"}}`)
	require.Contains(t, string(raw), `"ParentAccountID":`)
	require.NotContains(t, string(raw), "ParentProviderID")
	require.Nil(t, AccountFromProviderRecord(nil))
	require.Nil(t, (*Account)(nil).ProviderRecord())
}
