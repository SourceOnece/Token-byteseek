//go:build unit

package service

import (
	"context"
	"encoding/json"
	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSiteBillingModePreservesDefaultsAndInjection(t *testing.T) {
	for _, tc := range []struct {
		value   string
		enabled bool
	}{{"", true}, {"true", true}, {"false", false}} {
		t.Run(tc.value, func(t *testing.T) {
			svc := NewSettingService(&settingPublicRepoStub{values: map[string]string{SettingKeySubscriptionEnabled: tc.value, SettingBalancePayDisabled: "true"}}, &config.Config{})
			settings, err := svc.GetPublicSettings(context.Background())
			require.NoError(t, err)
			require.Equal(t, tc.enabled, settings.SubscriptionEnabled)
			require.True(t, settings.PaymentBalanceDisabled)
			payload, err := svc.GetPublicSettingsForInjection(context.Background())
			require.NoError(t, err)
			encoded, err := json.Marshal(payload)
			require.NoError(t, err)
			var decoded map[string]any
			require.NoError(t, json.Unmarshal(encoded, &decoded))
			require.Equal(t, tc.enabled, decoded["subscription_enabled"])
			require.Equal(t, true, decoded["payment_balance_disabled"])
		})
	}
}
