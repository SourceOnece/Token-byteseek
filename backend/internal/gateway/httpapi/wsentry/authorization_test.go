package wsentry

import (
	"context"
	"errors"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/apikey"
	"github.com/TokenFlux/TokenRouter/internal/billing"
	"github.com/TokenFlux/TokenRouter/internal/identity"
	"github.com/stretchr/testify/require"
)

type turnAuthentication struct {
	key   *apikey.APIKey
	err   error
	input apikey.AuthenticationInput
}

func (a *turnAuthentication) GetByKey(context.Context, string) (*apikey.APIKey, error) {
	panic("unexpected cached lookup")
}

func (a *turnAuthentication) Reauthenticate(_ context.Context, _ *apikey.APIKey, input apikey.AuthenticationInput) (*apikey.APIKey, error) {
	a.input = input
	return a.key, a.err
}

// TestAuthorizeTurn 用当前快照检查资金；身份失败不得进入资金检查，也不得覆盖上一轮快照。
func TestAuthorizeTurn(t *testing.T) {
	for _, name := range []string{"allowed", "deleted", "funding_denied", "subscription_missing"} {
		t.Run(name, func(t *testing.T) {
			original := &apikey.APIKey{ID: 1, User: &identity.User{ID: 2}, BillingMode: billing.APIKeyBillingModeBalance, RateLimit5h: 10}
			current := apikey.CopyAPIKey(original)
			current.RateLimit5h = 1
			auth := &turnAuthentication{key: current}
			if name == "deleted" {
				auth.err = apikey.ErrAPIKeyNotFound
			}
			if name == "subscription_missing" {
				current.BillingMode = billing.APIKeyBillingModeSubscription
			}
			called := false
			denied := errors.New("insufficient funds")
			adapter := &openAIWSEntryAdapter{key: original, bindings: Bindings{Keys: auth, CheckFunding: func(_ context.Context, key *apikey.APIKey, sub *billing.UserSubscription, _ string, afterWait bool) error {
				called = true
				require.Same(t, current, key)
				require.Nil(t, sub)
				require.False(t, afterWait)
				if name == "funding_denied" {
					return denied
				}
				return nil
			}}}
			adapter.call.ClientIP = "192.0.2.1"
			err := adapter.AuthorizeTurn(context.Background())
			if name == "allowed" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			require.Equal(t, name == "allowed" || name == "funding_denied", called)
			require.True(t, auth.input.CheckMemberLimits)
			require.Equal(t, "192.0.2.1", auth.input.ClientIP)
			require.Equal(t, 10.0, original.RateLimit5h)
		})
	}
}

type turnSubscriptions struct {
	current *billing.UserSubscription
}

func (s turnSubscriptions) GetUsableSubscription(context.Context, int64, ...int64) (*billing.UserSubscription, bool, error) {
	return s.current, false, nil
}

func (s turnSubscriptions) GetSubscriptionForAPIKey(context.Context, int64, int64) (*billing.UserSubscription, error) {
	return s.current, nil
}

func (s turnSubscriptions) ValidateAndCheckLimits(*billing.UserSubscription) (bool, error) {
	return false, nil
}

// TestAuthorizeTurnReloadsSubscription 验证资金检查读取本轮订阅，不复用连接建立时的剩余额度。
func TestAuthorizeTurnReloadsSubscription(t *testing.T) {
	key := &apikey.APIKey{ID: 1, User: &identity.User{ID: 2}}
	old := &billing.UserSubscription{ID: 3, DailyUsageUSD: 0}
	current := &billing.UserSubscription{ID: 3, DailyUsageUSD: 10}
	denied := errors.New("subscription exhausted")
	adapter := &openAIWSEntryAdapter{key: key, subscription: old, bindings: Bindings{
		Keys: &turnAuthentication{key: key}, Subscriptions: turnSubscriptions{current: current},
		CheckFunding: func(_ context.Context, _ *apikey.APIKey, sub *billing.UserSubscription, _ string, _ bool) error {
			require.Same(t, current, sub)
			return denied
		},
	}}
	require.ErrorIs(t, adapter.AuthorizeTurn(context.Background()), denied)
	require.Same(t, old, adapter.subscription)
	require.Zero(t, old.DailyUsageUSD)
}
