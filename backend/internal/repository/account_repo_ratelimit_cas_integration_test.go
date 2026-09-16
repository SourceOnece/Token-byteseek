//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAccountRepositorySetRateLimitedIfUnchanged(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	repo := newAccountRepositoryWithSQL(tx.Client(), tx, nil)

	makeAcct := func(name string) *service.Account {
		return mustCreateAccount(t, tx.Client(), &service.Account{
			Name: name, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey,
			Credentials: map[string]any{"base_url": "https://www.ollama.com", "api_key": name},
		})
	}

	acct := makeAcct("cas-match")
	base := time.Now().Add(5 * time.Second)
	require.NoError(t, repo.SetRateLimited(ctx, acct.ID, base))

	cur, err := repo.GetByID(ctx, acct.ID)
	require.NoError(t, err)
	require.NotNil(t, cur)
	require.NotNil(t, cur.RateLimitedAt)
	require.NotNil(t, cur.RateLimitResetAt)
	expUpdated, expLimited, expReset := cur.UpdatedAt, cur.RateLimitedAt, cur.RateLimitResetAt

	newReset := time.Now().Add(2 * time.Hour)
	updated, err := repo.SetRateLimitedIfUnchanged(ctx, acct.ID, expUpdated, expLimited, expReset, newReset)
	require.NoError(t, err)
	require.True(t, updated, "a write matching the observed generation must apply")

	after, err := repo.GetByID(ctx, acct.ID)
	require.NoError(t, err)
	require.NotNil(t, after.RateLimitResetAt)
	require.WithinDuration(t, newReset, *after.RateLimitResetAt, 2*time.Second)

	stale := time.Now().Add(3 * time.Hour)
	updated, err = repo.SetRateLimitedIfUnchanged(ctx, acct.ID, expUpdated, expLimited, expReset, stale)
	require.NoError(t, err)
	require.False(t, updated, "a write whose row version moved on must not apply")

	updated, err = repo.SetRateLimitedIfUnchanged(ctx, acct.ID, after.UpdatedAt, expLimited, expReset, stale)
	require.NoError(t, err)
	require.False(t, updated, "re-armed limited/reset generation must not accept the old one even with the current row version")

	clearedReset := time.Now().Add(30 * time.Minute)
	updated, err = repo.SetRateLimitedIfUnchanged(ctx, acct.ID, after.UpdatedAt, nil, nil, clearedReset)
	require.NoError(t, err)
	require.False(t, updated, "nil-limited expectation must not match a set generation")

	updated, err = repo.SetRateLimitedIfUnchanged(ctx, acct.ID, expUpdated, nil, nil, clearedReset)
	require.NoError(t, err)
	require.False(t, updated, "stale row version must not apply even with nil limited/reset")

	fresh := makeAcct("cas-fresh")
	freshCur, err := repo.GetByID(ctx, fresh.ID)
	require.NoError(t, err)
	require.Nil(t, freshCur.RateLimitedAt)
	require.Nil(t, freshCur.RateLimitResetAt)
	firstReset := time.Now().Add(time.Hour)
	updated, err = repo.SetRateLimitedIfUnchanged(ctx, fresh.ID, freshCur.UpdatedAt, nil, nil, firstReset)
	require.NoError(t, err)
	require.True(t, updated, "nil generation CAS must apply to a never-limited account")

	require.NoError(t, repo.ClearRateLimit(ctx, fresh.ID))
	afterClear, err := repo.GetByID(ctx, fresh.ID)
	require.NoError(t, err)
	updated, err = repo.SetRateLimitedIfUnchanged(ctx, fresh.ID, afterClear.UpdatedAt, nil, nil, time.Now().Add(time.Hour))
	require.NoError(t, err)
	require.True(t, updated, "cleared account with nil generation accepts a fresh write")
}
