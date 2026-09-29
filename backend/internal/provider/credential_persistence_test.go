package provider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type credentialStoreStub struct {
	calls   int
	updates int
	value   *Record
}

func (s *credentialStoreStub) Update(_ context.Context, v *Record) error {
	s.calls++
	s.value = v
	return nil
}
func (s *credentialStoreStub) UpdateCredentials(_ context.Context, _ int64, _ map[string]any) error {
	s.updates++
	return nil
}

func TestPersistCredentialsProtectsShadowAndUsesNarrowUpdater(t *testing.T) {
	r := &Record{ID: 7, Credentials: map[string]any{"old": true}}
	s := &credentialStoreStub{}
	ok, err := PersistCredentials(context.Background(), s, r, map[string]any{"access_token": "synthetic"}, nil)
	require.True(t, ok)
	require.NoError(t, err)
	require.Equal(t, 1, s.updates)
	require.Zero(t, s.calls)
	require.Equal(t, "synthetic", r.Credentials["access_token"])

	parent := int64(3)
	shadow := &Record{ID: 8, ParentProviderID: &parent}
	warned := false
	ok, err = PersistCredentials(context.Background(), s, shadow, map[string]any{"access_token": "must-not-store"}, func(string, ...any) { warned = true })
	require.False(t, ok)
	require.NoError(t, err)
	require.True(t, warned)
	require.Equal(t, 1, s.updates)
	require.Nil(t, shadow.Credentials)
}
