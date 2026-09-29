package provider

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/egress"
	"github.com/stretchr/testify/require"
)

func TestRecordSchedulingAndBillingKeepAccountSemantics(t *testing.T) {
	zero := 0.0
	now := time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)
	r := &Record{ID: 7, Status: StatusActive, Schedulable: true, RateMultiplier: &zero, Concurrency: 3}
	require.True(t, r.SchedulingWindowOpen(now))
	require.Zero(t, r.BillingRateMultiplier())
	require.Equal(t, 3, r.EffectiveLoadFactor())

	until := now.Add(time.Minute)
	r.OverloadUntil = &until
	require.False(t, r.SchedulingWindowOpen(now))
	require.Equal(t, "provider (id=7)", r.String())
	require.NotContains(t, r.String(), "secret")
	require.Equal(t, "provider <nil>", (*Record)(nil).String())
}

// 内部Record不用于缓存/导出，禁止因新命名迁移让凭据经默认序列化进入日志。
func TestRecordKeepsSecretsOutOfGenericJSONAndFormatting(t *testing.T) {
	parent := int64(4)
	r := &Record{ID: 9, ParentProviderID: &parent,
		Credentials: map[string]any{"access_token": "canary-token"}, Extra: map[string]any{"private": "canary-extra"},
		Proxy: &egress.Proxy{ID: 3, Host: "proxy.example", Port: 8080, Password: "canary-password"}}
	raw, err := json.Marshal(r)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "canary-")
	for _, format := range []string{"%v", "%+v", "%#v"} {
		require.Equal(t, "provider (id=9)", fmt.Sprintf(format, r))
	}
}
