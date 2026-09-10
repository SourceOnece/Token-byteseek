//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/config"
	"github.com/stretchr/testify/require"
)

// 通过现有设置仓储替身检验动态读取，不发送实际邀请邮件。
type teamInvitationSettingsRepo struct {
	settingUpdateRepoStub
	readErr error
}

func (r *teamInvitationSettingsRepo) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	if r.readErr != nil {
		return nil, r.readErr
	}
	out := map[string]string{}
	for _, k := range keys {
		if v, ok := r.values[k]; ok {
			out[k] = v
		}
	}
	return out, nil
}

func TestTeamInvitationSettingsDefaultParseAndWrite(t *testing.T) {
	for _, values := range []map[string]string{nil, {SettingKeyTeamInvitationCooldownSeconds: "0", SettingKeyTeamInvitationHourlyLimit: "invalid"}} {
		r := &teamInvitationSettingsRepo{settingUpdateRepoStub: settingUpdateRepoStub{values: values}}
		s := NewSettingService(r, &config.Config{})
		limits, err := s.GetTeamInvitationRateLimits(context.Background())
		require.NoError(t, err)
		require.Equal(t, DefaultTeamInvitationRateLimits(), limits)
		all := s.parseSettings(values)
		require.Equal(t, 60, all.TeamInvitationCooldownSeconds)
		require.Equal(t, 20, all.TeamInvitationHourlyLimit)
	}
	r := &teamInvitationSettingsRepo{}
	s := NewSettingService(r, &config.Config{})
	settings := s.parseSettings(nil)
	settings.TeamInvitationCooldownSeconds = 10
	settings.TeamInvitationHourlyLimit = 100
	require.NoError(t, s.UpdateSettings(context.Background(), settings))
	require.Equal(t, "10", r.updates[SettingKeyTeamInvitationCooldownSeconds])
	require.Equal(t, "100", r.updates[SettingKeyTeamInvitationHourlyLimit])
	limiter := &fakeTeamInvitationLimiter{allowed: true}
	team := NewTeamService(nil, nil, nil, nil, limiter, s, nil)
	require.NoError(t, team.checkInvitationRate(context.Background(), 1, "member@example.test"))
	require.Equal(t, TeamInvitationRateLimits{10, 100}, limiter.limits)
	r.values[SettingKeyTeamInvitationHourlyLimit] = "200"
	require.NoError(t, team.checkInvitationRate(context.Background(), 1, "member@example.test"))
	require.Equal(t, 200, limiter.limits.HourlyLimit)
	r.readErr = errors.New("storage unavailable")
	require.ErrorIs(t, team.checkInvitationRate(context.Background(), 1, "member@example.test"), ErrTeamInvitationUnavailable)
}
